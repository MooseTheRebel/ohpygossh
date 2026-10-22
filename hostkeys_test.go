package gohpygossh_test

import (
	"bytes"
	"fmt"
	"gohpygossh"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// freshPubKey returns a newly generated, unrelated "<algo> <base64>" key.
func freshPubKey(t *testing.T, keyType string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "other_"+keyType)
	if out, err := exec.Command("ssh-keygen", "-q", "-N", "", "-t", keyType, "-f", path).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v: %s", err, out)
	}
	pub, err := os.ReadFile(path + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(pub))
	return fields[0] + " " + fields[1]
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runWithKnownHosts(s *testSSHD, kh string) (string, error) {
	return gohpygossh.RunWithPolicy(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok",
		&gohpygossh.HostKeyPolicy{KnownHostsPath: kh})
}

func wantOK(t *testing.T, out string, err error) {
	t.Helper()
	if err != nil || strings.TrimSpace(out) != "ohpygossh-ok" {
		t.Fatalf("want success, got err=%v out=%q", err, out)
	}
}

func wantErrContaining(t *testing.T, err error, subs ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want error containing %q, got nil", subs)
	}
	for _, sub := range subs {
		if !strings.Contains(err.Error(), sub) {
			t.Fatalf("error %q does not contain %q", err, sub)
		}
	}
}

func TestKnownHostAccepted(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	s.writeKnownHosts(t, kh)
	out, err := runWithKnownHosts(s, kh)
	wantOK(t, out, err)
}

func TestUnknownHostRejected(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, fmt.Sprintf("[127.0.0.1]:%d %s\n", s.Port+1, s.HostPubKeys["ed25519"]))
	_, err := runWithKnownHosts(s, kh)
	wantErrContaining(t, err, fmt.Sprintf("unknown host [127.0.0.1]:%d", s.Port), "SHA256:")
}

func TestChangedHostKeyRejected(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, fmt.Sprintf("[127.0.0.1]:%d %s\n", s.Port, freshPubKey(t, "ed25519")))
	_, err := runWithKnownHosts(s, kh)
	wantErrContaining(t, err, "host key mismatch", "man-in-the-middle")
}

func TestMissingKnownHostsFileRejected(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "does-not-exist", "known_hosts")
	_, err := runWithKnownHosts(s, kh)
	wantErrContaining(t, err, "cannot read known_hosts file", kh)
}

func TestHashedKnownHostsAccepted(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	s.writeKnownHosts(t, kh)
	if out, err := exec.Command("ssh-keygen", "-H", "-f", kh).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen -H: %v: %s", err, out)
	}
	content, err := os.ReadFile(kh)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(content, []byte("|1|")) {
		t.Fatalf("known_hosts was not hashed: %s", content)
	}
	out, err := runWithKnownHosts(s, kh)
	wantOK(t, out, err)
}

func TestKnownHostsSubsetOfKeyTypesAccepted(t *testing.T) {
	t.Parallel()
	s := startSSHD(t, "rsa", "ecdsa", "ed25519")
	for _, kt := range []string{"rsa", "ecdsa", "ed25519"} {
		t.Run(kt, func(t *testing.T) {
			kh := filepath.Join(t.TempDir(), "known_hosts")
			s.writeKnownHosts(t, kh, kt)
			out, err := runWithKnownHosts(s, kh)
			wantOK(t, out, err)
		})
	}
}

func TestBareHostEntryDoesNotMatchNonDefaultPort(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, "127.0.0.1 "+s.HostPubKeys["ed25519"]+"\n")
	_, err := runWithKnownHosts(s, kh)
	wantErrContaining(t, err, "unknown host")
}

func TestKnownHostByNameAccepted(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, fmt.Sprintf("[localhost]:%d %s\n", s.Port, s.HostPubKeys["ed25519"]))
	out, err := gohpygossh.RunWithPolicy(fmt.Sprintf("localhost:%d", s.Port), s.User, s.ClientKey,
		"echo ohpygossh-ok", &gohpygossh.HostKeyPolicy{KnownHostsPath: kh})
	wantOK(t, out, err)
}

