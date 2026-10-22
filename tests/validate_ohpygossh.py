"""
This file serves as a test of ohpygossh.
"""

import glob
import json
import os
import subprocess
import sys
import tempfile
import time
from os import chdir, listdir
from os.path import join
from pathlib import Path
from shlex import split
from shutil import copy, which

GIT_ROOT = Path(__file__).resolve().parent.parent
DOCKER_IN_VAGRANT = GIT_ROOT / "ssh-servers" / "docker-in-vagrant"


def run_multiple_commands(commands, description_of_command="") -> tuple[str, str]:
    process = subprocess.Popen(
        "/bin/bash", stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True
    )
    out, err = process.communicate(commands)

    print(
        f"""Start multi-command output -- {description_of_command}

STDOUT
{out}

STDERR
{err}

Finished -- {description_of_command}
    """
    )


def get_vagrant_ssh_field(field: str, vagrantfile_dir: Path):
    """Retrieves some 'ssh-confg' from the active Vagrant VM."""
    # Captures the output of vagrant ssh-config
    process = subprocess.Popen(
        ["vagrant", "ssh-config"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        text=True,
        cwd=vagrantfile_dir,
    )

    ssh_config_output, _err = process.communicate()

    # Iterates over lines and searches for the line starting
    # with a given field (e.g. "Port" or "HostName").
    #
    # Note, these fields usually are prefixed with whitespace
    for line in ssh_config_output.splitlines():
        if line.lstrip().startswith(field):
            # Extracts the hostname (second word) and returns it
            return line.split()[1]

    # If no "HostName" line is found, raise an exception
    raise RuntimeError(f"Failed to find {field=} in vagrant ssh-config")


def test_with_keys_only():
    keys_only_user = "keysOnlyUser"
    try:
        print("Python validation starting, for 'test_with_keys_only'")

        from ohpygossh.gohpygossh import GenerateKeysForSSH, KeysForSSH

        with tempfile.TemporaryDirectory() as tmpDir:
            print("'ohpygossh': Generating SSH key pair")
            keys: KeysForSSH = GenerateKeysForSSH(tmpDir, keys_only_user)

            print(
                f"""


Keypair written to disk

{keys.PrivKeyAbsPath=}

{keys.PublicKeyAbsPath=}

"""
            )

            commands = f"""

echo "Gathering SSH public key content 'ohpygossh' public SSH key has content"


# Change directory
cd {DOCKER_IN_VAGRANT}

cat {keys.PublicKeyAbsPath} > new_id_rsa.pub

# List content in the directory
ls -la

# Launch machine
vagrant up --provider=docker

echo "Checking vagrant status"
vagrant status && echo "That's the status"
"""
            run_multiple_commands(
                commands=commands, description_of_command="Vagrant up, etc."
            )

            ssh_hostname = get_vagrant_ssh_field(
                "HostName", vagrantfile_dir=DOCKER_IN_VAGRANT
            )

            ssh_port = get_vagrant_ssh_field("Port", vagrantfile_dir=DOCKER_IN_VAGRANT)
            run_multiple_commands(
                commands=f"""

echo "Testing generated keypair"
echo "Performing authentication as 'vagrant' user"

ssh -o StrictHostKeyChecking=no -i {keys.PrivKeyAbsPath} vagrant@{ssh_hostname} -p {ssh_port} 'echo "We did it!"'

""",
                description_of_command="Open connection, with ssh -i flag and run command",
            )

    except Exception as e:
        raise RuntimeError("An unexpected error occurred testing ohpygossh") from e


def test_say_hello():
    try:
        print("Python validation starting...")

        from ohpygossh.gohpygossh import Hello

        # Prints a basic str value: 'Hello, world'
        print(Hello())
    except Exception as e:
        raise RuntimeError("An unexpected error occurred testing: Hello") from e


def test_with_cloud_init():
    try:
        print("Python validation starting, for 'test_with_cloud_init'")
        from ohpygossh.gohpygossh import GenerateKeyPairAndCloudInit, KeysAndInit

        vagrant_test_dir = GIT_ROOT / "ssh-servers" / "cloud-init"

        with tempfile.TemporaryDirectory() as tmpDir:
            print("Created temporary directory", tmpDir)

            # Copy all files
            for filename in glob.glob(join(vagrant_test_dir, "*.*")):
                copy(filename, tmpDir)

            # Copy the Vagrantifle
            for filename in glob.glob(join(vagrant_test_dir, "Vagrantfile")):
                copy(filename, tmpDir)

            kai: KeysAndInit = GenerateKeyPairAndCloudInit(tmpDir, "validation-user")

            print(listdir(tmpDir))

            # Change the working directory
            chdir(tmpDir)

            # Run vagrant up command
            vm_process = subprocess.Popen(
                ["vagrant", "up"], stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )

            # Wait for the process to finish and capture output
            stdout, stderr = vm_process.communicate()

            # Check for errors
            if stderr:
                print("Error bringing up Vagrant VM:", stderr.decode("utf-8"))
            else:
                print("Vagrant VM successfully brought up.")

            ssh_cmd = f"ssh -o StrictHostKeyChecking=no -i {kai.SshKeyPath} {kai.CloudUser}@192.168.56.10 'echo \"Connection success\"'"
            split_ssh_cmd = split(ssh_cmd)
            ssh_process = subprocess.Popen(
                split_ssh_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE
            )

            stdout, stderr = ssh_process.communicate()

            # Check for errors
            if ssh_process.returncode != 0:
                print("Error running SSH command:", stderr.decode("utf-8"))
            else:
                print(
                    f"""SSH command successfully executed.

SSH result:
{stdout.decode("utf-8")}

"""
                )

    except Exception as e:
        raise RuntimeError("An unexpected error occurred testing ohpygossh") from e


def _multipass_cmd() -> list:
    """On Linux snap installs the multipassd socket is root-owned; prefix with sudo."""
    if sys.platform == "linux" and os.geteuid() != 0 and which("sudo"):
        return ["sudo", "multipass"]
    return ["multipass"]


def _lxc_cmd() -> list:
    """Prefix lxc with sudo if the LXD unix socket isn't accessible as-is.

    The socket's group is configurable (e.g. canonical/setup-lxd defaults to
    "adm", not "lxd"), so checking group membership by name is unreliable;
    probing the actual command is the only check that matches reality.
    """
    if sys.platform == "linux" and os.geteuid() != 0 and which("sudo"):
        if (
            subprocess.run(
                ["lxc", "list"], capture_output=True, timeout=10, check=False
            ).returncode
            != 0
        ):
            return ["sudo", "lxc"]
    return ["lxc"]


def _pin_vm_host_key(ip: str, remote_exec: list, tmp_dir: str) -> str:
    """Write a known_hosts file trusting the VM's Ed25519 host key.

    The key is read through the hypervisor's exec channel (remote_exec, e.g.
    ["lxc", "exec", name, "--"]), not over SSH, so it can be trusted. Only the
    Ed25519 key is pinned although the VM also has RSA/ECDSA keys, which
    checks that ohpygossh asks the server for a key type it can verify.
    """
    for attempt in range(10):
        result = subprocess.run(
            [*remote_exec, "cat", "/etc/ssh/ssh_host_ed25519_key.pub"],
            capture_output=True,
            text=True,
            check=False,
        )
        if result.returncode == 0 and result.stdout.strip():
            break
        print(f"Waiting for VM host key (attempt {attempt + 1}/10)...")
        time.sleep(3)
    else:
        raise RuntimeError(f"Could not read VM host key: {result.stderr!r}")
    algo, key = result.stdout.split()[:2]
    known_hosts = os.path.join(tmp_dir, "known_hosts")
    Path(known_hosts).write_text(f"{ip} {algo} {key}\n")
    return known_hosts


def _expect_host_key_rejected(ip: str, kai, known_hosts: str, phrase: str) -> None:
    """Assert that connecting with known_hosts fails with phrase in the error."""
    from ohpygossh.gohpygossh import HostKeyPolicy, RunWithPolicy

    policy = HostKeyPolicy(KnownHostsPath=known_hosts)
    try:
        RunWithPolicy(ip, kai.CloudUser, kai.SshKeyPath, "true", policy)
    except RuntimeError as e:
        if phrase not in str(e):
            raise RuntimeError(f"Expected {phrase!r} in error, got: {e}") from e
        print(f"Host key correctly rejected ({phrase}): {e}")
        return
    raise RuntimeError(f"host key verification did not reject {ip} ({phrase})")


def _verify_ssh_and_sftp(ip: str, kai, tmp_dir: str, remote_exec: list) -> None:
    """Check host key verification, then run SSH and an SFTP round-trip."""
    from ohpygossh.gohpygossh import (
        DownloadWithPolicy,
        GenerateShortUUID,
        HostKeyPolicy,
        RunWithPolicy,
        UploadWithPolicy,
    )

    # Against the real VM, an unknown host and a wrong key must both be refused.
    empty_known_hosts = os.path.join(tmp_dir, "known_hosts.empty")
    Path(empty_known_hosts).write_text("")
    _expect_host_key_rejected(ip, kai, empty_known_hosts, "unknown host")

    wrong_key = os.path.join(tmp_dir, "wrong_host_key")
    subprocess.run(
        ["ssh-keygen", "-q", "-N", "", "-t", "ed25519", "-f", wrong_key], check=True
    )
    wrong_known_hosts = os.path.join(tmp_dir, "known_hosts.wrong")
    algo, key = Path(f"{wrong_key}.pub").read_text().split()[:2]
    Path(wrong_known_hosts).write_text(f"{ip} {algo} {key}\n")
    _expect_host_key_rejected(ip, kai, wrong_known_hosts, "host key mismatch")

    policy = HostKeyPolicy(KnownHostsPath=_pin_vm_host_key(ip, remote_exec, tmp_dir))

    output = RunWithPolicy(
        ip, kai.CloudUser, kai.SshKeyPath, "echo 'Connection success'", policy
    )

    if "Connection success" not in output:
        raise RuntimeError(f"Unexpected SSH output: {output!r}")

    print(f"Run e2e test passed. Output: {output!r}")

    # SFTP round-trip: upload a file, verify it landed, download it back.
    unique_marker = GenerateShortUUID(8)
    upload_content = f"ohpygossh-sftp-test:{unique_marker}\n"
    local_upload = os.path.join(tmp_dir, "upload.txt")
    remote_path = f"/home/{kai.CloudUser}/uploaded.txt"
    local_download = os.path.join(tmp_dir, "download.txt")

    Path(local_upload).write_text(upload_content)

    UploadWithPolicy(
        ip, kai.CloudUser, kai.SshKeyPath, local_upload, remote_path, policy
    )
    print(f"Upload complete: {local_upload} -> {remote_path}")

    cat_output = RunWithPolicy(
        ip, kai.CloudUser, kai.SshKeyPath, f"cat {remote_path}", policy
    )
    if cat_output.strip() != upload_content.strip():
        raise RuntimeError(
            f"Remote file content mismatch after upload. Got: {cat_output!r}"
        )
    print("Remote file content verified via Run/cat.")

    DownloadWithPolicy(
        ip, kai.CloudUser, kai.SshKeyPath, remote_path, local_download, policy
    )
    print(f"Download complete: {remote_path} -> {local_download}")

    downloaded_content = Path(local_download).read_text()
    if downloaded_content != upload_content:
        raise RuntimeError(
            f"Downloaded content does not match original. Got: {downloaded_content!r}"
        )

    print(f"SFTP round-trip e2e test passed. Marker: {unique_marker!r}")


def test_with_multipass():
    if not which("multipass"):
        print("multipass not installed, skipping test_with_multipass")
        return

    try:
        print("Python validation starting, for 'test_with_multipass'")

        from ohpygossh.gohpygossh import (
            GenerateKeyPairAndCloudInit,
            GenerateRandomHostname,
        )

        vm_name = GenerateRandomHostname()
        mp = _multipass_cmd()

        # multipass snap AppArmor profile allows @{HOME}/** but not /tmp/**;
        # create the temp dir under $HOME so the daemon can read the cloud-init file.
        with tempfile.TemporaryDirectory(dir=Path.home()) as tmpDir:
            kai = GenerateKeyPairAndCloudInit(tmpDir, "cloud-user")

            print(f"Launching multipass VM: {vm_name}")
            try:
                launch_cmd = [
                    *mp,
                    "launch",
                    "lts",
                    "--name",
                    vm_name,
                    "--cpus",
                    "1",
                    "--memory",
                    "1G",
                    "--disk",
                    "5G",
                    "--cloud-init",
                    kai.CloudInitPath,
                ]
                for attempt, wait in enumerate([0, 10, 30, 90]):
                    if wait:
                        print(
                            f"Launch attempt {attempt} failed, retrying in {wait}s..."
                        )
                        time.sleep(wait)
                    if subprocess.run(launch_cmd, check=False).returncode == 0:
                        break
                else:
                    raise RuntimeError("multipass launch failed after 4 attempts")

                info_result = subprocess.run(
                    [*mp, "info", "--format", "json", vm_name],
                    capture_output=True,
                    text=True,
                    check=True,
                )
                info_data = json.loads(info_result.stdout)
                vm_info = info_data.get("info", {}).get(vm_name, {})
                ipv4_addresses = vm_info.get("ipv4", [])
                if not ipv4_addresses:
                    raise RuntimeError(
                        f"No IPv4 address found for VM {vm_name}. Info: {info_data}"
                    )
                ip = ipv4_addresses[0]
                print(f"VM IP: {ip}")

                _verify_ssh_and_sftp(ip, kai, tmpDir, [*mp, "exec", vm_name, "--"])

            finally:
                print(f"Deleting multipass VM: {vm_name}")
                subprocess.run([*mp, "delete", "--purge", vm_name], check=False)

    except Exception as e:
        if isinstance(e, RuntimeError):
            raise
        raise RuntimeError(
            "An unexpected error occurred testing ohpygossh with multipass"
        ) from e


def test_with_lxc():
    if not which("lxc"):
        print("lxc not installed, skipping test_with_lxc")
        return

    try:
        print("Python validation starting, for 'test_with_lxc'")

        from ohpygossh.gohpygossh import (
            GenerateKeyPairAndCloudInit,
            GenerateRandomHostname,
        )

        vm_name = GenerateRandomHostname()
        lxc = _lxc_cmd()

        with tempfile.TemporaryDirectory(dir=Path.home()) as tmpDir:
            kai = GenerateKeyPairAndCloudInit(tmpDir, "cloud-user")
            cloud_init_content = Path(kai.CloudInitPath).read_text()

            # LXD/cloud-init requires the #cloud-config header to recognise the
            # payload as YAML; the Go generator omits it.
            if not cloud_init_content.lstrip().startswith("#cloud-config"):
                cloud_init_content = "#cloud-config\n" + cloud_init_content.lstrip()

            print(f"Launching LXC container: {vm_name}")
            try:
                for attempt, wait in enumerate([0, 10, 30, 90]):
                    if wait:
                        print(
                            f"Launch attempt {attempt} failed, retrying in {wait}s..."
                        )
                        subprocess.run(
                            [*lxc, "delete", "--force", vm_name],
                            stdout=subprocess.DEVNULL,
                            stderr=subprocess.DEVNULL,
                            check=False,
                        )
                        time.sleep(wait)
                    # init + config set + start avoids embedding multi-line YAML
                    # in --config= and uses the modern cloud-init.user-data key.
                    if (
                        subprocess.run(
                            [*lxc, "init", "ubuntu:lts", vm_name], check=False
                        ).returncode
                        != 0
                    ):
                        continue
                    if (
                        subprocess.run(
                            [
                                *lxc,
                                "config",
                                "set",
                                vm_name,
                                "cloud-init.user-data",
                                cloud_init_content,
                            ],
                            check=False,
                        ).returncode
                        != 0
                    ):
                        continue
                    if (
                        subprocess.run([*lxc, "start", vm_name], check=False).returncode
                        == 0
                    ):
                        break
                else:
                    raise RuntimeError("lxc launch failed after 4 attempts")

                # Poll for a routable IPv4 address via lxc list JSON first. This
                # queries LXD's own state without exec'ing into the container, so
                # it works even when the in-container shell is not yet ready and
                # is more robust than running `lxc exec` immediately after start.
                ip = None
                for attempt in range(24):
                    try:
                        list_result = subprocess.run(
                            [*lxc, "list", vm_name, "--format", "json"],
                            capture_output=True,
                            text=True,
                            timeout=10,
                            check=True,
                        )
                        containers = json.loads(list_result.stdout or "[]")
                        for container in containers:
                            if container.get("name") != vm_name:
                                continue
                            network = (container.get("state") or {}).get(
                                "network"
                            ) or {}
                            for info in network.values():
                                for addr in (info or {}).get("addresses", []):
                                    if (
                                        addr.get("family") == "inet"
                                        and addr.get("scope") == "global"
                                    ):
                                        ip = addr.get("address")
                                        break
                                if ip:
                                    break
                        if ip:
                            break
                    except (
                        subprocess.TimeoutExpired,
                        subprocess.CalledProcessError,
                        json.JSONDecodeError,
                    ):
                        pass
                    print(f"Waiting for IP address (attempt {attempt + 1}/24)...")
                    time.sleep(5)

                if not ip:
                    raise RuntimeError(f"No IPv4 address found for container {vm_name}")
                print(f"Container IP: {ip}")

                # Wait for cloud-init to finish provisioning the user and SSH keys.
                # Exit 0 = success; 1 = ran but some modules errored (SSH keys may
                # still be provisioned); 2 = cloud-init disabled.  All three are
                # acceptable — the SSH test below is the real gate.
                ci_result = subprocess.run(
                    [*lxc, "exec", vm_name, "--", "cloud-init", "status", "--wait"],
                    check=False,
                )
                if ci_result.returncode == 1:
                    print(
                        "Warning: cloud-init reported errors (exit 1); "
                        "continuing to verify SSH connectivity"
                    )
                elif ci_result.returncode not in (0, 2):
                    raise RuntimeError(
                        f"cloud-init status returned unexpected exit code "
                        f"{ci_result.returncode}"
                    )

                _verify_ssh_and_sftp(ip, kai, tmpDir, [*lxc, "exec", vm_name, "--"])

            finally:
                print(f"Deleting LXC container: {vm_name}")
                subprocess.run(
                    [*lxc, "delete", "--force", vm_name],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    check=False,
                )

    except Exception as e:
        if isinstance(e, RuntimeError):
            raise
        raise RuntimeError(
            "An unexpected error occurred testing ohpygossh with lxc"
        ) from e


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--only",
        choices=["multipass", "lxc"],
        help="Run only the named test suite (default: run all)",
    )
    args = parser.parse_args()
    only = args.only

    test_say_hello()

    if only is None or only == "multipass":
        test_with_multipass()

    if only is None or only == "lxc":
        test_with_lxc()

    if only is None:
        if which("vagrant"):
            test_with_keys_only()
        else:
            print("vagrant not installed, skipping test_with_keys_only")
