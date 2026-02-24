package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/pkg/crypto"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

// ── test environment ─────────────────────────────────────────────────

type testEnv struct {
	router *gin.Engine
	cfg    *config.Config
	hub    *ws.Hub
	token  string // JWT access token for admin

	// seeded IDs
	tenantID uuid.UUID
	userID   uuid.UUID
	roleID   uuid.UUID

	// repos (for direct access in tests)
	userRepo        *memUserRepo
	tenantRepo      *memTenantRepo
	roleRepo        *memRoleRepo
	serverRepo      *memServerRepo
	agentRepo       *memAgentRepo
	auditLogRepo    *memAuditLogRepo
	osProfileRepo   *memOSProfileRepo
	diskLayoutRepo  *memDiskLayoutRepo
	scriptRepo      *memScriptRepo
	installTaskRepo *memInstallTaskRepo
	inventoryRepo   *memInventoryRepo
	ipPoolRepo      *memIPPoolRepo
	ipAddressRepo   *memIPAddressRepo
	switchRepo      *memSwitchRepo
	switchPortRepo  *memSwitchPortRepo
}

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:          "test-jwt-secret-key-for-integration-test-32-chars",
			AccessTokenTTL:  60,
			RefreshTokenTTL: 168,
		},
		Crypto: config.CryptoConfig{
			EncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		Server: config.ServerConfig{Mode: "test"},
	}
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	cfg := testConfig()

	// Repos
	userRepo := newMemUserRepo()
	tenantRepo := newMemTenantRepo()
	roleRepo := newMemRoleRepo()
	serverRepo := newMemServerRepo()
	agentRepo := newMemAgentRepo()
	auditLogRepo := newMemAuditLogRepo()
	osProfileRepo := newMemOSProfileRepo()
	diskLayoutRepo := newMemDiskLayoutRepo()
	scriptRepo := newMemScriptRepo()
	installTaskRepo := newMemInstallTaskRepo()
	inventoryRepo := newMemInventoryRepo()
	ipPoolRepo := newMemIPPoolRepo()
	ipAddressRepo := newMemIPAddressRepo()
	switchRepo := newMemSwitchRepo()
	switchPortRepo := newMemSwitchPortRepo()

	// WebSocket Hub
	hub := ws.NewHub(logger)
	go hub.Run()

	// Services
	authService := service.NewAuthService(userRepo, roleRepo, tenantRepo, cfg)
	serverService := service.NewServerService(serverRepo, cfg)
	agentService := service.NewAgentService(agentRepo)
	userService := service.NewUserService(userRepo, roleRepo)
	roleService := service.NewRoleService(roleRepo)
	tenantService := service.NewTenantService(tenantRepo)
	kvmService := service.NewKVMService(serverRepo, hub, cfg, logger)
	ipmiService := service.NewIPMIService(serverRepo, hub, cfg, logger)
	osProfileService := service.NewOSProfileService(osProfileRepo)
	diskLayoutService := service.NewDiskLayoutService(diskLayoutRepo)
	scriptService := service.NewScriptService(scriptRepo)
	reinstallService := service.NewReinstallService(serverRepo, osProfileRepo, diskLayoutRepo, scriptRepo, installTaskRepo, hub, cfg, logger)
	switchService := service.NewSwitchService(switchRepo, switchPortRepo, hub, logger)
	bandwidthService := service.NewBandwidthService(switchRepo, switchPortRepo, hub, logger)
	inventoryService := service.NewInventoryService(inventoryRepo, serverRepo, hub, logger)
	ipService := service.NewIPService(ipPoolRepo, ipAddressRepo)

	// Event handlers
	hub.OnEvent(ws.ActionAgentHeartbeat, func(agentID uuid.UUID, msg *ws.Message) {
		agentService.UpdateLastSeen(nil, agentID)
	})
	hub.OnEvent(ws.ActionPXEStatus, reinstallService.HandlePXEStatusEvent)
	hub.OnEvent(ws.ActionSNMPData, bandwidthService.HandleSNMPDataEvent)
	hub.OnEvent(ws.ActionInventoryResult, inventoryService.HandleInventoryResultEvent)

	// Router
	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api/v1")

	// Public routes (no rate limit in test)
	authHandler := NewAuthHandler(authService)
	authHandler.RegisterRoutes(api)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg.JWT.Secret))
	protected.Use(middleware.AuditLog(auditLogRepo, logger))

	authHandler.RegisterProtectedRoutes(protected)

	serverHandler := NewServerHandler(serverService)
	serverHandler.RegisterRoutes(protected)

	agentHandler := NewAgentHandler(agentService, hub)
	agentHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermAgentManage)))

	userHandler := NewUserHandler(userService)
	userHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermUserManage)))

	roleHandler := NewRoleHandler(roleService)
	roleHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermRoleManage)))

	tenantHandler := NewTenantHandler(tenantService)
	tenantHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermTenantManage)))

	ipmiHandler := NewIPMIHandler(ipmiService)
	ipmiHandler.RegisterPowerRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermServerPower)))
	ipmiHandler.RegisterSensorRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermIPMISensors)))

	osProfileHandler := NewOSProfileHandler(osProfileService)
	osProfileHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermOSProfileManage)))

	diskLayoutHandler := NewDiskLayoutHandler(diskLayoutService)
	diskLayoutHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermDiskLayoutManage)))

	scriptHandler := NewScriptHandler(scriptService)
	scriptHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermScriptManage)))

	reinstallHandler := NewReinstallHandler(reinstallService)
	reinstallHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermOSReinstall)))

	switchHandler := NewSwitchHandler(switchService)
	switchHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermSwitchManage)))

	bandwidthHandler := NewBandwidthHandler(bandwidthService)
	bandwidthHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermBandwidthView)))

	inventoryHandler := NewInventoryHandler(inventoryService)
	inventoryHandler.RegisterRoutes(
		protected.Group("", middleware.RequirePermission(roleRepo, domain.PermInventoryView)),
		protected.Group("", middleware.RequirePermission(roleRepo, domain.PermInventoryScan)),
	)

	ipHandler := NewIPHandler(ipService)
	ipHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermIPManage)))

	auditHandler := NewAuditHandler(auditLogRepo)
	auditHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermAuditView)))

	kvmHandler := NewKVMHandler(kvmService, logger)
	kvmHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermIPMIKVM)))
	kvmHandler.RegisterPublicRoutes(r)

	// Agent WebSocket endpoint
	r.GET("/api/v1/agents/ws", func(c *gin.Context) {
		HandleAgentWebSocket(c, hub, agentRepo, logger)
	})

	// Seed data
	tenantID := uuid.New()
	roleID := uuid.New()
	userID := uuid.New()

	tenantRepo.Create(nil, &domain.Tenant{
		ID:   tenantID,
		Name: "Test Datacenter",
		Slug: "test-dc",
	})

	roleRepo.Create(nil, &domain.Role{
		ID:          roleID,
		TenantID:    &tenantID,
		Name:        "Super Admin",
		Permissions: []string{"*"},
		IsSystem:    true,
	})

	pwHash, _ := crypto.HashPassword("Admin123!")
	userRepo.Create(nil, &domain.User{
		ID:           userID,
		TenantID:     tenantID,
		Email:        "admin@test.com",
		PasswordHash: pwHash,
		Name:         "Admin",
		RoleID:       &roleID,
		IsActive:     true,
	})

	// Generate JWT token
	token, err := crypto.GenerateAccessToken(
		userID.String(), tenantID.String(), roleID.String(),
		cfg.JWT.Secret, cfg.JWT.AccessTokenTTL,
	)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	return &testEnv{
		router:          r,
		cfg:             cfg,
		hub:             hub,
		token:           token,
		tenantID:        tenantID,
		userID:          userID,
		roleID:          roleID,
		userRepo:        userRepo,
		tenantRepo:      tenantRepo,
		roleRepo:        roleRepo,
		serverRepo:      serverRepo,
		agentRepo:       agentRepo,
		auditLogRepo:    auditLogRepo,
		osProfileRepo:   osProfileRepo,
		diskLayoutRepo:  diskLayoutRepo,
		scriptRepo:      scriptRepo,
		installTaskRepo: installTaskRepo,
		inventoryRepo:   inventoryRepo,
		ipPoolRepo:      ipPoolRepo,
		ipAddressRepo:   ipAddressRepo,
		switchRepo:      switchRepo,
		switchPortRepo:  switchPortRepo,
	}
}

