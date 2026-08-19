
set positional-arguments
set shell := ["bash", "-cue"]
set fallback := true

root_dir := `git rev-parse --show-toplevel`
comp_dir := justfile_directory()
flake_dir := root_dir / "tools/nix"
output_dir := comp_dir / ".output"
build_dir := output_dir / "build"
container_mgr := "podman"

# Default target if you do not specify a target.
default:
    just --list

# Run an executable.
run *args:
    go run ./main.go "$@"

# Build the project.
build *args:
    #!/usr/bin/env bash
    export GOBIN="{{build_dir}}/bin"

    echo "Go generate ..."
    go generate ./...

    echo "Go build ..."
    go install -tags debug,development "$@" ./...

install:
    go install ./

# Enter the default Nix development shell and execute the command `"$@`.
develop *args:
    just nix-develop "default" "$@"

# Enter the Nix development shell `$1` and execute the command `${@:2}`.
nix-develop *args:
    #!/usr/bin/env bash
    set -eu
    shell="$1"; shift 1;
    args=("$@") && [ "${#args[@]}" != 0 ] || args="$SHELL"
    mkdir -p .devenv/state && pwd >.devenv/state/pwd
    nix develop \
        --accept-flake-config \
        --override-input devenv-root "path:.devenv/state/pwd" \
        "{{flake_dir}}#$shell" \
        --command "${args[@]}"

format:
    just develop treefmt

# Lint the project.
lint *args:
    golangci-lint run \
        --max-issues-per-linter 0 \
        --max-same-issues 0 \
        --timeout 10m0s \
        --config "{{root_dir}}/tools/configs/golangci-lint/golangci.yaml" \
        "$@"


# Test the project.
test *args:
    go test "$@" ./...
