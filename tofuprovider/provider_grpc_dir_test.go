package tofuprovider

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestStartGRPCPluginInDirSetsWorkingDir verifies StartGRPCPluginInDir launches
// the plugin child process with cmd.Dir set to the requested directory, so a
// provider that resolves a relative path targets that directory rather than the
// calling process's cwd.
//
// The stand-in "plugin" is a POSIX shell that records its working directory to a
// file relative to its own cwd and exits. The rpcplugin handshake then fails (it
// is not a real plugin) — that error is expected and ignored; the child has
// already run in the target dir by the time StartGRPCPluginInDir returns, so the
// file lands in dir only if cmd.Dir was honored.
func TestStartGRPCPluginInDirSetsWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell to capture the working directory")
	}
	dir := t.TempDir()

	_, _ = StartGRPCPluginInDir(context.Background(), dir, "/bin/sh", "-c", "pwd > cwd.txt")

	data, err := os.ReadFile(filepath.Join(dir, "cwd.txt"))
	if err != nil {
		t.Fatalf("child did not write cwd.txt in the target dir (cmd.Dir not honored?): %v", err)
	}
	got, err := filepath.EvalSymlinks(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("resolve child cwd: %v", err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("resolve target dir: %v", err)
	}
	if got != want {
		t.Fatalf("child ran in %q, want %q", got, want)
	}
}
