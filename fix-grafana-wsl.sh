
#!/usr/bin/env bash
set -e

echo "================================================="
echo "✅ FIXING GRAFANA + DATASOURCES (WSL ARCH LINUX)"
echo "================================================="

# Required services
SERVICES=(
  "app:8080"
  "grafana:3000"
  "prometheus:9090"
  "clickhouse:8123"
  "influxdb:8086"
)

# 1. Docker check
if ! command -v docker >/dev/null; then
  echo "❌ Docker not installed"
  exit 1
fi

echo "✅ Docker found"

# 2. Verify containers running
echo
echo "🔍 Checking running containers..."
docker ps --format "table {{.Names}}\t{{.Status}}"

# 3. Check InfluxDB listener inside container
echo
echo "🔍 Checking InfluxDB port..."
docker exec influxdb ss -lnt | grep 8086 || {
  echo "❌ InfluxDB not listening on 8086"
  exit 1
}

# 4. Connectivity tests from Grafana container
echo
echo "🌐 Testing connectivity from Grafana container..."

for svc in "${SERVICES[@]}"; do
  NAME=${svc%%:*}
  PORT=${svc##*:}

  echo -n "➡️  $NAME:$PORT ... "
  if docker exec grafana sh -c "nc -z $NAME $PORT"; then
    echo "✅ OK"
  else
    echo "❌ FAIL"
  fi
done

# 5. Auto-create Grafana datasources
echo
echo "📊 Creating Grafana datasources..."

GF_API="http://admin:admin@localhost:3000/api/datasources"

create_ds() {
  curl -s -X POST "$GF_API" \
    -H "Content-Type: application/json" \
    -d "$1" >/dev/null
}

# InfluxDB
create_ds '{
  "name":"InfluxDB",
  "type":"influxdb",
  "access":"proxy",
  "url":"http://influxdb:8086",
  "database":"nbgo",
  "isDefault":true
}'

# Prometheus
create_ds '{
  "name":"Prometheus",
  "type":"prometheus",
  "url":"http://prometheus:9090",
  "access":"proxy"
}'

# ClickHouse
create_ds '{
  "name":"ClickHouse",
  "type":"grafana-clickhouse-datasource",
  "url":"http://clickhouse:8123",
  "access":"proxy"
}'

echo "✅ Datasources created"

# 6. Test InfluxDB query
echo
echo "🧪 Testing InfluxDB query via curl..."
curl -s http://localhost:8086/ping >/dev/null \
  && echo "✅ InfluxDB reachable" \
  || echo "❌ InfluxDB NOT reachable"

echo
echo "🎉 DONE"
echo "Open Grafana: http://localhost:3000"
