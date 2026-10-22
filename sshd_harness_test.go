package gohpygossh_test

import (
	"bytes"
	"fmt"
	"gohpygossh"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testSSHD is a real, unprivileged OpenSSH sshd listening on 127.0.0.1.
// A non-root sshd can only authenticate the user it runs as.
type testSSHD struct {
	Addr        string            // "127.0.0.1:PORT"
	Port        int               //
	User        string            // current OS user
	ClientKey   string            // private key listed in authorized_keys
	HostPubKeys map[string]string // ssh-keygen type -> "<algo> <base64>"
	Dir         string            //
}

// findSSHD returns an absolute path to sshd (OpenSSH refuses to re-exec from
// a relative one). It skips the test when sshd is absent, unless
// OHPYGOSSH_REQUIRE_SSHD=1, in which case it fails.
func findSSHD(t *testing.T) string {
	t.Helper()
	if p, err := exec.LookPath("sshd"); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	if _, err := os.Stat("/usr/sbin/sshd"); err == nil {
		return "/usr/sbin/sshd"
	}
	if os.Getenv("OHPYGOSSH_REQUIRE_SSHD") == "1" {
		t.Fatal("sshd not installed, and OHPYGOSSH_REQUIRE_SSHD=1")
	}
	t.Skip("sshd not installed")
	return ""
}

// startSSHD launches sshd with one host key per ssh-keygen type in
// hostKeyTypes (default "ed25519") and stops it when the test ends.
func startSSHD(t *testing.T, hostKeyTypes ...string) *testSSHD {
	t.Helper()
	sshd := findSSHD(t)
	if len(hostKeyTypes) == 0 {
		hostKeyTypes = []string{"ed25519"}
	}

	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	s := &testSSHD{User: u.Username, Dir: dir, HostPubKeys: map[string]string{}}

	var hostKeyLines []string
	for _, kt := range hostKeyTypes {
		path := filepath.Join(dir, "host_"+kt)
		args := []string{"-q", "-N", "", "-t", kt, "-f", path}
		if kt == "rsa" {
			args = append(args, "-b", "3072")
		}
		if out, err := exec.Command("ssh-keygen", args...).CombinedOutput(); err != nil {
			t.Fatalf("ssh-keygen %v: %v: %s", args, err, out)
		}
		pub, err := os.ReadFile(path + ".pub")
		if err != nil {
			t.Fatal(err)
		}
		fields := strings.Fields(string(pub))
		s.HostPubKeys[kt] = fields[0] + " " + fields[1]
		hostKeyLines = append(hostKeyLines, "HostKey "+path)
	}

	keys, err := gohpygossh.GenerateKeysForSSH(dir, u.Username)
	if err != nil {
		t.Fatal(err)
	}
	s.ClientKey = keys.PrivKeyAbsPath
	pub, err := os.ReadFile(keys.PublicKeyAbsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "authorized_keys"), pub, 0o600); err != nil { //nolint:gosec // path is under t.TempDir()
		t.Fatal(err)
	}

	tmpl, err := os.ReadFile(filepath.Join("testdata", "sshd_config.in"))
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "sshd_config")

	var lastLog string
	for range 2 {
		port := freePort(t)
		cfg := strings.NewReplacer(
			"@PORT@", strconv.Itoa(port),
			"@DIR@", dir,
			"@HOST_KEY_LINES@", strings.Join(hostKeyLines, "\n"),
		).Replace(string(tmpl))
		if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil { //nolint:gosec // path is under t.TempDir()
			t.Fatal(err)
		}
		if out, err := exec.Command(sshd, "-t", "-f", cfgPath).CombinedOutput(); err != nil {
			t.Fatalf("sshd -t: %v: %s", err, out)
		}
		// OpenSSH >= 9.8 penalises sources whose connections never
		// authenticate (the banner probe, rejected host keys), which would
		// make later connections in a test flaky. Older sshd rejects the
		// option, so only pass it where it is understood.
		args := []string{"-D", "-e", "-f", cfgPath}
		if exec.Command(sshd, "-t", "-f", cfgPath, "-o", "PerSourcePenalties=no").Run() == nil {
			args = append(args, "-o", "PerSourcePenalties=no")
		}

		var log bytes.Buffer
		cmd := exec.Command(sshd, args...)
		cmd.Stderr = &log
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		if waitForSSHBanner(port, 10*time.Second) {
			s.Port = port
			s.Addr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
			t.Cleanup(func() {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				if t.Failed() {
					t.Logf("sshd log:\n%s", log.String())
				}
			})
			return s
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		lastLog = log.String()
	}
	t.Fatalf("sshd did not start; last log:\n%s", lastLog)
	return nil
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForSSHBanner(port int, timeout time.Duration) bool {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 8)
			n, _ := conn.Read(buf)
			conn.Close()
			if string(buf[:n]) == "SSH-2.0-" {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// knownHostsLine returns the known_hosts entry for this server's keyType host key.
func (s *testSSHD) knownHostsLine(keyType string) string {
	return fmt.Sprintf("[127.0.0.1]:%d %s", s.Port, s.HostPubKeys[keyType])
}

// writeKnownHosts writes entries for keyTypes (default: every host key) to path.
func (s *testSSHD) writeKnownHosts(t *testing.T, path string, keyTypes ...string) {
	t.Helper()
	if len(keyTypes) == 0 {
		for kt := range s.HostPubKeys {
			keyTypes = append(keyTypes, kt)
		}
	}
	var b strings.Builder
	for _, kt := range keyTypes {
		b.WriteString(s.knownHostsLine(kt))
		b.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Validates the harness itself against the reference client: if OpenSSH's
// own `ssh` accepts our known_hosts line with StrictHostKeyChecking=yes, the
// lines the harness writes are correct, so later tests compare our library's
// behavior with OpenSSH's.
func TestHarnessKnownHostsAcceptedByOpenSSH(t *testing.T) {
	s := startSSHD(t)
	kh := filepath.Join(s.Dir, "known_hosts")
	s.writeKnownHosts(t, kh)
	out, err := exec.Command("ssh", "-F", "/dev/null", //nolint:gosec // args come from this test's own sshd and t.TempDir()
		"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile="+kh, "-o", "GlobalKnownHostsFile=/dev/null",
		"-i", s.ClientKey, "-p", strconv.Itoa(s.Port),
		s.User+"@127.0.0.1", "echo harness-ok").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "harness-ok" {
		t.Fatalf("%v: %s", err, out)
	}
}

// Run reaches a non-22 port via "host:port". HOME points at a dir whose
// .ssh/known_hosts trusts the server, so this test keeps passing once host
// key verification is mandatory.
func TestRunHostPort(t *testing.T) {
	s := startSSHD(t)
	home := t.TempDir()
	s.writeKnownHosts(t, filepath.Join(home, ".ssh", "known_hosts"))
	t.Setenv("HOME", home)
	out, err := gohpygossh.Run(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok")
	if err != nil || strings.TrimSpace(out) != "ohpygossh-ok" {
		t.Fatalf("%v: %q", err, out)
	}
}
