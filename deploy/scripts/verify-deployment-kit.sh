#!/usr/bin/env sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
compose="$root/deploy/compose/integin-infrastructure.compose.yaml"

test -f "$compose"
test -f "$root/deploy/systemd/integin-server.service.template"
test -f "$root/deploy/env/acceptance.env.schema"

if docker compose version >/dev/null 2>&1; then
  compose_command="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  compose_command="docker-compose"
else
  printf '%s\n' 'Docker Compose v2 or docker-compose is required for validation.' >&2
  exit 1
fi

# shellcheck disable=SC2086
$compose_command -f "$compose" config --no-interpolate >/dev/null
"$root/deploy/scripts/build-targets.sh"
go test ./... -count=1
go vet ./...
printf '%s\n' "INTEGIN deployment kit validation completed without deployment."