// ── helper functions ─────────────────────────────────────────────────

func doRequest(router *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func mustParseResponse(t *testing.T, w *httptest.ResponseRecorder) domain.APIResponse {
	t.Helper()
	var resp domain.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v (body: %s)", err, w.Body.String())
	}
	return resp
}

func extractMap(t *testing.T, resp domain.APIResponse) map[string]any {
	t.Helper()
	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal data map: %v", err)
	}
	return m
}

func assertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Fatalf("expected status %d, got %d (body: %s)", expected, w.Code, w.Body.String())
	}
}

func assertSuccess(t *testing.T, resp domain.APIResponse) {
	t.Helper()
	if !resp.Success {
		t.Fatalf("expected success=true, got error: %s", resp.Error)
	}
}

// ── mock agent connection ────────────────────────────────────────────

// connectMockAgent injects a fake agent into the hub that responds to all requests.
// Uses a gorilla/websocket test server pair.
func connectMockAgent(t *testing.T, env *testEnv, agentID uuid.UUID) {
	t.Helper()

	upgrader := gorillaws.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Create a test WebSocket server — the "panel side" of the connection
	mockAgentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Register this connection in the hub as an agent
		agentConn := ws.NewAgentConnection(agentID, conn, env.hub, zap.NewNop())
		env.hub.Register(agentConn)
		go agentConn.WritePump()
		go agentConn.ReadPump()
	}))
	t.Cleanup(mockAgentServer.Close)

	// Connect to the test server to trigger the upgrade — the "agent side"
	wsURL := "ws" + strings.TrimPrefix(mockAgentServer.URL, "http")
	dialer := gorillaws.Dialer{}
	clientConn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial mock agent server: %v", err)
	}
	t.Cleanup(func() { clientConn.Close() })

	// Run mock agent loop: read requests from hub, send canned responses back
	go func() {
		for {
			_, data, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			var msg ws.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			if msg.Type != ws.TypeRequest {
				continue
			}
			payload := mockPayloadFor(msg.Action)
			resp := ws.NewResponse(msg.ID, payload, "")
			respData, _ := json.Marshal(resp)
			clientConn.WriteMessage(gorillaws.TextMessage, respData)
		}
	}()

	// Wait for hub registration
	time.Sleep(50 * time.Millisecond)
}

