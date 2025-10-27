package commands

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveVersionReturnsExplicitValue(t *testing.T) {
	var described bool
	describeFn := func() (string, error) {
		described = true
		return "v0.0.0", nil
	}

	version := resolveVersion("v1.2.3", describeFn)

	require.False(t, described, "describeFn should not be called when version is explicit")
	require.Equal(t, "v1.2.3", version)
}

func TestResolveVersionFallsBackToGitDescribe(t *testing.T) {
	describeFn := func() (string, error) {
		return "v1.2.3-4-gabcdef0", nil
	}

	version := resolveVersion("dev", describeFn)

	require.Equal(t, "v1.2.3-4-gabcdef0", version)
}

func TestResolveVersionReturnsDevWhenDescribeFails(t *testing.T) {
	describeFn := func() (string, error) {
		return "", errors.New("git describe failed")
	}

	version := resolveVersion("dev", describeFn)

	require.Equal(t, "dev", version)
}
