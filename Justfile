# Development tasks for ohpygossh.
# Install `just`: https://github.com/casey/just
#
# Run `just` (or `just --list`) to see all available recipes.

set shell := ["bash", "-euo", "pipefail", "-c"]

# gopy and goimports versions are pinned in go.mod (as Go 1.24+ tool
# dependencies) and managed there by Dependabot; every consumer derives them
# from go.mod at recipe run-time (not parse-time -- `setup` is what installs
# Go, so a top-level `:=` variable would need Go before it exists) so there
# is exactly one place to bump each.
#
# Go itself is pinned to the go@1.26 Homebrew formula (matching go.mod's `go
# 1.26.0` and the Go version CI pins via actions/setup-go) rather than the
# rolling `go` formula. go@1.26 is keg-only, so every recipe that shells out
# to `go` (directly, or indirectly via pre-commit's golangci-lint hook, which
# always builds golangci-lint from source against whatever `go` is on PATH)
# must prepend its bin dir to PATH itself. Letting the rolling `go` formula
# drift ahead (e.g. to 1.27) previously reintroduced a golangci-lint/
# honnef.co/go/tools panic when linting this repo.

# Show available recipes
default:
    @just --list

# Install all tooling needed for local development (macOS and Linux, via Homebrew)
setup:
    #!/usr/bin/env bash
    set -euo pipefail
    if ! command -v brew >/dev/null 2>&1; then
        echo "error: Homebrew is required. Install it from https://brew.sh, then re-run 'just setup'." >&2
        exit 1
    fi

    echo "==> Installing Go 1.26, Python 3.11, poetry, pre-commit"
    # golangci-lint itself is not installed here: pre-commit builds it from the
    # pinned rev in .pre-commit-config.yaml, so a separate system copy would
    # only drift out of sync with that pin.
    brew install go@1.26 python@3.11 poetry pre-commit
    export PATH="$(brew --prefix go@1.26)/bin:$PATH"

    echo "==> Installing gopy and goimports"
    GOPY_VERSION="$(go list -m all | awk '$1 == "github.com/go-python/gopy" {print $2}')"
    go install "github.com/go-python/gopy@${GOPY_VERSION}"
    GOIMPORTS_VERSION="$(go list -m all | awk '$1 == "golang.org/x/tools" {print $2}')"
    go install "golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}"

    GOBIN="$(go env GOPATH)/bin"
    case ":$PATH:" in
        *":$GOBIN:"*) ;;
        *) echo "warning: $GOBIN is not on your PATH; add it to use gopy/goimports directly" >&2 ;;
    esac

    GO125_BIN="$(brew --prefix go@1.26)/bin"
    case ":$PATH:" in
        *":$GO125_BIN:"*) ;;
        *) echo "note: $GO125_BIN is not on your PATH; every 'just' recipe adds it itself, but add it yourself to run 'go' or 'golangci-lint' directly" >&2 ;;
    esac

    echo "==> Configuring the Python virtual environment"
    poetry env use "$(brew --prefix python@3.11)/bin/python3.11"
    poetry install --no-root

    echo "==> Installing git hooks"
    pre-commit install

    echo "Setup complete."

# Run Go and Python linters (same checks as CI)
lint:
    #!/usr/bin/env bash
    set -euo pipefail
    export PATH="$(brew --prefix go@1.26)/bin:$PATH"
    go get .
    pre-commit run --all-files

# Run the Go test suite
test:
    #!/usr/bin/env bash
    set -euo pipefail
    export PATH="$(brew --prefix go@1.26)/bin:$PATH"
    go test ./...

# Build the Python wheel (produces dist/ohpygossh-*.whl)
#
# CGO_ENABLED=1 is required for gopy's c-shared build and isn't always the
# default (e.g. on some Linux/arm64 toolchains); see build-golang-macos.yaml.
build:
    #!/usr/bin/env bash
    set -euo pipefail
    export PATH="$(brew --prefix go@1.26)/bin:$PATH"

    rm -rf .venv/ dist/

    POETRY=$(type -p poetry) || { echo "Unable to find 'poetry', it must be installed to continue" >&2; exit 1; }
    "$POETRY" install --no-root

    if ! [ -f .venv/bin/activate ]; then
        echo "Error: failed to locate virtual environment" >&2
        exit 1
    fi
    source .venv/bin/activate

    # NOTE: Using 'poetry shell' does not work in place of the above.
    CGO_ENABLED=1 PATH="$PATH:$HOME/go/bin" gopy build -output=ohpygossh -vm=python3 .

    python3 -m pip install --upgrade setuptools wheel
    python3 setup.py bdist_wheel

    # Prove that the wheel can be installed.
    pip install dist/ohpygossh-*.whl

# Re-validate an already-built wheel without rebuilding it
validate:
    #!/usr/bin/env bash
    set -euo pipefail

    # Reuse an existing venv rather than always recreating it -- that's the
    # slow part 'build' does, and this recipe exists to skip it.
    if ! [ -f .venv/bin/activate ]; then
        POETRY=$(type -p poetry) || { echo "Unable to find 'poetry', it must be installed to continue" >&2; exit 1; }
        "$POETRY" install --no-root
    fi
    if ! [ -f .venv/bin/activate ]; then
        echo "Error: failed to locate virtual environment" >&2
        exit 1
    fi
    source .venv/bin/activate

    # Prove that the wheel can be installed, then validate functionality.
    pip install dist/ohpygossh-*.whl
    python validate_ohpygossh.py

# Remove build artifacts and the virtual environment
clean:
    rm -rf .venv dist build __pycache__ ohpygossh.egg-info ohpygossh myssh

# Run everything CI runs: setup, lint, then test
ci: setup lint test

# Validate that gopy's pin is only sourced from go.mod (no stray hardcoded copies)
gopy-version-check:
    #!/usr/bin/env bash
    set -euo pipefail
    gopy_version="$(go list -m all | awk '$1 == "github.com/go-python/gopy" {print $2}')"
    echo "gopy is pinned to: ${gopy_version}"

    if [[ "${gopy_version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-(0\.)?[0-9]{14}-[0-9a-f]{12}(\+incompatible)?$ ]]; then
        echo "note: gopy is still pinned to an untagged commit (${gopy_version})."
        echo "      Check https://github.com/go-python/gopy/releases -- if a tagged"
        echo "      release now exists, run 'just gopy-version-bump' to switch."
    fi

    if grep -RInE 'gopy@[0-9a-f]{40}' --include='*.yaml' --include='*.yml' --include='*.sh' .github build-scripts Justfile; then
        echo "error: found a hardcoded gopy commit pin outside go.mod; derive it from go.mod instead" >&2
        exit 1
    fi
    echo "no stray hardcoded gopy pins found"

# Bump gopy to the tip of its default branch (go.mod/go.sum are the source of truth)
gopy-version-bump:
    # @master, not @latest: gopy's tags lag its default branch, so @latest
    # would silently downgrade past commits this project relies on.
    go get -tool github.com/go-python/gopy@master
    go mod tidy
