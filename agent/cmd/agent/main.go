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
	pxeExec := executor.NewPXEExecutor(logger)
	raidExec := executor.NewRAIDExecutor(logger)
	switchExec := executor.NewSwitchExecutor(logger)
	snmpExec := executor.NewSNMPExecutor(logger)
	solExec := executor.NewSOLExecutor(logger)

	// Build handler map
	handlers := map[string]client.ActionHandler{
		"ipmi.power.on":     ipmiExec.HandlePowerOn,
		"ipmi.power.off":    ipmiExec.HandlePowerOff,
		"ipmi.power.reset":  ipmiExec.HandlePowerReset,
		"ipmi.power.cycle":  ipmiExec.HandlePowerCycle,
		"ipmi.power.status": ipmiExec.HandlePowerStatus,
		"ipmi.sensors":      ipmiExec.HandleSensors,
		"ipmi.sol":          solExec.HandleSOL,
		"inventory.scan":    inventoryExec.HandleScan,
		"ipmi.kvm.start":    kvmExec.HandleKVMStart,
		"ipmi.kvm.stop":     kvmExec.HandleKVMStop,
		"ipmi.user.create":  ipmiExec.HandleCreateTempUser,
		"ipmi.user.delete":  ipmiExec.HandleDeleteTempUser,
		"pxe.prepare":       pxeExec.HandlePXEPrepare,
		"pxe.cleanup":       pxeExec.HandlePXECleanup,
		"raid.configure":    raidExec.HandleRAIDConfigure,
		"raid.status":       raidExec.HandleRAIDStatus,
		"switch.provision":   switchExec.HandleSwitchProvision,
		"switch.status":      switchExec.HandleSwitchStatus,
		"switch.dhcp_relay":  switchExec.HandleSwitchDHCPRelay,
		"switch.test":        switchExec.HandleTestConnection,
		"snmp.poll":         snmpExec.HandleSNMPPoll,
	}

	// Create WebSocket client
	wsClient := client.NewWSClient(cfg, logger, handlers)

	// Executors that need wsClient for sending events back to panel
	discoveryExec := executor.NewDiscoveryExecutor(logger, wsClient)
	handlers["discovery.start"] = discoveryExec.HandleDiscoveryStart
	handlers["discovery.stop"] = discoveryExec.HandleDiscoveryStop

	pxeInvExec := executor.NewPXEInventoryExecutor(logger, ipmiExec, pxeExec, wsClient)
	handlers["inventory.pxe"] = pxeInvExec.HandlePXEInventory

	// Config hot-reload watcher
	cfgWatcher := config.NewWatcher(*configPath, cfg, logger)
	cfgWatcher.OnChange(func(newCfg *config.Config) {
		logger.Info("config reloaded",
			zap.String("server_url", newCfg.ServerURL),
			zap.String("agent_id", newCfg.AgentID),
		)
	})
	go cfgWatcher.Watch()

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
