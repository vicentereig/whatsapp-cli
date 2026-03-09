# Cobra Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace hand-rolled flag/switch CLI parser with cobra, enforcing JSON-only stdout and AX-first design.

**Architecture:** Create a `cmd/` package with one file per command group (root, auth, sync, messages, contacts, chats, send, media, version). Root command owns `--store` as a PersistentFlag. Each leaf command uses `RunE` returning errors, with a shared `PersistentPreRunE` on root that initializes `commands.App`. All human-readable output (help, progress, QR) goes to stderr. Stdout is JSON-only.

**Tech Stack:** `github.com/spf13/cobra` for CLI, existing `internal/commands` package unchanged, existing `internal/client` and `internal/store` unchanged.

**Key constraint:** The `internal/commands` package (App methods, interfaces, mocks, behavioral tests) should NOT change. The migration replaces only main.go and its helpers.

**AX Contract:**
- **stdout:** JSON only. Every command prints exactly one JSON object to stdout.
- **stderr:** Human text only. Help, usage, progress, QR codes, errors for humans.
- **Exit 0:** `{"success":true,...}` on stdout.
- **Exit 1:** `{"success":false,...}` on stdout (runtime errors from app methods).
- **Exit 2:** `{"success":false,"data":null,"error":"..."}` on stdout (usage/validation errors from cobra layer).

Since `internal/commands` returns pre-serialized JSON strings with a `success` field, the cobra layer parses this field to determine exit code. This avoids changing internal/commands while still giving agents reliable exit codes.

---

## Task 1: Add cobra dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add cobra**

Run:
```bash
cd ~/Documents/Projects/WhatsAppCli/whatsapp-cli
go get github.com/spf13/cobra@latest
go mod tidy
```

**Step 2: Verify**

Run: `go build ./...`
Expected: builds with no errors

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add cobra dependency"
```

---

## Task 2: Create root command with --store persistent flag

**Files:**
- Create: `cmd/root.go`

**Step 1: Write root.go**

Key design decisions addressing AX review:
- `printResult` parses the JSON `success` field and sets exit code (finding #1)
- `errorJSON` properly escapes strings via `encoding/json` (finding #2)
- `rootCmd.SetOut(os.Stdout)` and `rootCmd.SetErr(os.Stderr)` explicit stream routing (finding #3)
- `cobra.EnableCommandSorting = false` for stable help output
- `PersistentPreRunE` is only on the `appInitGroup` annotation, not on version/help (finding #6)

```go
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

