#!/usr/bin/env bash
set -euo pipefail
image=${1:?scanner image required}
suffix="$$"
network="scanner-canary-$suffix"
redis="scanner-redis-$suffix"
scanner="scanner-api-$suffix"
cleanup() {
  docker rm -f "$scanner" "$redis" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT
docker network create "$network" >/dev/null
docker run -d --name "$redis" --network "$network" --network-alias redis redis:7-alpine >/dev/null
docker run -d --name "$scanner" --network "$network" -p 127.0.0.1::8080 \
  -e SCANNER_REDIS_URL=redis://redis:6379 "$image" >/dev/null
port=$(docker port "$scanner" 8080/tcp | cut -d: -f2)
curl --fail --silent --show-error --retry 30 --retry-delay 1 --retry-all-errors "http://127.0.0.1:$port/probe/healthy"
curl --fail --silent --show-error "http://127.0.0.1:$port/probe/ready"
curl --fail --silent --show-error "http://127.0.0.1:$port/api/v1/metadata" | python3 -c '
import json,sys
d=json.load(sys.stdin)
assert d["scanner"]["version"].lstrip("v") == "0.76.1", d["scanner"]
assert d["properties"]["org.label-schema.version"] == "0.39.1-appmana.1"
assert len(d["capabilities"]) >= 2
print("Scanner normal entrypoint, health, readiness, and version metadata passed")
'
