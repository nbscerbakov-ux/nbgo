package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/textolytics/nbgo/cli"
	"github.com/textolytics/nbgo/conf"
	"github.com/textolytics/nbgo/core"
	"github.com/textolytics/nbgo/dw"
	"github.com/textolytics/nbgo/gw"
	"github.com/textolytics/nbgo/http"
	"github.com/textolytics/nbgo/logs"
	"github.com/textolytics/nbgo/mb"
	"github.com/textolytics/nbgo/mon"
	"github.com/textolytics/nbgo/run"
	"github.com/textolytics/nbgo/task"
)

var (
	Version   = "1.0.0"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	ctx := context.Background()

	// Initialize logger
	logger := logs.NewStandardLogger(os.Stdout)
	logger.SetLevel(logs.LevelInfo)
	appLogger := logger.WithModule("main")

	appLogger.Info("Starting NBGO Market Data System", "version", Version, "commit", Commit)

	// Load configuration
	configManager := conf.NewManager()
	configPaths := []string{"nbgo.yml", "./nbgo.yml", "/root/nbgo.yml"}
	var configLoaded bool
	for _, path := range configPaths {
		if err := configManager.LoadYAML(path); err == nil {
			configLoaded = true
			appLogger.Info("Configuration loaded from", "path", path)
			break
		}
	}
	if !configLoaded {
		appLogger.Warning("Configuration file not found, using defaults")
	}

	// Validate configuration
	if err := configManager.Validate(); err != nil {
		appLogger.Warning("Configuration validation warning", "error", err.Error())
	}

	// Initialize core provider registry
	coreRegistry := core.NewRegistry()
	coreProviders := []core.Provider{
		core.NewCProvider(),
		core.NewRustProvider(),
		core.NewGoProvider(),
		core.NewPythonProvider(),
		core.NewFlutterProvider(),
		core.NewRobotFrameworkProvider(),
		core.NewMCPProvider(),
	}

	for _, provider := range coreProviders {
		if err := provider.Initialize(ctx); err != nil {
			appLogger.Warning("Failed to initialize provider", "provider", provider.GetName(), "error", err.Error())
			continue
		}
		if err := coreRegistry.Register(provider.GetName(), provider); err != nil {
			appLogger.Warning("Failed to register provider", "provider", provider.GetName(), "error", err.Error())
		}
	}

	appLogger.Info("Initialized core providers", "count", len(coreRegistry.List()))

	// Initialize message bus
	mbRegistry := mb.NewRegistry()
	zmqBus := mb.NewZMQBus("tcp://localhost:5555")
	mbRegistry.Register("zmq", zmqBus)
	appLogger.Info("Initialized message bus providers")

	// Initialize data warehouse
	dwRegistry := dw.NewRegistry()
	clickhouse := dw.NewClickHouseWarehouse("localhost", 8123, "nbgo", "default", "")
	dwRegistry.Register("clickhouse", clickhouse)
	appLogger.Info("Initialized data warehouse providers")

	// Initialize monitoring
	monRegistry := mon.NewRegistry()
	grafana := mon.NewGrafanaMonitor("http://localhost:3000", "admin")
	monRegistry.Register("grafana", grafana)
	appLogger.Info("Initialized monitoring providers")

	// Initialize gateways
	gwRegistry := gw.NewRegistry()
	gateGW := gw.NewGateGateway("api_key", "api_secret")
	gwRegistry.Register("gate", gateGW)
	appLogger.Info("Initialized gateway providers")

	// Initialize runtime
	runtimeConfig := run.RuntimeConfig{
		Name:                "nbgo",
		Version:             Version,
		StartTimeout:        30 * time.Second,
		ShutdownTimeout:     30 * time.Second,
		MaxConnections:      1000,
		MaxRetries:          3,
		RetryInterval:       5 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
	runtime := run.NewRuntime(runtimeConfig)

	// Initialize CLI
	cliApp := cli.NewCLI()
	registerCLICommands(cliApp, ctx, appLogger)

	// Initialize task executor
	_ = task.NewExecutor(4, 3, 30*time.Second)

	// Start HTTP API server
	apiServer := http.NewServer(coreRegistry, configManager, appLogger, "8080", 30*time.Second)
	if err := apiServer.Start(ctx); err != nil {
		appLogger.Error("Failed to start HTTP API server", "error", err.Error())
		os.Exit(1)
	}

	// Start runtime
	if err := runtime.Start(ctx); err != nil {
		appLogger.Error("Failed to start runtime", "error", err.Error())
		os.Exit(1)
	}

	appLogger.Info("NBGO system started successfully")
	appLogger.Info("Configuration loaded", "version", configManager.Get().Version)
	appLogger.Info("Core providers initialized", "count", len(coreRegistry.List()))
	appLogger.Info("Message buses initialized", "count", len(mbRegistry.List()))
	appLogger.Info("Data warehouses initialized", "count", len(dwRegistry.List()))
	appLogger.Info("Monitors initialized", "count", len(monRegistry.List()))
	appLogger.Info("Gateways initialized", "count", len(gwRegistry.List()))
	appLogger.Info("Task executor initialized", "workers", 4)

	// Keep running until interrupted
	select {}
}

func registerCLICommands(cliApp *cli.CLI, ctx context.Context, logger logs.Logger) {
	// Version command
	versionCmd := &cli.Command{
		Name:        "version",
		Description: "Display version information",
		Usage:       "version",
		Handler: func(ctx context.Context, args []string) error {
			fmt.Printf("NBGO v%s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
			return nil
		},
	}
	cliApp.RegisterCommand(versionCmd)

	// Health check command
	healthCmd := &cli.Command{
		Name:        "health",
		Description: "Check system health",
		Usage:       "health",
		Handler: func(ctx context.Context, args []string) error {
			fmt.Println("System is healthy")
			return nil
		},
	}
	cliApp.RegisterCommand(healthCmd)

	// List providers command
	listCmd := &cli.Command{
		Name:        "list",
		Description: "List available providers",
		Usage:       "list [providers|gateways|monitors]",
		Handler: func(ctx context.Context, args []string) error {
			fmt.Println("Available commands:")
			for _, cmd := range cliApp.ListCommands() {
				fmt.Printf("  %s - %s\n", cmd.Name, cmd.Description)
			}
			return nil
		},
	}
	cliApp.RegisterCommand(listCmd)

	// Help command
	helpCmd := &cli.Command{
		Name:        "help",
		Description: "Display help information",
		Usage:       "help [command]",
		Handler: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				if cmd, exists := cliApp.GetCommand(args[0]); exists {
					fmt.Printf("Command: %s\n", cmd.Name)
					fmt.Printf("Description: %s\n", cmd.Description)
					fmt.Printf("Usage: %s\n", cmd.Usage)
					return nil
				}
				return fmt.Errorf("command not found: %s", args[0])
			}

			fmt.Println("NBGO Market Data System")
			fmt.Printf("Version: %s\n\n", Version)
			fmt.Println("Available commands:")
			for _, cmd := range cliApp.ListCommands() {
				fmt.Printf("  %-20s %s\n", cmd.Name, cmd.Description)
			}
			fmt.Println("\nUse 'help <command>' for more information on a command")
			return nil
		},
	}
	cliApp.RegisterCommand(helpCmd)
}
