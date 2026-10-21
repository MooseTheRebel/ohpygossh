"""A real, unprivileged OpenSSH sshd for the host key tests.

Mirrors startSSHD in sshd_harness_test.go and shares its config template.
"""

import os
import pwd
import shutil
import socket
import subprocess
import time
from dataclasses import dataclass
from pathlib import Path

import pytest

TEMPLATE = Path(__file__).resolve().parent.parent / "testdata" / "sshd_config.in"


@dataclass
class SSHD:
    addr: str  # "127.0.0.1:PORT"
    port: int
    user: str  # current OS user; a non-root sshd can only log in as itself
    client_key: str
    host_pub_keys: dict[str, str]  # ssh-keygen type -> "<algo> <base64>"
    dir: Path

    def known_hosts_line(self, key_type: str = "ed25519") -> str:
        return f"[127.0.0.1]:{self.port} {self.host_pub_keys[key_type]}"

    def write_known_hosts(self, path: Path, *key_types: str) -> Path:
        path.parent.mkdir(parents=True, exist_ok=True)
        types = key_types or tuple(self.host_pub_keys)
        path.write_text("".join(self.known_hosts_line(kt) + "\n" for kt in types))
        return path


def keygen(path: Path, key_type: str = "ed25519") -> str:
    """Generate a key pair at path; return its "<algo> <base64>" public key."""
    subprocess.run(
        ["ssh-keygen", "-q", "-N", "", "-t", key_type, "-f", str(path)], check=True
    )
    return " ".join(Path(f"{path}.pub").read_text().split()[:2])


def fingerprint(pub_path: Path) -> str:
    """Return the "SHA256:..." fingerprint of a public key file."""
    out = subprocess.run(
        ["ssh-keygen", "-lf", str(pub_path)], check=True, capture_output=True, text=True
    ).stdout
    return out.split()[1]


def _find_sshd() -> str:
    # OpenSSH refuses to re-exec from a relative path.
    found = shutil.which("sshd")
    if found is None and Path("/usr/sbin/sshd").exists():
        found = "/usr/sbin/sshd"
    if found:
        return str(Path(found).absolute())
    if os.environ.get("OHPYGOSSH_REQUIRE_SSHD") == "1":
        pytest.fail("sshd not installed, and OHPYGOSSH_REQUIRE_SSHD=1")
    pytest.skip("sshd not installed")


def _free_port() -> int:
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def _wait_for_banner(port: int, timeout: float = 10.0) -> bool:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=1) as conn:
                conn.settimeout(2)
                if conn.recv(8) == b"SSH-2.0-":
                    return True
        except OSError:
            pass
        time.sleep(0.1)
    return False


@pytest.fixture
def sshd(tmp_path):
    sshd_bin = _find_sshd()
    d = tmp_path / "sshd"
    d.mkdir()
    host_key = d / "host_ed25519"
    host_pub_keys = {"ed25519": keygen(host_key)}
    client_key = d / "client_ed25519"
    keygen(client_key)
    (d / "authorized_keys").write_text(Path(f"{client_key}.pub").read_text())

    cfg_path = d / "sshd_config"
    log_path = d / "sshd.log"
    for _ in range(2):
        port = _free_port()
        cfg_path.write_text(
            TEMPLATE.read_text()
            .replace("@PORT@", str(port))
            .replace("@DIR@", str(d))
            .replace("@HOST_KEY_LINES@", f"HostKey {host_key}")
        )
        check = subprocess.run(
            [sshd_bin, "-t", "-f", str(cfg_path)],
            check=False,
            capture_output=True,
            text=True,
        )
        if check.returncode != 0:
            pytest.fail(f"sshd -t: {check.stderr}")
        args = [sshd_bin, "-D", "-e", "-f", str(cfg_path)]
        # OpenSSH >= 9.8 penalises sources whose connections never
        # authenticate; older sshd rejects the option. See startSSHD.
        probe = [sshd_bin, "-t", "-f", str(cfg_path), "-o", "PerSourcePenalties=no"]
        if subprocess.run(probe, check=False, capture_output=True).returncode == 0:
            args += ["-o", "PerSourcePenalties=no"]
        log = log_path.open("w")
        proc = subprocess.Popen(args, stderr=log)
        if _wait_for_banner(port):
            break
        proc.kill()
        proc.wait()
        log.close()
    else:
        pytest.fail(f"sshd did not start; last log:\n{log_path.read_text()}")

    yield SSHD(
        addr=f"127.0.0.1:{port}",
        port=port,
        user=pwd.getpwuid(os.getuid()).pw_name,
        client_key=str(client_key),
        host_pub_keys=host_pub_keys,
        dir=d,
    )

    proc.kill()
    proc.wait()
    log.close()
    # pytest only shows teardown output for failing tests.
    print(f"sshd log:\n{log_path.read_text()}")