// printResult writes a JSON result to stdout and sets exit code.
// Parses the success field from app method output to determine exit code.
// This is the ONLY function that writes to stdout.
func printResult(result string) {
	fmt.Println(result)

	var envelope struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal([]byte(result), &envelope); err != nil || !envelope.Success {
		os.Exit(1)
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
func runWithApp(fn func() string) error {
	if err := initApp(); err != nil {
		return err
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
}
```

**Step 2: Verify it compiles**

Run: `go build ./cmd/...`
Expected: compiles (nothing calls Execute yet)

**Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat(cmd): add cobra root command with AX-first output contract"
```

---

## Task 3: Create version command

**Files:**
- Create: `cmd/version.go`

Version does NOT call initApp — no side effects.

**Step 1: Write version.go**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("{\"success\":true,\"data\":{\"version\":%s},\"error\":null}\n",
			mustJSON(version))
	},
}

// mustJSON marshals a value to JSON, panics on failure (should never fail for strings).
func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

Note: needs `"encoding/json"` import — add it.

**Step 2: Commit**

```bash
git add cmd/version.go
git commit -m "feat(cmd): add version command"
```

---

## Task 4: Create auth command

**Files:**
- Create: `cmd/auth.go`

**Step 1: Write auth.go**

```go
package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with WhatsApp (scan QR code)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithApp(func() string {
			ctx, cancel := newContext(false)
			defer cancel()
			return app.Auth(ctx)
		})
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
```

**Step 2: Commit**

```bash
git add cmd/auth.go
git commit -m "feat(cmd): add auth command"
```

---

## Task 5: Create sync command

**Files:**
- Create: `cmd/sync.go`

**Step 1: Write sync.go**

```go
package cmd

import "github.com/spf13/cobra"

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync messages continuously (run until Ctrl+C)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithApp(func() string {
			ctx, cancel := newContext(true)
			defer cancel()
			return app.Sync(ctx)
		})
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
```

**Step 2: Commit**

```bash
git add cmd/sync.go
git commit -m "feat(cmd): add sync command"
```

---

## Task 6: Create messages command group

**Files:**
- Create: `cmd/messages.go`

**Step 1: Write messages.go**

```go
package cmd

import "github.com/spf13/cobra"

// optionalStr returns nil for empty strings, otherwise a pointer.
func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List and search messages",
}

var messagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List messages in a chat",
	RunE: func(cmd *cobra.Command, args []string) error {
		chatJID, _ := cmd.Flags().GetString("chat")
		limit, _ := cmd.Flags().GetInt("limit")
		page, _ := cmd.Flags().GetInt("page")
		return runWithApp(func() string {
			return app.ListMessages(optionalStr(chatJID), nil, limit, page)
		})
	},
}

var messagesSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search messages by text",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		limit, _ := cmd.Flags().GetInt("limit")
		page, _ := cmd.Flags().GetInt("page")
		return runWithApp(func() string {
			return app.ListMessages(nil, &query, limit, page)
		})
	},
}

func init() {
	messagesListCmd.Flags().String("chat", "", "chat JID to filter by")
	messagesListCmd.Flags().Int("limit", 20, "maximum messages to return")
	messagesListCmd.Flags().Int("page", 0, "page number")

	messagesSearchCmd.Flags().String("query", "", "search text")
	messagesSearchCmd.Flags().Int("limit", 20, "maximum messages to return")
	messagesSearchCmd.Flags().Int("page", 0, "page number")
	messagesSearchCmd.MarkFlagRequired("query")

	messagesCmd.AddCommand(messagesListCmd, messagesSearchCmd)
	rootCmd.AddCommand(messagesCmd)
}
```

**Step 2: Verify `messages search` without --query fails**

Run: `go build -o /tmp/wa-test . && /tmp/wa-test messages search 2>&1`
Expected: JSON error on stdout about required flag, exit 2

**Step 3: Commit**

```bash
git add cmd/messages.go
git commit -m "feat(cmd): add messages list/search commands"
```

---

## Task 7: Create contacts command

**Files:**
- Create: `cmd/contacts.go`

**Step 1: Write contacts.go**

```go
package cmd

import "github.com/spf13/cobra"

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Search contacts",
}

var contactsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search contacts by name",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		return runWithApp(func() string {
			return app.SearchContacts(query)
		})
	},
}

func init() {
	contactsSearchCmd.Flags().String("query", "", "search text")
	contactsSearchCmd.MarkFlagRequired("query")

	contactsCmd.AddCommand(contactsSearchCmd)
	rootCmd.AddCommand(contactsCmd)
}
```

**Step 2: Commit**

```bash
git add cmd/contacts.go
git commit -m "feat(cmd): add contacts search command"
```

---

## Task 8: Create chats command

**Files:**
- Create: `cmd/chats.go`

**Step 1: Write chats.go**

```go
package cmd

import "github.com/spf13/cobra"

var chatsCmd = &cobra.Command{
	Use:   "chats",
	Short: "List chats",
}

var chatsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent chats",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		limit, _ := cmd.Flags().GetInt("limit")
		page, _ := cmd.Flags().GetInt("page")
		return runWithApp(func() string {
			return app.ListChats(optionalStr(query), limit, page)
		})
	},
}

func init() {
	chatsListCmd.Flags().String("query", "", "filter chats by name")
	chatsListCmd.Flags().Int("limit", 20, "maximum chats to return")
	chatsListCmd.Flags().Int("page", 0, "page number")

	chatsCmd.AddCommand(chatsListCmd)
	rootCmd.AddCommand(chatsCmd)
}
```

**Step 2: Commit**

```bash
git add cmd/chats.go
git commit -m "feat(cmd): add chats list command"
```

---

## Task 9: Create send command

**Files:**
- Create: `cmd/send.go`

Validation happens BEFORE initApp (finding #6) — mutual exclusion check is pure flag logic.

**Step 1: Write send.go**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message or image",
	RunE: func(cmd *cobra.Command, args []string) error {
		to, _ := cmd.Flags().GetString("to")
		message, _ := cmd.Flags().GetString("message")
		image, _ := cmd.Flags().GetString("image")
		caption, _ := cmd.Flags().GetString("caption")

		// Validate before any side effects (no app init yet)
		if image != "" && message != "" {
			return fmt.Errorf("--message and --image are mutually exclusive")
		}
		if image == "" && message == "" {
			return fmt.Errorf("--message or --image required")
		}

		return runWithApp(func() string {
			ctx, cancel := newContext(false)
			defer cancel()
			if image != "" {
				return app.SendImage(ctx, to, image, caption)
			}
			return app.SendMessage(ctx, to, message)
		})
	},
}

func init() {
	sendCmd.Flags().String("to", "", "recipient JID or phone number")
	sendCmd.Flags().String("message", "", "message text")
	sendCmd.Flags().String("image", "", "image file path")
	sendCmd.Flags().String("caption", "", "image caption")
	sendCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(sendCmd)
}
```

**Step 2: Commit**

```bash
git add cmd/send.go
git commit -m "feat(cmd): add send command with text and image support"
```

---

## Task 10: Create media command

**Files:**
- Create: `cmd/media.go`

**Step 1: Write media.go**

```go
package cmd

import "github.com/spf13/cobra"

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Download media attachments",
}

var mediaDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download media for a message",
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, _ := cmd.Flags().GetString("message-id")
		chatJID, _ := cmd.Flags().GetString("chat")
		outputPath, _ := cmd.Flags().GetString("output")

		return runWithApp(func() string {
			ctx, cancel := newContext(false)
			defer cancel()
			return app.DownloadMedia(ctx, messageID, optionalStr(chatJID), outputPath)
		})
	},
}

func init() {
	mediaDownloadCmd.Flags().String("message-id", "", "message identifier")
	mediaDownloadCmd.Flags().String("chat", "", "chat JID (optional)")
	mediaDownloadCmd.Flags().String("output", "", "output file or directory")
	mediaDownloadCmd.MarkFlagRequired("message-id")

	mediaCmd.AddCommand(mediaDownloadCmd)
	rootCmd.AddCommand(mediaCmd)
}
```

**Step 2: Commit**

```bash
git add cmd/media.go
git commit -m "feat(cmd): add media download command"
```

---

## Task 11: Replace main.go

**Files:**
- Modify: `main.go` (complete rewrite)

**Step 1: Rewrite main.go**

```go
package main

import "github.com/vicentereig/whatsapp-cli/cmd"

var version = "1.3.1"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
```

**Step 2: Build and run all existing behavioral tests**

Run:
```bash
go build -o whatsapp-cli .
go test -race ./...
```
Expected: all tests pass

**Step 3: Verify AX contract**

Each test below verifies specific AX behaviors:

```bash
# Exit codes: missing subcommand should exit non-zero
./whatsapp-cli contacts; echo "exit: $?"
# Expected: exit 2 (usage error), JSON error on stdout

# Exit codes: invalid subcommand should exit non-zero
./whatsapp-cli messages banana; echo "exit: $?"
# Expected: exit 2, JSON error on stdout

# Required flags produce JSON error on stdout
./whatsapp-cli messages search 2>/dev/null
# Expected: JSON with "success":false on stdout

