package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vicentereig/whatsapp-cli/internal/commands"
)

const defaultTimeout = 5 * time.Minute

var (
	version  = "dev"
	storeDir string
	app      *commands.App
)

func SetVersion(v string) {
	version = v
}

// errorJSON produces a properly-escaped JSON error string for stdout.
func errorJSON(msg string) string {
	escaped, _ := json.Marshal(msg)
	return fmt.Sprintf(`{"success":false,"data":null,"error":%s}`, escaped)
}

// exitCode is set by runWithApp to signal the process exit code.
// Execute() reads this after rootCmd.Execute() returns.
var exitCode int

// printResult writes a JSON result to stdout and records the exit code.
// Parses the success field from app method output to determine exit code.
// This is the ONLY function that writes to stdout.
func printResult(result string) {
	fmt.Println(result)

	var envelope struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal([]byte(result), &envelope); err != nil || !envelope.Success {
		exitCode = 1
	}
}

var rootCmd = &cobra.Command{
	Use:           "whatsapp-cli",
	Short:         "Command line interface for WhatsApp",
	Long:          "WhatsApp CLI - send messages, sync history, search contacts and chats.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// initApp is called by leaf commands that need the WhatsApp app.
// Leaf commands call this in RunE instead of using PersistentPreRunE,
// so validation errors (mutual exclusion, missing flags) fail fast
// without side effects.
func initApp() error {
	if app != nil {
		return nil
	}
	absStore, err := filepath.Abs(storeDir)
	if err != nil {
		return fmt.Errorf("invalid store path: %w", err)
	}
	app, err = commands.NewApp(absStore, version)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	return nil
}

func closeApp() {
	if app != nil {
		app.Close()
		app = nil
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&storeDir, "store", "./store", "storage directory")

	// Route all cobra output (help, usage, errors) to stderr.
	// stdout is reserved for JSON results only.
	rootCmd.SetOut(os.Stderr)
	rootCmd.SetErr(os.Stderr)
}

// newContext returns a context appropriate for the command.
// sync gets signal-based cancellation; everything else gets a timeout.
func newContext(isSync bool) (context.Context, context.CancelFunc) {
	if isSync {
		ctx, cancel := context.WithCancel(context.Background())
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()
		return ctx, cancel
	}
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// runWithApp wraps a command function that needs the app initialized.
// Handles init, execution, cleanup, and result printing.
// Init errors are runtime failures (exit 1), not usage errors (exit 2),
// so they go through printResult rather than returning to cobra.
func runWithApp(fn func() string) error {
	if err := initApp(); err != nil {
		printResult(errorJSON(err.Error()))
		return nil
	}
	defer closeApp()
	printResult(fn())
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra-level errors (bad flags, unknown commands, validation).
		// Print properly-escaped JSON to stdout, exit 2 for usage errors.
		fmt.Println(errorJSON(err.Error()))
		os.Exit(2)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
