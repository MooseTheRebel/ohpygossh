package gohpygossh_test

import (
	"gohpygossh"
	"path/filepath"
	"testing"
	"time"
)

func knownHostsOptions(t *testing.T, s *testSSHD) *gohpygossh.RunOptions {
	t.Helper()
	kh := filepath.Join(t.TempDir(), "known_hosts")
	s.writeKnownHosts(t, kh)
	return &gohpygossh.RunOptions{Policy: &gohpygossh.HostKeyPolicy{KnownHostsPath: kh}}
}

func TestRunWithOptionsReturnsStdoutStderrExitCode(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	res, err := gohpygossh.RunWithOptions(s.Addr, s.User, s.ClientKey,
		"sh -c 'echo out; echo err >&2; exit 3'", knownHostsOptions(t, s))
	if err != nil {
		t.Fatalf("want nil error for a non-zero exit, got %v", err)
	}
	if res.Stdout != "out\n" || res.Stderr != "err\n" || res.ExitCode != 3 {
		t.Fatalf("got %+v, want Stdout=%q Stderr=%q ExitCode=3", res, "out\n", "err\n")
	}
}

func TestRunWithOptionsTimeout(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	opts := knownHostsOptions(t, s)
	opts.TimeoutSeconds = 1
	start := time.Now()
	_, err := gohpygossh.RunWithOptions(s.Addr, s.User, s.ClientKey, "sleep 30", opts)
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("timeout took %v", elapsed)
	}
	wantErrContaining(t, err, "ohpygossh: command timed out after 1s")
}

// Run keeps its old contract: stdout only, and an error for a non-zero exit.
func TestRunNonZeroExitStillErrors(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	out, err := gohpygossh.RunWithPolicy(s.Addr, s.User, s.ClientKey,
		"sh -c 'echo out; exit 3'", knownHostsOptions(t, s).Policy)
	if out != "out\n" {
		t.Fatalf("out = %q", out)
	}
	wantErrContaining(t, err, "exited with status 3")
}
