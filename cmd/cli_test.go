package cmd_test

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

func TestCLI_InitFailure_Exits1_Not2(t *testing.T) {
	binary := buildBinary(t)

	// /dev/null/wa is an invalid path — triggers runtime init failure, not usage error
	stdout, _, exitCode := runCLI(t, binary,
		"send", "--to", "123", "--message", "hi", "--store", "/dev/null/wa")
	if exitCode != 1 {
		t.Fatalf("init failure should exit 1, got %d", exitCode)
	}
	assertValidJSON(t, stdout, false)
}

func TestCLI_StrayArgs_Rejected(t *testing.T) {
	binary := buildBinary(t)

	tests := []struct {
		name string
		args []string
	}{
		{"version banana", []string{"version", "banana"}},
		{"auth extra", []string{"auth", "extra"}},
		{"sync extra", []string{"sync", "extra"}},
		{"send extra", []string{"send", "--to", "123", "--message", "hi", "extra"}},
		{"messages list extra", []string{"messages", "list", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, exitCode := runCLI(t, binary, tt.args...)
			if exitCode == 0 {
				t.Errorf("stray args should exit non-zero for %v", tt.args)
			}
			assertValidJSON(t, stdout, false)
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
