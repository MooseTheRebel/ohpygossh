package gohpygossh_test

import (
	"fmt"
	"gohpygossh"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larstobi/go-multipass/multipass"
)

func TestHello(t *testing.T) {
	got := gohpygossh.Hello()
	want := "Hello, world"
	if got != want {
		t.Error("Unexpected value")
	}
}

func TestGenerateKeyPairAndCloudInit(t *testing.T) {
	// Create a temporary directory for multipass operations
	tmpDir, err := os.MkdirTemp("", "g-opg-test")
	if err != nil {
		t.Fatal(err)
	}

	r, err := gohpygossh.GenerateKeyPairAndCloudInit(tmpDir, "cloud-user")
	if err != nil {
		t.Fatal(err)
	}

	result, _ := os.ReadDir(tmpDir)
	gotFiles := len(result)
	// Directory should include key pair (2 files) and 1 cloud init file
	if gotFiles != 3 {
		t.Error("Unexpected value")
	}

	want := "id_rsa"
	if !strings.Contains(r.SshKeyPath, want) {
		t.Error("Unexpected value for 'ssh_key_abs_path")
	}

	want_cloud_init := "cloud-init.yaml"
	if !strings.Contains(r.CloudInitPath, want_cloud_init) {
		t.Error("Unexpected value for 'cloud_init_abs_path's")
	}

	defer os.RemoveAll(tmpDir)
}

func TestGenerateKeysForSSH(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "keys-only-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	r, err := gohpygossh.GenerateKeysForSSH(tmpDir, "test-user")
	if err != nil {
		t.Fatal(err)
	}

	if r.CloudUser != "test-user" {
		t.Errorf("unexpected CloudUser: %s", r.CloudUser)
	}

	if !strings.Contains(r.PrivKeyAbsPath, "id_rsa") {
		t.Errorf("unexpected PrivKeyAbsPath: %s", r.PrivKeyAbsPath)
	}

	if !strings.Contains(r.PublicKeyAbsPath, ".pub") {
		t.Errorf("unexpected PublicKeyAbsPath: %s", r.PublicKeyAbsPath)
	}

	if _, err := os.Stat(r.PrivKeyAbsPath); err != nil {
		t.Errorf("private key file not found: %s", r.PrivKeyAbsPath)
	}

	if _, err := os.Stat(r.PublicKeyAbsPath); err != nil {
		t.Errorf("public key file not found: %s", r.PublicKeyAbsPath)
	}
}

func TestGenerateShortUUID(t *testing.T) {
	for _, length := range []int{2, 4, 8, 16} {
		t.Run(fmt.Sprintf("length-%d", length), func(t *testing.T) {
			id, err := gohpygossh.GenerateShortUUID(length)
			if err != nil {
				t.Fatalf("GenerateShortUUID(%d) failed: %v", length, err)
			}
			if len(id) != length {
				t.Errorf("GenerateShortUUID(%d) = %q; want length %d", length, id, length)
			}
		})
	}

	id1, _ := gohpygossh.GenerateShortUUID(10)
	id2, _ := gohpygossh.GenerateShortUUID(10)
	if id1 == id2 {
		t.Errorf("expected unique values from consecutive calls, both were %q", id1)
	}
}

func TestGenerateRandomHostname(t *testing.T) {
	const prefix = "ohpyvm-"

	for i := 0; i < 20; i++ {
		name, err := gohpygossh.GenerateRandomHostname()
		if err != nil {
			t.Fatalf("GenerateRandomHostname() failed: %v", err)
		}
		if !strings.HasPrefix(name, prefix) {
			t.Errorf("VM name %q missing expected prefix %q", name, prefix)
		}
		suffix := strings.TrimPrefix(name, prefix)
		if len(suffix) != 4 {
			t.Errorf("VM name suffix %q is not exactly 4 chars", suffix)
		}
		for _, c := range suffix {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
				t.Errorf("suffix %q contains non-alphanumeric character %q", suffix, c)
			}
		}
	}

	name1, _ := gohpygossh.GenerateRandomHostname()
	name2, _ := gohpygossh.GenerateRandomHostname()
	if name1 == name2 {
		t.Errorf("expected unique VM names, both were %q", name1)
	}
}

func TestPublicKeyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pubkeyfile-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	r, err := gohpygossh.GenerateKeysForSSH(tmpDir, "test-user")
	if err != nil {
		t.Fatal(err)
	}

	auth := gohpygossh.PublicKeyFile(r.PrivKeyAbsPath)
	if auth == nil {
		t.Error("expected non-nil AuthMethod for valid private key")
	}

	authBad := gohpygossh.PublicKeyFile("/nonexistent/path/to/key")
	if authBad != nil {
		t.Error("expected nil AuthMethod for nonexistent file")
	}
}

func TestGenerateKeyPairAndCloudInit_Content(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cloud-init-content-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cloudUser := "myuser"
	r, err := gohpygossh.GenerateKeyPairAndCloudInit(tmpDir, cloudUser)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(r.CloudInitPath)
	if err != nil {
		t.Fatalf("could not read cloud-init file: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, cloudUser) {
		t.Errorf("cloud-init content missing username %q", cloudUser)
	}

	pubKey, err := os.ReadFile(r.SshKeyPath + ".pub")
	if err != nil {
		t.Fatalf("could not read public key file: %v", err)
	}
	if !strings.Contains(contentStr, strings.TrimSpace(string(pubKey))) {
		t.Error("cloud-init content missing public key")
	}
}

func TestRunWithMultipass(t *testing.T) {
	// Install go-multipass if not already installed
	if _, err := exec.LookPath("multipass"); err != nil {
		t.Skip("multipass not installed")
	}

	// Create a temporary directory for multipass operations
	tmpDir, err := os.MkdirTemp("", "multipass-test2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	r, err := gohpygossh.GenerateKeyPairAndCloudInit(tmpDir, "cloud-user")
	if err != nil {
		t.Fatal(err)
	}

	dynVMName, err := gohpygossh.GenerateRandomHostname()
	if err != nil {
		t.Fatal(err)
	}

	instance, err := multipass.LaunchV2(&multipass.LaunchReqV2{
		Image:         "lts",
		CPUS:          "2",
		Disk:          "10g",
		Name:          dynVMName,
		Memory:        "3g",
		CloudInitFile: r.CloudInitPath,
	})
	if err != nil {
		fmt.Println(err)
		t.Fatal(err)
	}
	// Delete runs "multipass delete --purge", fully removing the VM when the test ends.
	defer func() { _ = multipass.Delete(&multipass.DeleteRequest{Name: dynVMName}) }()

	// Get the hostname of the multipass instance
	fmt.Println(instance.IP)

	info, err := multipass.Info(&multipass.InfoRequest{Name: dynVMName})
	hostname := info.IP
	if err != nil {
		t.Fatal(err)
	}

	// Pin the VM's host key, read over multipass's exec channel rather than
	// SSH, so the connection below is verified. cloud-init may still be
	// generating host keys, so retry briefly.
	var fields []string
	for range 10 {
		var hostPub []byte
		hostPub, err = exec.Command("multipass", "exec", dynVMName, "--",
			"cat", "/etc/ssh/ssh_host_ed25519_key.pub").Output()
		fields = strings.Fields(string(hostPub))
		if err == nil && len(fields) >= 2 {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err != nil || len(fields) < 2 {
		t.Fatalf("reading VM host key: %v (fields: %q)", err, fields)
	}
	kh := filepath.Join(tmpDir, "known_hosts")
	if err := os.WriteFile(kh, []byte(hostname+" "+fields[0]+" "+fields[1]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Execute the 'echo' command on the remote server
	output, err := gohpygossh.RunWithPolicy(hostname, r.CloudUser, r.SshKeyPath, "echo 'Connection success'",
		&gohpygossh.HostKeyPolicy{KnownHostsPath: kh})
	if err != nil {
		t.Fatal(err)
	}

	// Verify the output contains expected files or directories
	if !strings.Contains(output, "Connection success") {
		t.Error("Expected 'Connection success' in output but found:", output)
	}
}
