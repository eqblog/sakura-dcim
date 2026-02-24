package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/sakura-dcim/sakura-dcim/agent/internal/client"
	"github.com/sakura-dcim/sakura-dcim/agent/internal/config"
	"github.com/sakura-dcim/sakura-dcim/agent/internal/executor"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	configPath := flag.String("config", "config.yaml", "Config file path")
	flag.Parse()

	if *showVersion {
		fmt.Printf("sakura-agent %s\n", version)
		os.Exit(0)
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Initialize executors
	ipmiExec := executor.NewIPMIExecutor(logger)
	inventoryExec := executor.NewInventoryExecutor(logger)
	kvmExec := executor.NewKVMExecutor(logger)

	// Create WebSocket client
	wsClient := client.NewWSClient(cfg, logger, map[string]client.ActionHandler{
		"ipmi.power.on":     ipmiExec.HandlePowerOn,
		"ipmi.power.off":    ipmiExec.HandlePowerOff,
		"ipmi.power.reset":  ipmiExec.HandlePowerReset,
		"ipmi.power.cycle":  ipmiExec.HandlePowerCycle,
		"ipmi.power.status": ipmiExec.HandlePowerStatus,
		"ipmi.sensors":      ipmiExec.HandleSensors,
		"inventory.scan":    inventoryExec.HandleScan,
		"ipmi.kvm.start":    kvmExec.HandleKVMStart,
		"ipmi.kvm.stop":     kvmExec.HandleKVMStop,
	})

	// Connect
	go wsClient.ConnectWithRetry()

	logger.Info("sakura-agent started",
		zap.String("version", version),
		zap.String("server", cfg.ServerURL),
		zap.String("agent_id", cfg.AgentID),
	)

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down agent...")
	kvmExec.StopAll()
	wsClient.Close()
}
