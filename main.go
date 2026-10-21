package gohpygossh

/*
Main based on
https://gist.github.com/iamralch/b7f56afc966a6b6ac2fc
*/
import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func Hello() string {
	return "Hello, world"
}

type KeysForSSH struct {
	CloudUser        string
	PrivKeyAbsPath   string
	PublicKeyAbsPath string
}

func GenerateKeysForSSH(destinationDir, cloudUser string) (KeysForSSH, error) {
	fmt.Printf("Generating keypair for user: %s", cloudUser)

	temp_file, err := os.CreateTemp(destinationDir, "id_rsa_test")
	if err != nil {
		return KeysForSSH{}, err
	}
	privateKeyFileName := temp_file.Name()
	temp_file.Close()
	os.Remove(privateKeyFileName)

	if err := exec.Command("ssh-keygen", "-b", "2048", "-t", "rsa", "-q", "-N", "", "-f", privateKeyFileName).Run(); err != nil {
		return KeysForSSH{}, err
	}

	publicKeyAbsPath, err := filepath.Abs(privateKeyFileName + ".pub")
	if err != nil {
		return KeysForSSH{}, err
	}

	privateKeyAbsPath, err := filepath.Abs(privateKeyFileName)
	if err != nil {
		return KeysForSSH{}, err
	}

	return KeysForSSH{
		CloudUser:        cloudUser,
		PublicKeyAbsPath: publicKeyAbsPath,
		PrivKeyAbsPath:   privateKeyAbsPath,
	}, nil
}

type KeysAndInit struct {
	CloudInitPath string
	SshKeyPath    string
	CloudUser     string
}

func GenerateKeyPairAndCloudInit(destinationDir, cloudUser string) (KeysAndInit, error) {
	fmt.Printf("Generating keypair & cloud-init for user: %s", cloudUser)

	temp_file, err := os.CreateTemp(destinationDir, "id_rsa_test")
	if err != nil {
		return KeysAndInit{}, err
	}
	privateKeyFileName := temp_file.Name()
	temp_file.Close()
	os.Remove(privateKeyFileName)

	if err := exec.Command("ssh-keygen", "-b", "2048", "-t", "rsa", "-q", "-N", "", "-f", privateKeyFileName).Run(); err != nil {
		return KeysAndInit{}, err
	}

	publicKey, err := os.ReadFile(privateKeyFileName + ".pub")
	if err != nil {
		return KeysAndInit{}, err
	}

	cloud_init_file_path := fmt.Sprintf("%s/cloud-init.yaml", destinationDir)
	cloudInitFile, err := os.Create(cloud_init_file_path)
	if err != nil {
		return KeysAndInit{}, err
	}
	defer cloudInitFile.Close()

	cloudInitFile.WriteString(`
users:
- name: ` + cloudUser + `
  groups: users, admin
  sudo: ALL=(ALL) NOPASSWD:ALL
  ssh_authorized_keys:
    - ` + strings.TrimSpace(string(publicKey)) + `
`)

	cloud_init_abs_path, err := filepath.Abs(cloudInitFile.Name())
	if err != nil {
		return KeysAndInit{}, err
	}

	privateKeyAbsPath, err := filepath.Abs(privateKeyFileName)
	if err != nil {
		return KeysAndInit{}, err
	}

	return KeysAndInit{
		CloudInitPath: cloud_init_abs_path,
		SshKeyPath:    privateKeyAbsPath,
		CloudUser:     cloudUser,
	}, nil
}

func GenerateShortUUID(length int) (string, error) {
	// Generate random bytes
	randomBytes := make([]byte, (length*3+3)/4)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Base64 encode without padding
	encodedString := base64.RawURLEncoding.EncodeToString(randomBytes)

	// Truncate to desired length and remove trailing '=' padding
	return encodedString[:length], nil
}

// GenerateRandomHostname returns a valid hostname of the form "ohpyvm-XXXX"
// where XXXX is four lowercase alphanumeric characters derived from a random
// UUID. Designed for multipass VM naming (which forbids consecutive hyphens,
// leading/trailing hyphens, and non-alphanumeric characters), but may serve
// other use cases that require a short, safe, random hostname.
func GenerateRandomHostname() (string, error) {
	raw, err := GenerateShortUUID(8)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, c := range strings.ToLower(raw) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
			if b.Len() == 4 {
				break
			}
		}
	}
	for b.Len() < 4 {
		b.WriteRune('a')
	}
	return "ohpyvm-" + b.String(), nil
}