func mockPayloadFor(action string) any {
	switch action {
	case "ipmi.power.status":
		return map[string]any{"status": "on"}
	case "ipmi.power.on", "ipmi.power.off", "ipmi.power.reset", "ipmi.power.cycle":
		return map[string]any{"output": "Chassis Power Control: Up/Down"}
	case "ipmi.sensors":
		return map[string]any{"sensors": []map[string]any{
			{"name": "CPU Temp", "type": "temperature", "value": 45.0, "unit": "C"},
			{"name": "Fan 1", "type": "fan", "value": 3200.0, "unit": "RPM"},
		}}
	case "inventory.scan":
		return map[string]any{
			"cpu":    map[string]any{"model": "Intel Xeon E-2288G", "cores": 8},
			"memory": map[string]any{"total_mb": 32768},
		}
	case "switch.provision", "switch.status":
		return map[string]any{"admin_status": "up", "oper_status": "up", "speed_mbps": 1000}
	case "pxe.prepare", "pxe.cleanup":
		return map[string]any{"status": "ok"}
	case "ipmi.kvm.start":
		return map[string]any{"session_id": uuid.New().String(), "port": 5900}
	case "ipmi.kvm.stop":
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

// ══════════════════════════════════════════════════════════════════════
// INTEGRATION TEST — Full Server Lifecycle
// ══════════════════════════════════════════════════════════════════════

func TestServerLifecycle(t *testing.T) {
	env := setupTestEnv(t)

	var (
		serverID    uuid.UUID
		agentID     uuid.UUID
		switchID    uuid.UUID
		portID      uuid.UUID
		poolID      uuid.UUID
		osProfileID uuid.UUID
		layoutID    uuid.UUID
	)

	// ── Phase A: Authentication ──────────────────────────────────────

	t.Run("Phase_A_Auth", func(t *testing.T) {
		t.Run("Login", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/auth/login", "", map[string]string{
				"email":    "admin@test.com",
				"password": "Admin123!",
			})
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			if data["access_token"] == nil || data["access_token"] == "" {
				t.Fatal("expected access_token in response")
			}
			env.token = data["access_token"].(string)
		})

		t.Run("GetCurrentUser", func(t *testing.T) {
			w := doRequest(env.router, "GET", "/api/v1/auth/me", env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			if data["email"] != "admin@test.com" {
				t.Errorf("expected admin@test.com, got %v", data["email"])
			}
		})

		t.Run("CreateAgent", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/agents", env.token, map[string]any{
				"name":         "DC-Tokyo-01",
				"location":     "Tokyo, Japan",
				"capabilities": []string{"ipmi", "pxe", "kvm", "snmp"},
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			agentMap := data["agent"].(map[string]any)
			agentID = uuid.MustParse(agentMap["id"].(string))
			t.Logf("Agent created: %s", agentID)
		})

		t.Run("ListAgents", func(t *testing.T) {
			w := doRequest(env.router, "GET", "/api/v1/agents?page=1&page_size=20", env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})
	})

	// Connect mock agent at parent test level so it persists across all phases
	connectMockAgent(t, env, agentID)
	if !env.hub.IsAgentOnline(agentID) {
		t.Fatal("agent should be online after connection")
	}

	// ── Phase B: Server Provisioning ─────────────────────────────────

	t.Run("Phase_B_ServerProvisioning", func(t *testing.T) {
		t.Run("CreateServer", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/servers", env.token, map[string]any{
				"hostname":   "web-prod-01",
				"label":      "Production Web Server",
				"primary_ip": "10.0.1.100",
				"ipmi_ip":    "10.0.0.100",
				"ipmi_user":  "ADMIN",
				"ipmi_pass":  "secret123",
				"agent_id":   agentID.String(),
				"tags":       []string{"web", "production"},
				"notes":      "Main web server",
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			serverID = uuid.MustParse(data["id"].(string))
			t.Logf("Server created: %s", serverID)

			// Verify IPMI credentials are cleared from response
			if data["ipmi_user"] != nil && data["ipmi_user"] != "" {
				t.Error("ipmi_user should be empty in response")
			}
		})

		t.Run("ListServers", func(t *testing.T) {
			w := doRequest(env.router, "GET", "/api/v1/servers?page=1&page_size=20", env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			total := data["total"].(float64)
			if total < 1 {
				t.Errorf("expected at least 1 server, got %v", total)
			}
		})

		t.Run("GetServer", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/servers/%s", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			if data["hostname"] != "web-prod-01" {
				t.Errorf("expected hostname web-prod-01, got %v", data["hostname"])
			}
		})

		t.Run("UpdateServer", func(t *testing.T) {
			newLabel := "Updated Web Server"
			w := doRequest(env.router, "PUT", fmt.Sprintf("/api/v1/servers/%s", serverID), env.token, map[string]any{
				"label": newLabel,
			})
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			if data["label"] != newLabel {
				t.Errorf("expected label %q, got %v", newLabel, data["label"])
			}
		})
	})

	// ── Phase C: Power Management (IPMI) ─────────────────────────────

	t.Run("Phase_C_PowerManagement", func(t *testing.T) {
		t.Run("GetPowerStatus", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/servers/%s/power", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("PowerOff", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/servers/%s/power", serverID), env.token, map[string]string{
				"action": "off",
			})
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("PowerOn", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/servers/%s/power", serverID), env.token, map[string]string{
				"action": "on",
			})
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("PowerCycle", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/servers/%s/power", serverID), env.token, map[string]string{
				"action": "cycle",
			})
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})
	})

	// ── Phase D: Hardware Discovery ──────────────────────────────────

	t.Run("Phase_D_HardwareDiscovery", func(t *testing.T) {
		t.Run("ReadSensors", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/servers/%s/sensors", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("TriggerInventoryScan", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/servers/%s/inventory/scan", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			// Wait for async processing
			time.Sleep(200 * time.Millisecond)
		})

		t.Run("ViewInventory", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/servers/%s/inventory", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})
	})

	// ── Phase E: Network Configuration ───────────────────────────────

	t.Run("Phase_E_NetworkConfig", func(t *testing.T) {
		t.Run("CreateSwitch", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/switches", env.token, map[string]any{
				"name":           "TOR-Switch-01",
				"ip":             "10.0.0.1",
				"vendor":         "cisco_ios",
				"model":          "Nexus 9300",
				"snmp_community": "public",
				"snmp_version":   "v2c",
				"ssh_user":       "admin",
				"ssh_pass":       "switchpass",
				"ssh_port":       22,
				"agent_id":       agentID.String(),
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			switchID = uuid.MustParse(data["id"].(string))
			t.Logf("Switch created: %s", switchID)
		})

		t.Run("CreateSwitchPort", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/switches/%s/ports", switchID), env.token, map[string]any{
				"port_index":   1,
				"port_name":    "Ethernet1/1",
				"speed_mbps":   10000,
				"vlan_id":      100,
				"admin_status": "up",
				"server_id":    serverID.String(),
				"description":  "web-prod-01 uplink",
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			portID = uuid.MustParse(data["id"].(string))
			t.Logf("Port created: %s", portID)
		})

		t.Run("ProvisionPort", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/switches/%s/ports/%s/provision", switchID, portID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("GetPortStatus", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/switches/%s/ports/%s/status", switchID, portID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})
	})

	// ── Phase F: IP Address Assignment ───────────────────────────────

	t.Run("Phase_F_IPAddressAssignment", func(t *testing.T) {
		t.Run("CreateIPPool", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/ip-pools", env.token, map[string]any{
				"network":     "10.0.1.0/24",
				"gateway":     "10.0.1.1",
				"description": "Production VLAN",
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			poolID = uuid.MustParse(data["id"].(string))
			t.Logf("IP Pool created: %s", poolID)
		})

		t.Run("CreateAddresses", func(t *testing.T) {
			for _, addr := range []string{"10.0.1.10", "10.0.1.11", "10.0.1.12"} {
				w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/ip-pools/%s/addresses", poolID), env.token, map[string]any{
					"address": addr,
					"status":  "available",
				})
				assertStatus(t, w, http.StatusCreated)
				resp := mustParseResponse(t, w)
				assertSuccess(t, resp)
			}
		})

		t.Run("ListAddresses", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/ip-pools/%s/addresses", poolID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("AutoAssignIP", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/ip-pools/%s/assign", poolID), env.token, map[string]any{
				"server_id": serverID.String(),
			})
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			if data["status"] != "assigned" {
				t.Errorf("expected assigned, got %v", data["status"])
			}
		})
	})

	// ── Phase G: OS Installation ─────────────────────────────────────

	t.Run("Phase_G_OSInstallation", func(t *testing.T) {
		t.Run("CreateOSProfile", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/os-profiles", env.token, map[string]any{
				"name":          "Ubuntu 22.04 LTS",
				"os_family":     "ubuntu",
				"version":       "22.04",
				"arch":          "amd64",
				"kernel_url":    "http://archive.ubuntu.com/ubuntu/dists/jammy/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/linux",
				"initrd_url":    "http://archive.ubuntu.com/ubuntu/dists/jammy/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/initrd.gz",
				"boot_args":     "auto=true priority=critical",
				"template_type": "preseed",
				"template":      "d-i debian-installer/locale string en_US\n",
				"is_active":     true,
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			osProfileID = uuid.MustParse(data["id"].(string))
		})

		t.Run("CreateDiskLayout", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/disk-layouts", env.token, map[string]any{
				"name":        "Standard 2-partition",
				"description": "Boot + Root",
				"layout": map[string]any{
					"partitions": []map[string]string{
						{"mount": "/boot", "size": "1G", "fs": "ext4"},
						{"mount": "/", "size": "100%FREE", "fs": "ext4"},
					},
				},
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			layoutID = uuid.MustParse(data["id"].(string))
		})

		t.Run("StartReinstall", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/servers/%s/reinstall", serverID), env.token, map[string]any{
				"os_profile_id":  osProfileID.String(),
				"disk_layout_id": layoutID.String(),
				"raid_level":     "auto",
				"root_password":  "SecurePass123!",
				"ssh_keys":       []string{"ssh-ed25519 AAAA... admin@test"},
			})
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("CheckReinstallStatus", func(t *testing.T) {
			time.Sleep(200 * time.Millisecond) // wait for async goroutine
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/servers/%s/reinstall/status", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})
	})

	// ── Phase H: KVM Console ─────────────────────────────────────────

	t.Run("Phase_H_KVMConsole", func(t *testing.T) {
		t.Run("StartKVM", func(t *testing.T) {
			w := doRequest(env.router, "POST", fmt.Sprintf("/api/v1/servers/%s/kvm", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			if data["session_id"] == nil || data["session_id"] == "" {
				t.Error("expected session_id in response")
			}
			if data["ws_url"] == nil || data["ws_url"] == "" {
				t.Error("expected ws_url in response")
			}
		})
	})

	// ── Phase I: Monitoring & Audit ──────────────────────────────────

	t.Run("Phase_I_Monitoring", func(t *testing.T) {
		t.Run("ViewBandwidth", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/servers/%s/bandwidth?period=hourly", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("ViewAuditLogs", func(t *testing.T) {
			w := doRequest(env.router, "GET", "/api/v1/audit-logs?page=1&page_size=50", env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			total := data["total"].(float64)
			if total < 1 {
				t.Error("expected at least 1 audit log entry")
			}
			t.Logf("Audit logs: %.0f entries recorded", total)
		})
	})

	// ── Phase J: Multi-Tenant ────────────────────────────────────────

	t.Run("Phase_J_MultiTenant", func(t *testing.T) {
		var childTenantID uuid.UUID

		t.Run("CreateTenant", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/tenants", env.token, map[string]any{
				"name":      "Tokyo Reseller",
				"slug":      "tokyo-reseller",
				"parent_id": env.tenantID.String(),
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			childTenantID = uuid.MustParse(data["id"].(string))
		})

		t.Run("GetTenantHierarchy", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/tenants/%s/tree", env.tenantID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
			data := extractMap(t, resp)
			children := data["children"]
			if children == nil {
				t.Error("expected children in tree response")
			}
		})

		t.Run("CreateUser", func(t *testing.T) {
			w := doRequest(env.router, "POST", "/api/v1/users", env.token, map[string]any{
				"email":    "operator@test.com",
				"password": "Operator123!",
				"name":     "Operator",
				"role_id":  env.roleID.String(),
			})
			assertStatus(t, w, http.StatusCreated)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		_ = childTenantID // used in CreateTenant
	})

	// ── Phase K: Cleanup ─────────────────────────────────────────────

	t.Run("Phase_K_Cleanup", func(t *testing.T) {
		t.Run("DeleteServer", func(t *testing.T) {
			w := doRequest(env.router, "DELETE", fmt.Sprintf("/api/v1/servers/%s", serverID), env.token, nil)
			assertStatus(t, w, http.StatusOK)
			resp := mustParseResponse(t, w)
			assertSuccess(t, resp)
		})

		t.Run("VerifyDeleted", func(t *testing.T) {
			w := doRequest(env.router, "GET", fmt.Sprintf("/api/v1/servers/%s", serverID), env.token, nil)
			assertStatus(t, w, http.StatusNotFound)
		})
	})
}
