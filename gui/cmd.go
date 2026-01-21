package gui

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/textolytics/nbgo/cli"
	"github.com/textolytics/nbgo/conf"
	"github.com/textolytics/nbgo/logs"
)

// RunCLI runs the CLI application with given arguments
func RunCLI(args []string) error {
	fs := flag.NewFlagSet("nbgo-gui", flag.ContinueOnError)
	mode := fs.String("mode", "tui", "UI mode: tui (terminal), cli (command-line), or settings")
	configPath := fs.String("config", "nbgo.yml", "Configuration file path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Create logger
	logger := logs.NewStandardLogger(os.Stdout)

	// Load configuration using config manager
	cfgMgr := conf.NewManager()
	if err := cfgMgr.LoadYAML(*configPath); err != nil {
		logger.Warnf("Failed to load configuration file %s, using defaults: %v", *configPath, err)
	}
	cfg := &conf.Config{
		Environment: make(map[string]string),
		Gateways:    make(map[string]interface{}),
	}

	// Create command registry
	cmdReg := cli.NewCommandRegistry()

	// Create application
	app := NewApplication("NBGO GUI", "1.0.0", cfg, logger, cmdReg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize application
	if err := app.Initialize(ctx); err != nil {
		logger.Errorf("Failed to initialize application: %v", err)
		return err
	}

	// Start application
	if err := app.Start(ctx); err != nil {
		logger.Errorf("Failed to start application: %v", err)
		return err
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run based on mode
	switch *mode {
	case "cli":
		runCLIMode(app, logger, sigChan)
	case "settings":
		runSettingsMode(app, logger, sigChan)
	case "tui":
		fallthrough
	default:
		runTUIMode(app, logger, sigChan)
	}

	logger.Info("Shutting down...")

	// Stop application
	if err := app.Stop(ctx); err != nil {
		logger.Errorf("Error during shutdown: %v", err)
	}

	return nil
}

// runCLIMode runs the CLI interactive mode
func runCLIMode(app *Application, logger logs.Logger, sigChan <-chan os.Signal) {
	logger.Info("Starting CLI mode...")
	fmt.Println("=== NBGO CLI Interface ===")
	fmt.Println("Type 'help' for commands, 'exit' to quit")
	fmt.Println("")

	scanner := bufio.NewScanner(os.Stdin)
	done := make(chan bool)

	go func() {
		for {
			fmt.Print("nbgo> ")
			if !scanner.Scan() {
				done <- true
				return
			}

			cmdLine := scanner.Text()
			if cmdLine == "exit" || cmdLine == "quit" {
				done <- true
				return
			}

			if cmdLine == "help" {
				printCLIHelp()
				continue
			}

			if cmdLine == "" {
				continue
			}

			// Execute command
			output, err := app.ExecuteCommand(context.Background(), cmdLine)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			if output != "" {
				fmt.Println(output)
			}
		}
	}()

	// Wait for signal or done
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
	case <-done:
		logger.Info("User exit")
	}
}

// runTUIMode runs the terminal UI mode
func runTUIMode(app *Application, logger logs.Logger, sigChan <-chan os.Signal) {
	logger.Info("Starting TUI mode...")
	fmt.Println("=== NBGO Terminal UI ===")
	fmt.Println("Views available:")

	uiMgr := app.GetUIManager()
	if uiMgr != nil {
		views := uiMgr.ListViews()
		for name, view := range views {
			fmt.Printf("  [%s] - %s\n", name, view.Description)
		}
	}

	fmt.Println("\nManagers available:")
	managers := app.ListManagers()
	for managerType := range managers {
		fmt.Printf("  - %s\n", managerType)
	}

	fmt.Println("\nPress Ctrl+C to exit")

	// Wait for signal
	<-sigChan
}

// runSettingsMode runs the settings management mode
func runSettingsMode(app *Application, logger logs.Logger, sigChan <-chan os.Signal) {
	logger.Info("Starting Settings mode...")
	fmt.Println("=== NBGO Settings Manager ===")

	settingsMgr := NewSettingsManager(logger)
	if err := settingsMgr.LoadSchemasFromDirectory("schema"); err != nil {
		logger.Errorf("Failed to load schemas: %v", err)
		return
	}

	if err := settingsMgr.GenerateAggregatedSettings(); err != nil {
		logger.Errorf("Failed to generate aggregated settings: %v", err)
		return
	}

	if err := settingsMgr.ValidateSettings(); err != nil {
		logger.Errorf("Settings validation failed: %v", err)
		return
	}

	fmt.Println("Settings loaded and validated successfully")
	fmt.Println("\nAvailable settings:")
	settingsMgr.PrintSettings()

	// Wait for signal
	<-sigChan
}

// printCLIHelp prints CLI help
func printCLIHelp() {
	help := `
Available Commands:
  start <service>       - Start a service
  stop <service>        - Stop a service
  restart <service>     - Restart a service
  status                - Show system status
  list [type]           - List resources
  config [key] [value]  - Get or set configuration
  logs [service]        - View logs
  help                  - Show this help
  exit/quit             - Exit the program
`
	fmt.Print(help)
}