func PublicKeyFile(file string) ssh.AuthMethod {
	keyBytes, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

// splitHostPort splits "host:port" or "[host]:port"; a hostname without a
// port (including "::1" or "[::1]") gets port 22. The host is lowercased, as
// OpenSSH does before matching known_hosts.
func splitHostPort(hostname string) (host, port string) {
	host, port, err := net.SplitHostPort(hostname)
	if err != nil {
		host, port = strings.TrimSuffix(strings.TrimPrefix(hostname, "["), "]"), "22"
	}
	return strings.ToLower(host), port
}

// newSSHClient dials an SSH connection using private-key authentication,
// trusting the server's host key only as policy allows.
func newSSHClient(hostname, username, privateKey string, policy *HostKeyPolicy) (*ssh.Client, error) {
	addr := net.JoinHostPort(splitHostPort(hostname))
	hostKeyCallback, hostKeyAlgorithms, err := policy.clientConfig(addr)
	if err != nil {
		return nil, err
	}
	keyBytes, err := os.ReadFile(privateKey)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:              username,
		Auth:              []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: hostKeyAlgorithms,
		Timeout:           60 * time.Second,
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	var hkErr *hostKeyError
	if errors.As(err, &hkErr) {
		return nil, hkErr
	}
	// The server offers none of the key types trusted for it, so no key
	// could be checked: report that as the mismatch it is.
	var negErr *ssh.AlgorithmNegotiationError
	if errors.As(err, &negErr) && negErr.What == "host key" && hostKeyAlgorithms != nil {
		return nil, fmt.Errorf(
			"ohpygossh: host key mismatch for %s: server offers %v host keys, but only %v are trusted for it; possible man-in-the-middle attack",
			knownhosts.Normalize(addr), negErr.RequestedAlgorithms, hostKeyAlgorithms,
		)
	}
	return client, err
}

// Run executes command on hostname ("host" or "host:port"), verifying the
// server against ~/.ssh/known_hosts. See RunWithPolicy for other policies.
func Run(hostname, username, privateKey, command string) (string, error) {
	return RunWithPolicy(hostname, username, privateKey, command, nil)
}

// RunWithPolicy is Run with an explicit host key policy; nil means the default.
func RunWithPolicy(hostname, username, privateKey, command string, policy *HostKeyPolicy) (string, error) {
	client, err := newSSHClient(hostname, username, privateKey, policy)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	err = session.Run(command)
	return b.String(), err
}

// Upload copies localPath to remotePath over SFTP, verifying the server
// against ~/.ssh/known_hosts. See UploadWithPolicy for other policies.
func Upload(hostname, username, privateKey, localPath, remotePath string) error {
	return UploadWithPolicy(hostname, username, privateKey, localPath, remotePath, nil)
}

// UploadWithPolicy is Upload with an explicit host key policy; nil means the default.
func UploadWithPolicy(hostname, username, privateKey, localPath, remotePath string, policy *HostKeyPolicy) error {
	client, err := newSSHClient(hostname, username, privateKey, policy)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return err
	}

	if _, err = io.Copy(remoteFile, localFile); err != nil {
		remoteFile.Close()
		_ = sftpClient.Remove(remotePath)
		return err
	}
	return remoteFile.Close()
}

// Download copies remotePath to localPath over SFTP, verifying the server
// against ~/.ssh/known_hosts. See DownloadWithPolicy for other policies.
func Download(hostname, username, privateKey, remotePath, localPath string) error {
	return DownloadWithPolicy(hostname, username, privateKey, remotePath, localPath, nil)
}

// DownloadWithPolicy is Download with an explicit host key policy; nil means the default.
func DownloadWithPolicy(hostname, username, privateKey, remotePath, localPath string, policy *HostKeyPolicy) error {
	client, err := newSSHClient(hostname, username, privateKey, policy)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}

	if _, err = io.Copy(localFile, remoteFile); err != nil {
		localFile.Close()
		_ = os.Remove(localPath)
		return err
	}
	return localFile.Close()
}