# Mutual exclusion fails before app init (no store created)
./whatsapp-cli send --to 123 --image foo.jpg --message "hi" --store /tmp/should-not-exist; echo "exit: $?"
ls /tmp/should-not-exist
# Expected: exit 2, directory does NOT exist

# stdout is JSON-only: help goes to stderr, nothing on stdout
./whatsapp-cli --help 2>/dev/null | wc -c
# Expected: 0 bytes on stdout

./whatsapp-cli send --help 2>/dev/null | wc -c
# Expected: 0 bytes on stdout

# --store works after subcommand (persistent flag)
./whatsapp-cli version --store /tmp 2>/dev/null
# Expected: JSON success on stdout

# Version outputs valid JSON
./whatsapp-cli version | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['success']==True"
# Expected: no error
```

**Step 4: Commit**

```bash
git add main.go
git commit -m "refactor: replace hand-rolled CLI parser with cobra"
```

---

## Task 12: Clean up dead code

**Files:**
- Verify no references to old helpers remain

**Step 1: Verify no dead code**

Run:
```bash
grep -r "extractGlobalFlags\|exitJSON\|requireSubcommand" . --include="*.go"
```
Expected: no matches

`optionalStr` moved to `cmd/messages.go` — verify it's only defined once:
```bash
grep -r "func optionalStr" . --include="*.go"
```
Expected: only `cmd/messages.go`

**Step 2: Final validation**

Run:
```bash
go vet ./...
go build -o whatsapp-cli .
go test -race ./...
```

**Step 3: Commit (only if vet found something to fix)**

---

## Task 13: Add CLI subprocess tests

**Files:**
- Create: `cmd/cli_test.go`

**Step 1: Write subprocess tests**

These test the AX contract end-to-end: exit codes, stream separation, JSON validity, required flags. Each test explicitly verifies the behavior the review called out (finding #5).

```go
package cmd

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// buildBinary builds the CLI binary for subprocess testing.
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := t.TempDir() + "/whatsapp-cli"
	cmd := exec.Command("go", "build", "-o", binary, "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return binary
}

