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
