package gohpygossh

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyPolicy decides which SSH server host keys are trusted. A nil policy,
// or the zero value, verifies the server against ~/.ssh/known_hosts.
type HostKeyPolicy struct {
	// KnownHostsPath is an OpenSSH known_hosts file; "" means ~/.ssh/known_hosts.
	KnownHostsPath string
	// Fingerprint pins the server's host key, as printed by `ssh-keygen -lf`
	// ("SHA256:...").
	Fingerprint string
	// InsecureIgnoreHostKey accepts any host key, which leaves connections
	// open to man-in-the-middle attacks. Use it only for testing.
	InsecureIgnoreHostKey bool
}

// globalKnownHostsPath is the system-wide known_hosts file that OpenSSH
// also reads; the default policy checks it alongside ~/.ssh/known_hosts.
var globalKnownHostsPath = "/etc/ssh/ssh_known_hosts"

// pinnedAlgorithms is the host key preference when pinning a fingerprint,
// which does not say which of the server's keys it belongs to. Ed25519 comes
// first: servers that have one present it, matching the README's advice to
// pin it; x/crypto's default order would ask for ECDSA first instead.
var pinnedAlgorithms = []string{
	ssh.KeyAlgoED25519,
	ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521,
	ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA,
}

// hostKeyError reports a rejected host key. newSSHClient returns it unwrapped
// so callers see why, rather than "ssh: handshake failed: ...".
type hostKeyError struct{ msg string }

func (e *hostKeyError) Error() string { return e.msg }

// clientConfig returns the host key callback, and the host key algorithms to
// request, for a connection to addr ("host:port").
func (p *HostKeyPolicy) clientConfig(addr string) (ssh.HostKeyCallback, []string, error) {
	if p == nil {
		p = &HostKeyPolicy{}
	}
	set := 0
	for _, isSet := range []bool{p.KnownHostsPath != "", p.Fingerprint != "", p.InsecureIgnoreHostKey} {
		if isSet {
			set++
		}
	}
	if set > 1 {
		return nil, nil, errors.New("ohpygossh: HostKeyPolicy: set at most one of KnownHostsPath, Fingerprint, InsecureIgnoreHostKey")
	}

	if p.InsecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil, nil //nolint:gosec // explicit caller opt-in via HostKeyPolicy.InsecureIgnoreHostKey
	}

	if p.Fingerprint != "" {
		if !strings.HasPrefix(p.Fingerprint, "SHA256:") {
			return nil, nil, errors.New(`ohpygossh: HostKeyPolicy.Fingerprint must look like "SHA256:..."`)
		}
		return func(hostname string, _ net.Addr, key ssh.PublicKey) error {
			if got := ssh.FingerprintSHA256(key); got != p.Fingerprint {
				return &hostKeyError{fmt.Sprintf(
					"ohpygossh: host key mismatch for %s: server presented %s key %s, expected %s; possible man-in-the-middle attack",
					knownhosts.Normalize(hostname), key.Type(), got, p.Fingerprint,
				)}
			}
			return nil
		}, pinnedAlgorithms, nil
	}

	files, err := p.knownHostsFiles()
	if err != nil {
		return nil, nil, err
	}
	check, err := knownhosts.New(files...)
	if err != nil {
		return nil, nil, fmt.Errorf("ohpygossh: cannot read known_hosts file %s: %w", strings.Join(files, ", "), err)
	}
	path := strings.Join(files, " or ")

	callback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := check(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return &hostKeyError{fmt.Sprintf("ohpygossh: %v", err)}
		}
		host := knownhosts.Normalize(hostname)
		if len(keyErr.Want) == 0 {
			return &hostKeyError{fmt.Sprintf(
				"ohpygossh: unknown host %s: server presented %s key %s, which is not in %s",
				host, key.Type(), ssh.FingerprintSHA256(key), path,
			)}
		}
		// Cite the entry the presented key was compared with, if any.
		want := keyErr.Want[0]
		for _, w := range keyErr.Want {
			if w.Key.Type() == key.Type() {
				want = w
				break
			}
		}
		return &hostKeyError{fmt.Sprintf(
			"ohpygossh: host key mismatch for %s: server presented %s key %s, which does not match %s:%d; possible man-in-the-middle attack",
			host, key.Type(), ssh.FingerprintSHA256(key), want.Filename, want.Line,
		)}
	}
	return callback, knownAlgorithms(check, addr), nil
}

// knownHostsFiles returns the known_hosts files to check: KnownHostsPath
// alone, or else ~/.ssh/known_hosts and the system-wide file, as OpenSSH
// does, leaving out whichever does not exist.
func (p *HostKeyPolicy) knownHostsFiles() ([]string, error) {
	if p.KnownHostsPath != "" {
		return []string{p.KnownHostsPath}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("ohpygossh: cannot locate known_hosts file: %w", err)
	}
	user := filepath.Join(home, ".ssh", "known_hosts")
	var files []string
	for _, f := range []string{user, globalKnownHostsPath} {
		if _, err := os.Stat(f); !errors.Is(err, fs.ErrNotExist) {
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("ohpygossh: cannot read known_hosts file %s: %w", user, fs.ErrNotExist)
	}
	return files, nil
}

// knownAlgorithms lists the host key algorithms known_hosts holds for addr,
// so the server is asked for a key type that can be checked. Otherwise a
// server may present, say, its ECDSA key while known_hosts lists only its
// Ed25519 key, which knownhosts reports as a mismatch. It returns nil for an
// unknown host, leaving the server's default choice to be reported.
func knownAlgorithms(check ssh.HostKeyCallback, addr string) []string {
	// An all-zero Ed25519 key matches no entry, so the KeyError lists every
	// key known for addr. knownhosts ignores remote when addr is given.
	probe, err := ssh.NewPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		return nil
	}
	var keyErr *knownhosts.KeyError
	if !errors.As(check(addr, &net.TCPAddr{}, probe), &keyErr) {
		return nil
	}
	var algos []string
	seen := map[string]bool{}
	for _, known := range keyErr.Want {
		types := []string{known.Key.Type()}
		if types[0] == ssh.KeyAlgoRSA {
			types = []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA}
		}
		for _, t := range types {
			if !seen[t] {
				seen[t] = true
				algos = append(algos, t)
			}
		}
	}
	return algos
}
