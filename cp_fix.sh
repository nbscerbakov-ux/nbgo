
#!/usr/bin/env bash
set -e

CONFIG="nbgo.yml"
BACKUP="nbgo.yml.bak"
TMPJSON="nbgo.tmp.json"

echo "🔍 Checking nbgo configuration file..."

# Check file exists
if [ ! -f "$CONFIG" ]; then
  echo "❌ ERROR: nbgo.yml not found!"
  exit 1
fi

# Backup original file
cp "$CONFIG" "$BACKUP"
echo "✅ Backup created: $BACKUP"

# Detect first non-empty character
FIRST_CHAR=$(sed -e '/^[[:space:]]*$/d' "$CONFIG" | head -n1 | sed 's/^[[:space:]]*//' | cut -c1)

# Remove UTF‑8 BOM if present
sed -i '1s/^\xEF\xBB\xBF//' "$CONFIG"

# If it looks like YAML, convert to JSON
if [[ "$FIRST_CHAR" =~ [a-zA-Z] ]]; then
  echo "⚠️  YAML detected → converting to JSON..."

  if ! command -v yq >/dev/null 2>&1; then
    echo "❌ ERROR: yq is required but not installed"
    echo "👉 Install with:  sudo apt install yq  OR  brew install yq"
    exit 1
  fi

  yq -o=json "$CONFIG" > "$TMPJSON"
  mv "$TMPJSON" "$CONFIG"

  echo "✅ Conversion successful: nbgo.yml is now valid JSON"
else
  echo "✅ File already appears to be valid JSON"
fi

# Final validation
if jq . "$CONFIG" >/dev/null 2>&1; then
  echo "✅ JSON validation OK"
else
  echo "❌ ERROR: JSON validation failed"
  echo "👉 Restore with: mv $BACKUP $CONFIG"
  exit 1
fi

echo "🎉 Fix complete. You can now run:"
echo "   ./nbgo-gui -mode tui"
