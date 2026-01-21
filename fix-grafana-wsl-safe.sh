
#!/usr/bin/env bash
set -e

echo "================================================="
echo "✅ FIXING GRAFANA + INFLUXDB (WSL ARCH LINUX)"
echo "================================================="

# ---------- helpers ----------
fail() { echo "❌ $1"; exit 1; }
ok() { echo "✅ $1"; }
warn() { echo "⚠️  $1"; }

command -v docker >/dev/null || fail "Docker not installed"

ok "Docker found"

# ---------- detect containers ----------
echo
echo "🔍 Detecting running containers..."
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"

# Find InfluxDB container by image name
INFLUX_CONTAINER=$(docker ps \
  --filter "ancestor=influxdb" \
  --format "{{.Names}}" | head -n1)

if [ -z "$INFLUX_CONTAINER" ]; then
  warn "InfluxDB container not running"
  warn "Trying docker-compose names..."

  INFLUX_CONTAINER=$(docker ps \
    --format "{{.Names}}" | grep -i influx | head -n1)
fi

[ -z "$INFLUX_CONTAINER" ] && fail "No running InfluxDB container found"

ok "InfluxDB container detected: $INFLUX_CONTAINER"

# ---------- check port 8086 ----------
echo
echo "🔍 Checking InfluxDB listener (8086)..."

docker exec "$INFLUX_CONTAINER" sh -c \
  "ss -lnt 2>/dev/null | grep 8086 || netstat -lnt 2>/dev/null | grep 8086" \
  >/dev/null && ok "InfluxDB listening on 8086" \
  || fail "InfluxDB is NOT listening on port 8086"

# ---------- test connectivity from Grafana ----------
echo
echo "🔍 Detecting Grafana container..."

GRAFANA_CONTAINER=$(docker ps \
  --filter "ancestor=grafana/grafana" \
  --format "{{.Names}}" | head -n1)

[ -z "$GRAFANA_CONTAINER" ] && fail "Grafana container not running"

ok "Grafana container detected: $GRAFANA_CONTAINER"

echo
echo "🌐 Testing Grafana → InfluxDB connectivity..."

docker exec "$GRAFANA_CONTAINER" sh -c \
  "nc -z influxdb 8086 || nc -z $INFLUX_CONTAINER 8086" \
  && ok "Grafana can reach InfluxDB" \
  || fail "Grafana cannot reach InfluxDB (wrong Docker network)"

# ---------- suggest correct datasource URL ----------
echo
echo "✅ USE THIS IN GRAFANA DATASOURCE:"
echo
echo "URL: http://$INFLUX_CONTAINER:8086"
echo "Database: nbgo"
echo "Access: proxy"
echo
echo "❌ DO NOT USE localhost or 127.0.0.1"

echo
ok "All checks passed"
