package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/handler"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
	influxRepo "github.com/sakura-dcim/sakura-dcim/backend/internal/repository/influxdb"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/repository/postgres"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/service"
	ws "github.com/sakura-dcim/sakura-dcim/backend/internal/websocket"
)

func main() {
	// Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Config
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Database
	ctx := context.Background()
	db, err := postgres.NewPool(ctx, &cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}
	defer rdb.Close()

	// InfluxDB (optional — fail-open if not configured)
	var bandwidthInflux *influxRepo.BandwidthRepo
	if cfg.InfluxDB.URL != "" && cfg.InfluxDB.Token != "" {
		influxClient := influxRepo.NewClient(&cfg.InfluxDB)
		bandwidthInflux = influxRepo.NewBandwidthRepo(influxClient, &cfg.InfluxDB)
		defer bandwidthInflux.Close()
		logger.Info("InfluxDB connected", zap.String("url", cfg.InfluxDB.URL))
	} else {
		logger.Warn("InfluxDB not configured — bandwidth data stored in-memory only")
	}

	// Repositories
	userRepo := postgres.NewUserRepo(db)
	roleRepo := postgres.NewRoleRepo(db)
	serverRepo := postgres.NewServerRepo(db)
	agentRepo := postgres.NewAgentRepo(db)
	tenantRepo := postgres.NewTenantRepo(db)
	auditLogRepo := postgres.NewAuditLogRepo(db)
	osProfileRepo := postgres.NewOSProfileRepo(db)
	diskLayoutRepo := postgres.NewDiskLayoutRepo(db)
	scriptRepo := postgres.NewScriptRepo(db)
	installTaskRepo := postgres.NewInstallTaskRepo(db)
	switchRepo := postgres.NewSwitchRepo(db)
	switchPortRepo := postgres.NewSwitchPortRepo(db)
	inventoryRepo := postgres.NewInventoryRepo(db)
	ipPoolRepo := postgres.NewIPPoolRepo(db)
	ipAddressRepo := postgres.NewIPAddressRepo(db)
	discoverySessionRepo := postgres.NewDiscoverySessionRepo(db)
	discoveredServerRepo := postgres.NewDiscoveredServerRepo(db)

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
	inventoryService := service.NewInventoryService(inventoryRepo, serverRepo, hub, cfg, logger)
	ipService := service.NewIPService(ipPoolRepo, ipAddressRepo)
	ipService.SetSwitchDeps(switchService, switchPortRepo)
	discoveryService := service.NewDiscoveryService(discoverySessionRepo, discoveredServerRepo, serverRepo, agentRepo, hub, logger)
	provisionService := service.NewProvisionService(serverRepo, ipService, reinstallService, switchPortRepo, hub, logger)

	// Wire InfluxDB into bandwidth service
	if bandwidthInflux != nil {
		bandwidthService.SetInfluxRepo(bandwidthInflux)
	}

	// Register heartbeat event handler
	hub.OnEvent(ws.ActionAgentHeartbeat, func(agentID uuid.UUID, msg *ws.Message) {
		if err := agentService.UpdateLastSeen(context.Background(), agentID); err != nil {
			logger.Error("failed to update agent last_seen", zap.Error(err), zap.String("agent_id", agentID.String()))
		}
	})

	// Register PXE status event handler
	hub.OnEvent(ws.ActionPXEStatus, reinstallService.HandlePXEStatusEvent)

	// Register SNMP data event handler
	hub.OnEvent(ws.ActionSNMPData, bandwidthService.HandleSNMPDataEvent)

	// Register inventory result event handler
	hub.OnEvent(ws.ActionInventoryResult, inventoryService.HandleInventoryResultEvent)

	// Register discovery result event handler
	hub.OnEvent(ws.ActionDiscoveryResult, discoveryService.HandleDiscoveryResultEvent)

	// Start periodic SNMP port sync (every 5 minutes)
	snmpCtx, snmpCancel := context.WithCancel(context.Background())
	defer snmpCancel()
	go switchService.StartPeriodicSNMPSync(snmpCtx, 5*time.Minute)

	// Gin
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.PrometheusMetrics())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "0.1.0"})
	})

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	api := r.Group("/api/v1")

	// Public routes (rate-limited: 20 req/min for auth endpoints)
	authHandler := handler.NewAuthHandler(authService)
	authRateLimited := api.Group("", middleware.RateLimit(rdb, 20, time.Minute))
	authHandler.RegisterRoutes(authRateLimited)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg.JWT.Secret))
	protected.Use(middleware.AuditLog(auditLogRepo, logger))

	authHandler.RegisterProtectedRoutes(protected)

	serverHandler := handler.NewServerHandler(serverService)
	serverHandler.RegisterRoutes(protected)

	agentHandler := handler.NewAgentHandler(agentService, hub)
	agentHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermAgentManage)))

	userHandler := handler.NewUserHandler(userService)
	userHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermUserManage)))

	roleHandler := handler.NewRoleHandler(roleService)
	roleHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermRoleManage)))

	tenantHandler := handler.NewTenantHandler(tenantService)
	tenantHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermTenantManage)))

	ipmiHandler := handler.NewIPMIHandler(ipmiService)
	ipmiHandler.RegisterPowerRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermServerPower)))
	ipmiHandler.RegisterSensorRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermIPMISensors)))

	osProfileHandler := handler.NewOSProfileHandler(osProfileService)
	osProfileHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermOSProfileManage)))

	diskLayoutHandler := handler.NewDiskLayoutHandler(diskLayoutService)
	diskLayoutHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermDiskLayoutManage)))

	scriptHandler := handler.NewScriptHandler(scriptService)
	scriptHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermScriptManage)))

	reinstallHandler := handler.NewReinstallHandler(reinstallService)
	reinstallHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermOSReinstall)))

	provisionHandler := handler.NewProvisionHandler(provisionService)
	provisionHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermOSReinstall)))

	switchHandler := handler.NewSwitchHandler(switchService)
	switchHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermSwitchManage)))

	bandwidthHandler := handler.NewBandwidthHandler(bandwidthService)
	bandwidthHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermBandwidthView)))

	inventoryHandler := handler.NewInventoryHandler(inventoryService)
	inventoryHandler.RegisterRoutes(
		protected.Group("", middleware.RequirePermission(roleRepo, domain.PermInventoryView)),
		protected.Group("", middleware.RequirePermission(roleRepo, domain.PermInventoryScan)),
	)

	ipHandler := handler.NewIPHandler(ipService)
	ipHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermIPManage)))

	auditHandler := handler.NewAuditHandler(auditLogRepo)
	auditHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermAuditView)))

	kvmHandler := handler.NewKVMHandler(kvmService, logger)
	kvmHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermIPMIKVM)))
	kvmHandler.RegisterPublicRoutes(r)

	discoveryHandler := handler.NewDiscoveryHandler(discoveryService)
	discoveryHandler.RegisterRoutes(
		protected.Group("", middleware.RequirePermission(roleRepo, domain.PermDiscoveryView)),
		protected.Group("", middleware.RequirePermission(roleRepo, domain.PermDiscoveryManage)),
	)

	// Agent WebSocket endpoint (separate auth)
	r.GET("/api/v1/agents/ws", func(c *gin.Context) {
		handler.HandleAgentWebSocket(c, hub, agentRepo, logger)
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		// NOTE: Do NOT set ReadTimeout / WriteTimeout here.
		// Go's net/http applies them as deadlines on the underlying net.Conn,
		// which kills long-lived WebSocket connections (agent WS, KVM relay).
	}

	go func() {
		logger.Info("starting server", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("server forced shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}
