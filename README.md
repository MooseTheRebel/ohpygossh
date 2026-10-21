# ohpygossh

A project to create a new cross-platform SSH wheel for Python.

<img width="617" alt="image" src="docs/assets/banner.png">

## Goals

1. An easy to install, multi-platform SSH wheel
1. To explore [`gopy`](https://github.com/go-python/gopy)

## Code of Conduct

1. Respect and give kudos to all the work that's come
before.
    1. In particular, the Herculean efforts of [`gopy`], [`paramiko`] and [`asyncssh`].
    1. Please also be kind to all other open-source software projects you find.

It is paramount that we all play nicely.  To that end, please do your very best
not to create churn or spur conversations that may upset other developers,
and/or cause debate without offering solutions.

## Usage

Install the library:

```bash
pip install ohpygossh
```

Import and use the library to perform SSH operations:

```python
from ohpygossh.gohpygossh import Run, Upload, Download

# SSH connection parameters
#
# The remote hostname or IP address
ssh_server = "example.com"
# Your user account on the remote machine
ssh_user = "ubuntu"
# A PEM encoded private key file (like ~/.ssh/id_ed25519 )
private_key = "/path/to/private/key"

# Execute a command on a remote host
output = Run(ssh_server, ssh_user, private_key, "ls -la")

# Upload a file to the remote host
Upload(ssh_server, ssh_user, private_key, "/local/file.txt", "/remote/file.txt")

# Download a file from the remote host
Download(ssh_server, ssh_user, private_key, "/remote/file.txt", "/local/file.txt")
```

`ssh_server` may include a port, e.g. `"example.com:2222"` or `"[2001:db8::1]:2222"`.

## Security: host key verification

`Run`, `Upload` and `Download` verify the server's host key against
`~/.ssh/known_hosts` and the system-wide `/etc/ssh/ssh_known_hosts`, the same
files OpenSSH uses. If the host is missing, or its key has changed, they raise
an error instead of connecting. That error is your protection against
man-in-the-middle attacks.

> **Upgrading from 0.8.x or earlier:** older versions accepted any host key.
> Code that connects to hosts not in `~/.ssh/known_hosts` now raises; add the
> hosts as described below.

To trust a host, add its key to `known_hosts` after confirming its fingerprint
through a channel you already trust. For example, run `ssh-keygen -lf
/etc/ssh/ssh_host_ed25519_key.pub` on the server's console, or check your
cloud provider's console output. `ssh-keyscan` alone only records whatever
key answers first (trust on first use), so compare its output with that
fingerprint.

For other setups, pass a `HostKeyPolicy` to the `*WithPolicy` variants:

```python
from ohpygossh.gohpygossh import (
    DownloadWithPolicy,
    HostKeyPolicy,
    RunWithPolicy,
    UploadWithPolicy,
)

# Use a known_hosts file of your own
policy = HostKeyPolicy(KnownHostsPath="/path/to/known_hosts")

# Or pin the fingerprint of the server's Ed25519 host key, as printed by
# `ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` on the server
policy = HostKeyPolicy(Fingerprint="SHA256:...")

output = RunWithPolicy(ssh_server, ssh_user, private_key, "ls -la", policy)
UploadWithPolicy(
    ssh_server, ssh_user, private_key, "/local/a.txt", "/remote/a.txt", policy
)
DownloadWithPolicy(
    ssh_server, ssh_user, private_key, "/remote/a.txt", "/local/a.txt", policy
)
```

Set at most one of `KnownHostsPath`, `Fingerprint` and
`InsecureIgnoreHostKey`.

`HostKeyPolicy(InsecureIgnoreHostKey=True)` turns verification off. Anyone
able to intercept the connection can then read and alter everything you send
and receive. Use it only for throwaway test machines.

## Development

Development tasks are run with [`just`](https://github.com/casey/just), a
cross-platform command runner. All targets work the same way on macOS and
Linux; see the `Justfile` for the full list.

```bash
# One-time setup: installs Go, Python, gopy, poetry, pre-commit, etc. (via Homebrew)
just setup

# Run linters (golangci-lint, ruff, codespell, ...)
just lint

# Run the Go test suite
just test

# Build the Python wheel (produces dist/ohpygossh-*.whl)
just build

# Remove build artifacts and the virtual environment
just clean
```

`just setup` requires [Homebrew](https://brew.sh) to be installed first, since
it's the one package manager available on both macOS and Linux.

### Naming

This project has to think about naming in two different contexts:

* The Python library, installed by end-users: `ohpygossh`.
* The Golang package, also used in the `pyproject.toml`: `gohpygossh`.

Briefly:

```python
# Install the library:
#   pip install ohpygossh

# Import functions and data structures
from ohpygossh.gohpygossh import ...
```

## Reference

Based on:
* [`gopy`]
* https://github.com/b-long/cookiecutter-gopy

[`gopy`]: https://github.com/go-python/gopy
[`paramiko`]: https://pypi.org/project/paramiko/
[`asyncssh`]: https://pypi.org/project/asyncssh/
