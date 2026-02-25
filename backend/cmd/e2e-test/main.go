package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ── colors ───────────────────────────────────────────────────────────

const (
	cReset  = "\033[0m"
	cRed    = "\033[31m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cCyan   = "\033[36m"
	cBold   = "\033[1m"
	cGray   = "\033[90m"
)

var (
	passed int
	failed int
	total  int
)

func pass(msg string) {
	passed++
	total++
	fmt.Printf("  %s[PASS]%s %s\n", cGreen, cReset, msg)
}

func fail(msg string, detail ...string) {
	failed++
	total++
	fmt.Printf("  %s[FAIL]%s %s\n", cRed, cReset, msg)
	if len(detail) > 0 && detail[0] != "" {
		fmt.Printf("         %s%s%s\n", cGray, detail[0], cReset)
	}
}

func section(name string) {
	fmt.Printf("\n%s%s── %s ──%s\n", cBold, cCyan, name, cReset)
}

// ── API helpers ──────────────────────────────────────────────────────

var (
	baseURL string
	token   string
)

type apiResp struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func api(method, path string, body any) (map[string]any, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respData, _ := io.ReadAll(resp.Body)

	var result map[string]any
	json.Unmarshal(respData, &result)

	return result, resp.StatusCode, nil
}

func getData(result map[string]any) map[string]any {
	if result == nil {
		return nil
	}
	if d, ok := result["data"]; ok {
		if m, ok := d.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getFloat(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func getNested(m map[string]any, keys ...string) map[string]any {
	current := m
	for _, key := range keys {
		if current == nil {
			return nil
		}
		v, ok := current[key]
		if !ok {
			return nil
		}
		nested, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		current = nested
	}
	return current
}

// ── main ─────────────────────────────────────────────────────────────

func main() {
	serverAddr := flag.String("server", "http://localhost:8080", "Backend server address")
	flag.Parse()
	baseURL = *serverAddr + "/api/v1"

	fmt.Printf("\n%s%s╔══════════════════════════════════════════════════╗%s\n", cBold, cGreen, cReset)
	fmt.Printf("%s%s║   Sakura DCIM — Real Environment E2E Test        ║%s\n", cBold, cGreen, cReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════╝%s\n\n", cBold, cGreen, cReset)

	// Check backend
	healthResp, err := http.Get(*serverAddr + "/health")
	if err != nil || healthResp.StatusCode != 200 {
		fmt.Printf("%sError: Backend not running at %s%s\n", cRed, *serverAddr, cReset)
		os.Exit(1)
	}
	healthResp.Body.Close()
	fmt.Printf("%sBackend is running at %s%s\n", cGreen, *serverAddr, cReset)

	// ── Phase 1: Authentication ──────────────────────────────────
	section("Phase 1: Authentication")

	result, status, _ := api("POST", "/auth/login", map[string]string{
		"email":    "admin@sakura-dcim.local",
		"password": "admin123",
	})
	if status == 200 {
		data := getData(result)
		token = getStr(data, "access_token")
		if token != "" {
			pass("Login → got access token")
		} else {
			fail("Login → no access_token in response")
		}
	} else {
		fail("Login", fmt.Sprintf("HTTP %d", status))
	}

	result, status, _ = api("GET", "/auth/me", nil)
	if status == 200 {
		data := getData(result)
		email := getStr(data, "email")
		pass(fmt.Sprintf("GetCurrentUser → %s", email))
	} else {
		fail("GetCurrentUser", fmt.Sprintf("HTTP %d", status))
	}

	// ── Phase 2: Agent Creation ──────────────────────────────────
	section("Phase 2: Agent Creation")

	var agentID, agentToken string
	result, status, _ = api("POST", "/agents", map[string]any{
		"name":         "E2E-Mock-DC-Tokyo",
		"location":     "Tokyo, Japan",
		"capabilities": []string{"ipmi", "pxe", "kvm", "snmp", "inventory"},
	})
	if status == 201 {
		data := getData(result)
		agentData := getNested(data, "agent")
		agentID = getStr(agentData, "id")
		agentToken = getStr(data, "token")
		pass(fmt.Sprintf("CreateAgent → ID: %s", agentID))
	} else {
		fail("CreateAgent", fmt.Sprintf("HTTP %d", status))
		os.Exit(1)
	}

	// ── Phase 3: Start Mock Agent ────────────────────────────────
	section("Phase 3: Start Mock Agent")

	// Find the mock-agent source relative to this binary
	mockAgentArgs := []string{
		"run", "./cmd/mock-agent",
		"-server", strings.Replace(*serverAddr, "http", "ws", 1) + "/api/v1/agents/ws",
		"-id", agentID,
		"-token", agentToken,
	}
	mockCmd := exec.Command("go", mockAgentArgs...)
	mockCmd.Dir = findBackendDir()
	mockCmd.Stdout = os.Stdout
	mockCmd.Stderr = os.Stderr
	if err := mockCmd.Start(); err != nil {
		fail("Start MockAgent", err.Error())
		os.Exit(1)
	}
	defer func() {
		if mockCmd.Process != nil {
			mockCmd.Process.Kill()
			mockCmd.Wait()
		}
	}()

	time.Sleep(4 * time.Second)
	pass(fmt.Sprintf("MockAgent started (PID: %d)", mockCmd.Process.Pid))

	// ── Phase 4: Server CRUD ─────────────────────────────────────
	section("Phase 4: Server CRUD")

	var serverID string
	result, status, _ = api("POST", "/servers", map[string]any{
		"hostname":   "e2e-web-prod-01",
		"label":      "E2E Production Server",
		"primary_ip": "10.0.1.100",
		"ipmi_ip":    "10.0.0.100",
		"ipmi_user":  "ADMIN",
		"ipmi_pass":  "S3cretIPMI!",
		"agent_id":   agentID,
		"tags":       []string{"e2e", "production"},
		"notes":      "Created by E2E test",
	})
	if status == 201 {
		data := getData(result)
		serverID = getStr(data, "id")
		pass(fmt.Sprintf("CreateServer → ID: %s", serverID))
	} else {
		fail("CreateServer", fmt.Sprintf("HTTP %d — %v", status, result))
		os.Exit(1)
	}

	result, status, _ = api("GET", "/servers?page=1&page_size=20", nil)
	if status == 200 {
		data := getData(result)
		pass(fmt.Sprintf("ListServers → %.0f server(s)", getFloat(data, "total")))
	} else {
		fail("ListServers", fmt.Sprintf("HTTP %d", status))
	}

	result, status, _ = api("GET", "/servers/"+serverID, nil)
	if status == 200 {
		data := getData(result)
		pass(fmt.Sprintf("GetServer → hostname: %s", getStr(data, "hostname")))
	} else {
		fail("GetServer", fmt.Sprintf("HTTP %d", status))
	}

	result, status, _ = api("PUT", "/servers/"+serverID, map[string]any{
		"label": "E2E Updated Server",
	})
	if status == 200 {
		data := getData(result)
		pass(fmt.Sprintf("UpdateServer → label: %s", getStr(data, "label")))
	} else {
		fail("UpdateServer", fmt.Sprintf("HTTP %d", status))
	}

	// ── Phase 5: IPMI Power Control ──────────────────────────────
	section("Phase 5: IPMI Power Control")

	result, status, _ = api("GET", "/servers/"+serverID+"/power", nil)
	if status == 200 {
		data := getData(result)
		pass(fmt.Sprintf("PowerStatus → %s", getStr(data, "status")))
	} else {
		fail("PowerStatus", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	for _, action := range []string{"off", "on", "cycle"} {
		result, status, _ = api("POST", "/servers/"+serverID+"/power", map[string]string{"action": action})
		if status == 200 {
			pass(fmt.Sprintf("Power %s → ok", strings.Title(action)))
		} else {
			fail(fmt.Sprintf("Power %s", strings.Title(action)), fmt.Sprintf("HTTP %d — %v", status, result))
		}
	}

	// ── Phase 6: IPMI Sensors ────────────────────────────────────
	section("Phase 6: IPMI Sensors")

	result, status, _ = api("GET", "/servers/"+serverID+"/sensors", nil)
	if status == 200 {
		data := getData(result)
		if sensors, ok := data["sensors"].([]any); ok {
			pass(fmt.Sprintf("ReadSensors → %d sensors", len(sensors)))
		} else {
			pass("ReadSensors → ok")
		}
	} else {
		fail("ReadSensors", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	// ── Phase 7: Inventory Scan ──────────────────────────────────
	section("Phase 7: Inventory Scan")

	result, status, _ = api("POST", "/servers/"+serverID+"/inventory/scan", nil)
	if status == 200 {
		pass("TriggerInventoryScan → ok")
	} else {
		fail("TriggerInventoryScan", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	fmt.Printf("  %sWaiting 3s for async inventory.result event...%s\n", cGray, cReset)
	time.Sleep(3 * time.Second)

	result, status, _ = api("GET", "/servers/"+serverID+"/inventory", nil)
	if status == 200 {
		data := getData(result)
		if comps, ok := data["components"].([]any); ok {
			pass(fmt.Sprintf("ViewInventory → %d component(s)", len(comps)))
		} else {
			pass("ViewInventory → ok")
		}
	} else {
		fail("ViewInventory", fmt.Sprintf("HTTP %d", status))
	}

	// ── Phase 8: Network — Switches & Ports ──────────────────────
	section("Phase 8: Network — Switches & Ports")

	var switchID, portID string

	result, status, _ = api("POST", "/switches", map[string]any{
		"name":           "E2E-TOR-Switch-01",
		"ip":             "10.0.0.1",
		"vendor":         "cisco_ios",
		"model":          "Nexus 9300",
		"snmp_community": "public",
		"snmp_version":   "v2c",
		"ssh_user":       "admin",
		"ssh_pass":       "switchpass",
		"ssh_port":       22,
		"agent_id":       agentID,
	})
	if status == 201 {
		data := getData(result)
		switchID = getStr(data, "id")
		pass(fmt.Sprintf("CreateSwitch → ID: %s", switchID))
	} else {
		fail("CreateSwitch", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	result, status, _ = api("POST", fmt.Sprintf("/switches/%s/ports", switchID), map[string]any{
		"port_index":   1,
		"port_name":    "Ethernet1/1",
		"speed_mbps":   10000,
		"vlan_id":      100,
		"admin_status": "up",
		"server_id":    serverID,
		"description":  "e2e-web-prod-01 uplink",
	})
	if status == 201 {
		data := getData(result)
		portID = getStr(data, "id")
		pass(fmt.Sprintf("CreateSwitchPort → ID: %s", portID))
	} else {
		fail("CreateSwitchPort", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	result, status, _ = api("POST", fmt.Sprintf("/switches/%s/ports/%s/provision", switchID, portID), nil)
	if status == 200 {
		pass("ProvisionPort → ok")
	} else {
		fail("ProvisionPort", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	result, status, _ = api("GET", fmt.Sprintf("/switches/%s/ports/%s/status", switchID, portID), nil)
	if status == 200 {
		pass("GetPortStatus → ok")
	} else {
		fail("GetPortStatus", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	// ── Phase 9: IP Address Management ───────────────────────────
	section("Phase 9: IP Address Management")

	var poolID string

	result, status, _ = api("POST", "/ip-pools", map[string]any{
		"network":     "10.0.1.0/24",
		"gateway":     "10.0.1.1",
		"description": "E2E Production VLAN",
	})
	if status == 201 {
		data := getData(result)
		poolID = getStr(data, "id")
		pass(fmt.Sprintf("CreateIPPool → ID: %s", poolID))
	} else {
		fail("CreateIPPool", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	for _, addr := range []string{"10.0.1.10", "10.0.1.11", "10.0.1.12"} {
		result, status, _ = api("POST", fmt.Sprintf("/ip-pools/%s/addresses", poolID), map[string]any{
			"address": addr,
			"status":  "available",
		})
		if status == 201 {
			pass(fmt.Sprintf("CreateAddress → %s", addr))
		} else {
			errMsg := ""
			if result != nil {
				if e, ok := result["error"]; ok {
					errMsg = fmt.Sprintf(" — %v", e)
				}
			}
			fail(fmt.Sprintf("CreateAddress %s", addr), fmt.Sprintf("HTTP %d%s", status, errMsg))
		}
	}

	result, status, _ = api("GET", fmt.Sprintf("/ip-pools/%s/addresses", poolID), nil)
	if status == 200 {
		pass("ListAddresses → ok")
	} else {
		fail("ListAddresses", fmt.Sprintf("HTTP %d", status))
	}

	result, status, _ = api("POST", fmt.Sprintf("/ip-pools/%s/assign", poolID), map[string]any{
		"server_id": serverID,
	})
	if status == 200 {
		data := getData(result)
		pass(fmt.Sprintf("AutoAssignIP → %s", getStr(data, "address")))
	} else {
		fail("AutoAssignIP", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	// ── Phase 10: OS Reinstallation ──────────────────────────────
	section("Phase 10: OS Reinstallation")

	var osProfileID, layoutID string

	result, status, _ = api("POST", "/os-profiles", map[string]any{
		"name":          "E2E Ubuntu 22.04",
		"os_family":     "ubuntu",
		"version":       "22.04",
		"arch":          "amd64",
		"kernel_url":    "http://archive.ubuntu.com/ubuntu/dists/jammy/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/linux",
		"initrd_url":    "http://archive.ubuntu.com/ubuntu/dists/jammy/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/initrd.gz",
		"boot_args":     "auto=true priority=critical",
		"template_type": "preseed",
		"template":      "d-i debian-installer/locale string en_US",
		"is_active":     true,
	})
	if status == 201 {
		data := getData(result)
		osProfileID = getStr(data, "id")
		pass(fmt.Sprintf("CreateOSProfile → ID: %s", osProfileID))
	} else {
		fail("CreateOSProfile", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	result, status, _ = api("POST", "/disk-layouts", map[string]any{
		"name":        "E2E Standard Layout",
		"description": "Boot + Root",
		"layout": map[string]any{
			"partitions": []map[string]string{
				{"mount": "/boot", "size": "1G", "fs": "ext4"},
				{"mount": "/", "size": "100%FREE", "fs": "ext4"},
			},
		},
	})
	if status == 201 {
		data := getData(result)
		layoutID = getStr(data, "id")
		pass(fmt.Sprintf("CreateDiskLayout → ID: %s", layoutID))
	} else {
		fail("CreateDiskLayout", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	result, status, _ = api("POST", fmt.Sprintf("/servers/%s/reinstall", serverID), map[string]any{
		"os_profile_id":  osProfileID,
		"disk_layout_id": layoutID,
		"raid_level":     "auto",
		"root_password":  "E2ESecurePass123!",
		"ssh_keys":       []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... e2e@test"},
	})
	if status == 200 {
		pass("StartReinstall → ok")
	} else {
		fail("StartReinstall", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	fmt.Printf("  %sWaiting 8s for PXE install simulation...%s\n", cGray, cReset)
	time.Sleep(8 * time.Second)

	result, status, _ = api("GET", fmt.Sprintf("/servers/%s/reinstall/status", serverID), nil)
	if status == 200 {
		data := getData(result)
		installStatus := getStr(data, "status")
		progress := getFloat(data, "progress")
		if installStatus == "completed" {
			pass(fmt.Sprintf("ReinstallStatus → %s (%.0f%%)", installStatus, progress))
		} else {
			fail("ReinstallStatus", fmt.Sprintf("expected 'completed', got '%s' (%.0f%%)", installStatus, progress))
		}
	} else {
		fail("ReinstallStatus", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	// ── Phase 11: KVM Console ────────────────────────────────────
	section("Phase 11: KVM Console")

	var sessionID string
	result, status, _ = api("POST", fmt.Sprintf("/servers/%s/kvm", serverID), nil)
	if status == 200 {
		data := getData(result)
		sessionID = getStr(data, "session_id")
		wsURL := getStr(data, "ws_url")
		pass(fmt.Sprintf("StartKVM → session: %s", sessionID))
		fmt.Printf("         %sws_url: %s%s\n", cGray, wsURL, cReset)
	} else {
		fail("StartKVM", fmt.Sprintf("HTTP %d — %v", status, result))
	}

	if sessionID != "" {
		result, status, _ = api("DELETE", fmt.Sprintf("/servers/%s/kvm?session=%s", serverID, sessionID), nil)
		if status == 200 {
			pass("StopKVM → ok")
		} else {
			fail("StopKVM", fmt.Sprintf("HTTP %d — %v", status, result))
		}
	}

	// ── Phase 12: Audit Logs ─────────────────────────────────────
	section("Phase 12: Audit Logs")

	result, status, _ = api("GET", "/audit-logs?page=1&page_size=50", nil)
	if status == 200 {
		data := getData(result)
		pass(fmt.Sprintf("ViewAuditLogs → %.0f entries", getFloat(data, "total")))
	} else {
		fail("ViewAuditLogs", fmt.Sprintf("HTTP %d", status))
	}

	// ── Phase 13: Cleanup ────────────────────────────────────────
	section("Phase 13: Cleanup")

	result, status, _ = api("DELETE", "/servers/"+serverID, nil)
	if status == 200 {
		pass("DeleteServer → ok")
	} else {
		fail("DeleteServer", fmt.Sprintf("HTTP %d", status))
	}

	_, status, _ = api("GET", "/servers/"+serverID, nil)
	if status == 404 {
		pass("VerifyDeleted → 404")
	} else {
		fail("VerifyDeleted", fmt.Sprintf("expected 404, got %d", status))
	}

	// Clean up all other resources to avoid stale data on next run
	if poolID != "" {
		api("DELETE", "/ip-pools/"+poolID, nil)
	}
	if switchID != "" {
		api("DELETE", "/switches/"+switchID, nil)
	}
	if osProfileID != "" {
		api("DELETE", "/os-profiles/"+osProfileID, nil)
	}
	if layoutID != "" {
		api("DELETE", "/disk-layouts/"+layoutID, nil)
	}
	if agentID != "" {
		api("DELETE", "/agents/"+agentID, nil)
	}
	fmt.Printf("  %s(cleaned up pools, switches, profiles, layouts, agent)%s\n", cGray, cReset)

	// ── Summary ──────────────────────────────────────────────────
	fmt.Printf("\n%s══════════════════════════════════════════════════%s\n", cBold, cReset)
	if failed == 0 {
		fmt.Printf("%s%s  Results: %d passed, %d failed out of %d tests%s\n", cBold, cGreen, passed, failed, total, cReset)
		fmt.Printf("%s  All tests passed!%s\n", cGreen, cReset)
	} else {
		fmt.Printf("%s%s  Results: %d passed, %d failed out of %d tests%s\n", cBold, cRed, passed, failed, total, cReset)
	}
	fmt.Printf("%s══════════════════════════════════════════════════%s\n\n", cBold, cReset)

	os.Exit(failed)
}

func findBackendDir() string {
	// Try relative paths from working directory
	candidates := []string{
		"backend",
		".",
		"../backend",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c + "/cmd/mock-agent/main.go"); err == nil {
			return c
		}
	}
	// Fallback
	if runtime.GOOS == "windows" {
		return "backend"
	}
	return "."
}
