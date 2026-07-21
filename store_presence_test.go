package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// A missing store directory must be reported, not created. `--store` defaults to
// a CWD-relative "./store", so without this guard running any command from the
// wrong directory produces an empty store that reads as zero messages —
// indistinguishable from data loss.
func TestCheckStorePresence(t *testing.T) {
	existing := t.TempDir()
	missing := filepath.Join(existing, "does-not-exist")

	tests := []struct {
		name      string
		command   string
		storeDir  string
		wantError bool
	}{
		{
			name:      "read command against a missing store is rejected",
			command:   "chats",
			storeDir:  missing,
			wantError: true,
		},
		{
			name:      "sync against a missing store is rejected",
			command:   "sync",
			storeDir:  missing,
			wantError: true,
		},
		{
			name:      "auth may create a store that does not exist yet",
			command:   "auth",
			storeDir:  missing,
			wantError: false,
		},
		{
			name:      "read command against an existing store is allowed",
			command:   "chats",
			storeDir:  existing,
			wantError: false,
		},
		{
			name:      "auth against an existing store is allowed",
			command:   "auth",
			storeDir:  existing,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkStorePresence(tt.command, tt.storeDir)
			if tt.wantError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "store does not exist")
			} else {
				require.NoError(t, err)
			}
		})
	}

	// The guard must not have created anything as a side effect.
	_, err := os.Stat(missing)
	require.True(t, os.IsNotExist(err), "checkStorePresence must not create the store directory")
}