func TestDefaultPolicyUsesHomeKnownHosts(t *testing.T) {
	s := startSSHD(t)
	gohpygossh.SetGlobalKnownHostsPath(t, filepath.Join(t.TempDir(), "absent"))
	home := t.TempDir()
	kh := filepath.Join(home, ".ssh", "known_hosts")
	s.writeKnownHosts(t, kh)
	t.Setenv("HOME", home)

	out, err := gohpygossh.Run(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok")
	wantOK(t, out, err)

	if err := os.Remove(kh); err != nil {
		t.Fatal(err)
	}
	_, err = gohpygossh.Run(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok")
	wantErrContaining(t, err, "cannot read known_hosts file", kh)
}

func TestUploadDownloadWithPolicy(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	s.writeKnownHosts(t, kh)
	pol := &gohpygossh.HostKeyPolicy{KnownHostsPath: kh}

	local := filepath.Join(t.TempDir(), "up.bin")
	want := []byte("ohpygossh-sftp\x00\xff round trip\n")
	writeFile(t, local, string(want))
	remote := filepath.Join(s.Dir, "uploaded.bin")
	if err := gohpygossh.UploadWithPolicy(s.Addr, s.User, s.ClientKey, local, remote, pol); err != nil {
		t.Fatal(err)
	}
	back := filepath.Join(t.TempDir(), "down.bin")
	if err := gohpygossh.DownloadWithPolicy(s.Addr, s.User, s.ClientKey, remote, back, pol); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(back)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("round trip mismatch: got %q want %q", got, want)
	}
}

func TestUploadToUnknownHostWritesNothing(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, "")
	local := filepath.Join(t.TempDir(), "up.txt")
	writeFile(t, local, "must not arrive\n")
	remote := filepath.Join(s.Dir, "should-not-exist.txt")

	err := gohpygossh.UploadWithPolicy(s.Addr, s.User, s.ClientKey, local, remote,
		&gohpygossh.HostKeyPolicy{KnownHostsPath: kh})
	wantErrContaining(t, err, "unknown host")
	if _, statErr := os.Stat(remote); !os.IsNotExist(statErr) {
		t.Fatalf("remote file exists after rejected upload (stat err: %v)", statErr)
	}
}

// fingerprint returns the "SHA256:..." fingerprint of a public key file.
func fingerprint(t *testing.T, pubPath string) string {
	t.Helper()
	out, err := exec.Command("ssh-keygen", "-lf", pubPath).Output()
	if err != nil {
		t.Fatalf("ssh-keygen -lf: %v", err)
	}
	return strings.Fields(string(out))[1]
}

func TestFingerprintPinAccepted(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	fp := fingerprint(t, filepath.Join(s.Dir, "host_ed25519.pub"))
	out, err := gohpygossh.RunWithPolicy(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok",
		&gohpygossh.HostKeyPolicy{Fingerprint: fp})
	wantOK(t, out, err)
}

func TestFingerprintPinRejected(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	other := filepath.Join(t.TempDir(), "other.pub")
	writeFile(t, other, freshPubKey(t, "ed25519")+"\n")
	_, err := gohpygossh.RunWithPolicy(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok",
		&gohpygossh.HostKeyPolicy{Fingerprint: fingerprint(t, other)})
	wantErrContaining(t, err, "host key mismatch", "expected SHA256:")
}

// policyErrorBeforeDial runs policy against a port nothing listens on, so an
// error that mentions the connection proves validation did not happen first.
func policyErrorBeforeDial(t *testing.T, policy *gohpygossh.HostKeyPolicy) error {
	t.Helper()
	keys, err := gohpygossh.GenerateKeysForSSH(t.TempDir(), "u")
	if err != nil {
		t.Fatal(err)
	}
	_, err = gohpygossh.RunWithPolicy("127.0.0.1:1", "u", keys.PrivKeyAbsPath, "true", policy)
	if err != nil && strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("policy was not validated before dialing: %v", err)
	}
	return err
}

func TestFingerprintMalformedRejected(t *testing.T) {
	t.Parallel()
	err := policyErrorBeforeDial(t, &gohpygossh.HostKeyPolicy{Fingerprint: "deadbeef"})
	wantErrContaining(t, err, `must look like "SHA256:..."`)
}

func TestInsecureOptInAccepted(t *testing.T) {
	s := startSSHD(t)
	t.Setenv("HOME", t.TempDir())
	out, err := gohpygossh.RunWithPolicy(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok",
		&gohpygossh.HostKeyPolicy{InsecureIgnoreHostKey: true})
	wantOK(t, out, err)
}

