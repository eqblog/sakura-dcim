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
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/backend/internal/config"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/handler"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/middleware"
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

	// WebSocket Hub
	hub := ws.NewHub(logger)
	go hub.Run()

	// Services
	authService := service.NewAuthService(userRepo, roleRepo, cfg)
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

	// Gin
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "0.1.0"})
	})

	// API routes
	api := r.Group("/api/v1")

	// Public routes
	authHandler := handler.NewAuthHandler(authService)
	authHandler.RegisterRoutes(api)

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

	kvmHandler := handler.NewKVMHandler(kvmService, logger)
	kvmHandler.RegisterRoutes(protected.Group("", middleware.RequirePermission(roleRepo, domain.PermIPMIKVM)))
	kvmHandler.RegisterPublicRoutes(r)

	// Agent WebSocket endpoint (separate auth)
	r.GET("/api/v1/agents/ws", func(c *gin.Context) {
		handler.HandleAgentWebSocket(c, hub, agentRepo, logger)
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
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