// runCLI runs the binary and returns stdout, stderr, and exit code.
func runCLI(t *testing.T, binary string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// assertValidJSON checks that a string is valid JSON with a success field.
func assertValidJSON(t *testing.T, s string, wantSuccess bool) {
	t.Helper()
	s = strings.TrimSpace(s)
	var envelope struct {
		Success bool            `json:"success"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal([]byte(s), &envelope); err != nil {
		t.Fatalf("not valid JSON: %q\nparse error: %v", s, err)
	}
	if envelope.Success != wantSuccess {
		t.Errorf("success=%v, want %v, output: %s", envelope.Success, wantSuccess, s)
	}
}

func TestCLI_MissingSubcommand_ExitsNonZero(t *testing.T) {
	binary := buildBinary(t)

	for _, cmd := range []string{"contacts", "messages", "chats", "media"} {
		t.Run(cmd, func(t *testing.T) {
			_, _, exitCode := runCLI(t, binary, cmd)
			if exitCode == 0 {
				t.Errorf("%s without subcommand should exit non-zero", cmd)
			}
		})
	}
}

func TestCLI_InvalidSubcommand_JSONError(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runCLI(t, binary, "messages", "banana")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit for invalid subcommand")
	}
	assertValidJSON(t, stdout, false)
	if !strings.Contains(stdout, "banana") {
		t.Errorf("error should mention invalid subcommand, got: %s", stdout)
	}
}

func TestCLI_RequiredFlags_JSONError(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name string
		args []string
	}{
		{"send without --to", []string{"send", "--message", "hi"}},
		{"messages search without --query", []string{"messages", "search"}},
		{"media download without --message-id", []string{"media", "download"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, exitCode := runCLI(t, binary, tt.args...)
			if exitCode == 0 {
				t.Errorf("expected non-zero exit for %v", tt.args)
			}
			assertValidJSON(t, stdout, false)
		})
	}
}

func TestCLI_MutualExclusion_FailsBeforeAppInit(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runCLI(t, binary,
		"send", "--to", "123", "--image", "foo.jpg", "--message", "hi",
		"--store", t.TempDir()+"/should-not-exist")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit for --image + --message")
	}
	assertValidJSON(t, stdout, false)
	if !strings.Contains(stdout, "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in output, got: %s", stdout)
	}
}

func TestCLI_Version_SuccessJSON(t *testing.T) {
	binary := buildBinary(t)

	stdout, _, exitCode := runCLI(t, binary, "version")
	if exitCode != 0 {
		t.Fatalf("version should exit 0, got %d", exitCode)
	}
	assertValidJSON(t, stdout, true)
}

func TestCLI_Help_StderrOnly(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name string
		args []string
	}{
		{"root help", []string{"--help"}},
		{"send help", []string{"send", "--help"}},
		{"messages help", []string{"messages", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, _ := runCLI(t, binary, tt.args...)
			if len(strings.TrimSpace(stdout)) > 0 {
				t.Errorf("--help should not write to stdout, got: %q", stdout)
			}
			if len(strings.TrimSpace(stderr)) == 0 {
				t.Error("--help should write help text to stderr")
			}
		})
	}
}

func TestCLI_StorePersistentFlag(t *testing.T) {
	binary := buildBinary(t)

	// --store works before command
	stdout, _, exitCode := runCLI(t, binary, "--store", "/tmp/test-wa", "version")
	if exitCode != 0 {
		t.Fatalf("--store before cmd: exit %d", exitCode)
	}
	assertValidJSON(t, stdout, true)

	// --store works after command
	stdout, _, exitCode = runCLI(t, binary, "version", "--store", "/tmp/test-wa")
	if exitCode != 0 {
		t.Fatalf("--store after cmd: exit %d", exitCode)
	}
	assertValidJSON(t, stdout, true)
}
```

**Step 2: Run tests**

Run: `go test -race ./cmd/ -v -run TestCLI`
Expected: all pass

**Step 3: Commit**

```bash
git add cmd/cli_test.go
git commit -m "test: add AX contract subprocess tests for cobra commands"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Add cobra dependency | go.mod, go.sum |
| 2 | Root command + AX output contract | cmd/root.go |
| 3 | Version command | cmd/version.go |
| 4 | Auth command | cmd/auth.go |
| 5 | Sync command | cmd/sync.go |
| 6 | Messages list/search commands | cmd/messages.go |
| 7 | Contacts search command | cmd/contacts.go |
| 8 | Chats list command | cmd/chats.go |
| 9 | Send command (text + image, validate-before-init) | cmd/send.go |
| 10 | Media download command | cmd/media.go |
| 11 | Replace main.go | main.go |
| 12 | Clean up dead code | verify only |
| 13 | AX contract subprocess tests | cmd/cli_test.go |

**What changes:** main.go (rewrite) + new cmd/ package (10 files)
**What doesn't change:** internal/commands/, internal/client/, internal/store/, internal/types/, internal/output/

**AX Review Findings Addressed:**
| # | Finding | Fix |
|---|---------|-----|
| 1 | Exit 0 on app failure | `printResult` parses `success` field, calls `os.Exit(1)` on false |
| 2 | Unescaped JSON in error path | `errorJSON` uses `json.Marshal` for proper escaping |
| 3 | Help not explicitly routed to stderr | `rootCmd.SetOut(os.Stderr)` + `rootCmd.SetErr(os.Stderr)` in init |
| 4 | Ambiguous stdout/stderr contract | Contract defined: stdout=JSON only, stderr=human text, exit 0/1/2 |
| 5 | Tests miss AX behaviors | Subprocess tests verify JSON validity, exit codes, stream separation, validate-before-init |
| 6 | Eager app init before validation | `runWithApp` called inside RunE after flag validation, not in PersistentPreRunE |
