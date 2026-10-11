#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(dirname -- "$script_dir")
compose_file="$script_dir/docker-compose.yml"

version=${1:-${KEA_VERSION:-}}
if [ -z "$version" ]; then
	echo "usage: $0 <kea-version>" >&2
	exit 2
fi

teardown() {
	KEA_VERSION="$version" docker compose --file "$compose_file" down --volumes --remove-orphans
}

trap teardown EXIT

echo "==> Kea $version: starting stack"
teardown
KEA_VERSION="$version" docker compose --file "$compose_file" up --detach --wait

echo "==> Kea $version: running acceptance tests"
make -C "$repo_root" testacc
