"""Host key verification, exercised through the installed wheel against a real sshd."""

import os
import subprocess
import sys

import pytest
from conftest import fingerprint, keygen

from ohpygossh.gohpygossh import (
    DownloadWithPolicy,
    HostKeyPolicy,
    RunWithPolicy,
    UploadWithPolicy,
)


def test_imports_installed_wheel_not_checkout():
    import ohpygossh

    assert (
        "site-packages" in ohpygossh.__file__ or "dist-packages" in ohpygossh.__file__
    )


def test_known_host_accepted(sshd, tmp_path):
    kh = sshd.write_known_hosts(tmp_path / "known_hosts")
    out = RunWithPolicy(
        sshd.addr,
        sshd.user,
        sshd.client_key,
        "echo ohpygossh-ok",
        HostKeyPolicy(KnownHostsPath=str(kh)),
    )
    assert out.strip() == "ohpygossh-ok"


def test_unknown_host_rejected(sshd, tmp_path):
    kh = tmp_path / "known_hosts"
    kh.write_text("")
    with pytest.raises(
        RuntimeError, match=r"unknown host \[127\.0\.0\.1\]:\d+.*SHA256:"
    ):
        RunWithPolicy(
            sshd.addr,
            sshd.user,
            sshd.client_key,
            "true",
            HostKeyPolicy(KnownHostsPath=str(kh)),
        )


def test_changed_host_key_rejected(sshd, tmp_path):
    other = keygen(tmp_path / "other_ed25519")
    kh = tmp_path / "known_hosts"
    kh.write_text(f"[127.0.0.1]:{sshd.port} {other}\n")
    with pytest.raises(RuntimeError, match="host key mismatch.*man-in-the-middle"):
        RunWithPolicy(
            sshd.addr,
            sshd.user,
            sshd.client_key,
            "true",
            HostKeyPolicy(KnownHostsPath=str(kh)),
        )


def test_fingerprint_pin_accepted(sshd):
    fp = fingerprint(sshd.dir / "host_ed25519.pub")
    out = RunWithPolicy(
        sshd.addr,
        sshd.user,
        sshd.client_key,
        "echo ohpygossh-ok",
        HostKeyPolicy(Fingerprint=fp),
    )
    assert out.strip() == "ohpygossh-ok"


def test_insecure_opt_in(sshd):
    out = RunWithPolicy(
        sshd.addr,
        sshd.user,
        sshd.client_key,
        "echo ohpygossh-ok",
        HostKeyPolicy(InsecureIgnoreHostKey=True),
    )
    assert out.strip() == "ohpygossh-ok"


def test_upload_download_round_trip(sshd, tmp_path):
    policy = HostKeyPolicy(
        KnownHostsPath=str(sshd.write_known_hosts(tmp_path / "known_hosts"))
    )
    local = tmp_path / "up.bin"
    want = b"ohpygossh-sftp\x00\xff round trip\n"
    local.write_bytes(want)
    remote = sshd.dir / "up.bin"
    UploadWithPolicy(
        sshd.addr, sshd.user, sshd.client_key, str(local), str(remote), policy
    )
    back = tmp_path / "down.bin"
    DownloadWithPolicy(
        sshd.addr, sshd.user, sshd.client_key, str(remote), str(back), policy
    )
    assert back.read_bytes() == want


def test_default_policy_reads_home_known_hosts(sshd, tmp_path):
    # A Go c-shared library snapshots the environment at load time, so
    # changing os.environ["HOME"] after import is invisible to Go. Run in a
    # child interpreter whose HOME is set before ohpygossh is imported, from a
    # directory with no ohpygossh/ to shadow the installed wheel.
    home = tmp_path / "home"
    kh = sshd.write_known_hosts(home / ".ssh" / "known_hosts")
    code = (
        "import sys; from ohpygossh.gohpygossh import Run; "
        "print(Run(*sys.argv[1:5]).strip())"
    )

    def run_child():
        return subprocess.run(
            [
                sys.executable,
                "-c",
                code,
                sshd.addr,
                sshd.user,
                sshd.client_key,
                "echo ohpygossh-ok",
            ],
            env={**os.environ, "HOME": str(home)},
            cwd=tmp_path,
            check=False,
            capture_output=True,
            text=True,
        )

    r = run_child()
    assert r.returncode == 0, r.stderr
    assert r.stdout.strip() == "ohpygossh-ok"

    # Without ~/.ssh/known_hosts the connection is refused: "cannot read" when
    # there is no system-wide /etc/ssh/ssh_known_hosts either, else "unknown
    # host" because that file does not list this test server.
    kh.unlink()
    r = run_child()
    assert r.returncode != 0
    assert "cannot read known_hosts file" in r.stderr or "unknown host" in r.stderr