func TestPolicyConflictRejected(t *testing.T) {
	t.Parallel()
	for name, policy := range map[string]*gohpygossh.HostKeyPolicy{
		"knownhosts+fingerprint": {KnownHostsPath: "/dev/null", Fingerprint: "SHA256:x"},
		"knownhosts+insecure":    {KnownHostsPath: "/dev/null", InsecureIgnoreHostKey: true},
		"fingerprint+insecure":   {Fingerprint: "SHA256:x", InsecureIgnoreHostKey: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			wantErrContaining(t, policyErrorBeforeDial(t, policy), "set at most one of")
		})
	}
}

// A stock server (e.g. Ubuntu) has RSA, ECDSA and Ed25519 host keys and
// presents ECDSA by default; pinning its Ed25519 fingerprint, as the README
// suggests, must still connect.
func TestFingerprintPinOfEd25519KeyAcceptedWhenServerHasSeveralKeys(t *testing.T) {
	t.Parallel()
	s := startSSHD(t, "rsa", "ecdsa", "ed25519")
	fp := fingerprint(t, filepath.Join(s.Dir, "host_ed25519.pub"))
	out, err := gohpygossh.RunWithPolicy(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok",
		&gohpygossh.HostKeyPolicy{Fingerprint: fp})
	wantOK(t, out, err)
}

func TestDefaultPolicyUsesGlobalKnownHosts(t *testing.T) {
	s := startSSHD(t)
	t.Setenv("HOME", t.TempDir()) // no ~/.ssh/known_hosts
	global := filepath.Join(t.TempDir(), "ssh_known_hosts")
	s.writeKnownHosts(t, global)
	gohpygossh.SetGlobalKnownHostsPath(t, global)
	out, err := gohpygossh.Run(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok")
	wantOK(t, out, err)
}

func TestDefaultPolicyMergesUserAndGlobalKnownHosts(t *testing.T) {
	s := startSSHD(t)
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".ssh", "known_hosts"), "unrelated.example.com "+freshPubKey(t, "ed25519")+"\n")
	t.Setenv("HOME", home)
	global := filepath.Join(t.TempDir(), "ssh_known_hosts")
	s.writeKnownHosts(t, global)
	gohpygossh.SetGlobalKnownHostsPath(t, global)
	out, err := gohpygossh.Run(s.Addr, s.User, s.ClientKey, "echo ohpygossh-ok")
	wantOK(t, out, err)
}

// OpenSSH lowercases hostnames when it writes and matches known_hosts.
func TestHostnameMatchIsCaseInsensitive(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, fmt.Sprintf("[localhost]:%d %s\n", s.Port, s.HostPubKeys["ed25519"]))
	out, err := gohpygossh.RunWithPolicy(fmt.Sprintf("LocalHost:%d", s.Port), s.User, s.ClientKey,
		"echo ohpygossh-ok", &gohpygossh.HostKeyPolicy{KnownHostsPath: kh})
	wantOK(t, out, err)
}

// known_hosts lists only an RSA key, but the server offers only Ed25519.
func TestNoCommonHostKeyTypeReportedAsMismatch(t *testing.T) {
	t.Parallel()
	s := startSSHD(t)
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, fmt.Sprintf("[127.0.0.1]:%d %s\n", s.Port, freshPubKey(t, "rsa")))
	_, err := runWithKnownHosts(s, kh)
	wantErrContaining(t, err, "host key mismatch", "man-in-the-middle")
	if strings.Contains(err.Error(), "handshake failed") {
		t.Fatalf("raw handshake error leaked: %v", err)
	}
}

// The mismatch error must cite the known_hosts line of the presented key's
// type, not merely the first line for the host.
func TestMismatchCitesLineOfPresentedKeyType(t *testing.T) {
	t.Parallel()
	s := startSSHD(t) // presents Ed25519
	kh := filepath.Join(t.TempDir(), "known_hosts")
	writeFile(t, kh, fmt.Sprintf("[127.0.0.1]:%d %s\n[127.0.0.1]:%d %s\n",
		s.Port, freshPubKey(t, "ecdsa"), s.Port, freshPubKey(t, "ed25519")))
	_, err := runWithKnownHosts(s, kh)
	wantErrContaining(t, err, "host key mismatch", kh+":2;")
}
