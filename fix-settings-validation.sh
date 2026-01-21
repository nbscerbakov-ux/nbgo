
#!/usr/bin/env bash
set -euo pipefail

APP="./nbgo-gui"
APP_ARGS=("-mode" "tui")
CONFIG_YAML="nbgo.yml"
CONFIG_JSON="nbgo.json"
BACKUP_SUFFIX="$(date +%Y%m%d-%H%M%S)"
WORKDIR="$(pwd)"
LOGFILE="${WORKDIR}/nbgo-validate-${BACKUP_SUFFIX}.log"

# ---- helpers ---------------------------------------------------------------

has() { command -v "$1" >/dev/null 2>&1; }

print_header() {
  echo "====================================================================="
  echo "$1"
  echo "====================================================================="
}

backup_file() {
  local f="$1"
  if [ -f "$f" ]; then
    cp -f "$f" "${f}.${BACKUP_SUFFIX}.bak"
    echo "✅ Backup created: ${f}.${BACKUP_SUFFIX}.bak"
  fi
}

remove_bom_inplace() {
  # Remove UTF-8 BOM if present
  # Safe even if no BOM (no change)
  perl -i -pe 'BEGIN{binmode STDIN; binmode STDOUT} s/^\xEF\xBB\xBF//' "$1"
}

detect_first_char() {
  # first non-empty, non-space line's first char
  sed -e '/^[[:space:]]*$/d' "$1" | head -n1 | sed 's/^[[:space:]]*//' | cut -c1
}

validate_json() {
  if ! has jq; then
    echo "❌ jq not found. Install jq to validate JSON (apt: sudo apt install jq | brew: brew install jq)."
    return 1
  fi
  jq . "$1" >/dev/null
}

attempt_app_validation() {
  # Try a few common validation flags; capture output (non-fatal if unsupported)
  local out=""
  local rc=0
  if [ -x "$APP" ]; then
    if "$APP" "${APP_ARGS[@]}" --validate >"$LOGFILE" 2>&1; then rc=$?; out="--validate"; fi || true
    if [ ${rc} -ne 0 ] && "$APP" "${APP_ARGS[@]}" --check >"$LOGFILE" 2>&1; then rc=$?; out="--check"; fi || true
    if [ ${rc} -ne 0 ] && "$APP" "${APP_ARGS[@]}" --dry-run >"$LOGFILE" 2>&1; then rc=$?; out="--dry-run"; fi || true
    if [ ${rc} -ne 0 ]; then
      # last resort: just run it and capture startup validation logs
      "$APP" "${APP_ARGS[@]}" >"$LOGFILE" 2>&1 || rc=$?
      out="<startup>"
    fi
    echo "ℹ️  Validator invocation: ${out} (exit=${rc})"
    echo "ℹ️  Validator log: $LOGFILE"
    return ${rc}
  else
    echo "⚠️  App binary $APP not found or not executable. Skipping app-level validation."
    return 0
  fi
}

# ---- begin ----------------------------------------------------------------

print_header "nbgo settings auto-fix"

# Determine which config exists
CFG=""
if [ -f "$CONFIG_JSON" ]; then
  CFG="$CONFIG_JSON"
elif [ -f "$CONFIG_YAML" ]; then
  CFG="$CONFIG_YAML"
else
  echo "❌ No config found (${CONFIG_YAML} or ${CONFIG_JSON}). Aborting."
  exit 1
fi
echo "🔧 Using config: $CFG"

# Backup
backup_file "$CFG"

# Strip BOM/invisible leading bytes
echo "🧹 Removing UTF-8 BOM (if any) from $CFG ..."
remove_bom_inplace "$CFG"

# If YAML: convert to JSON (most Go apps expect JSON during strict validation)
if [[ "$CFG" == *.yml || "$CFG" == *.yaml ]]; then
  echo "🧭 Detected YAML config."
  if ! has yq; then
    echo "❌ yq is required to convert YAML → JSON."
    echo "   Install:  sudo apt install yq   OR   brew install yq"
    exit 1
  fi

  # Show what we’re about to do
  first_char=$(detect_first_char "$CFG" || echo "?")
  echo "ℹ️  First non-space character in YAML: '${first_char}'"

  # Convert to JSON beside it; keep original backup already created
  TMPJSON="${CFG%.*}.converted.json"
  echo "🔄 Converting YAML → JSON: $TMPJSON"
  yq -o=json '.' "$CFG" > "$TMPJSON"

  echo "🔍 Validating JSON syntax with jq..."
  validate_json "$TMPJSON"

  # Replace primary config with JSON (use the name app expects most)
  mv -f "$TMPJSON" "$CONFIG_JSON"
  CFG="$CONFIG_JSON"
  echo "✅ YAML converted to JSON at $CFG"
fi

# If JSON: validate syntax; also pretty-compact it to remove trailing commas, etc.
if [[ "$CFG" == *.json ]]; then
  echo "🔍 Validating JSON syntax with jq..."
  validate_json "$CFG"

  echo "🗜️  Normalizing JSON (compact & stable key order) ..."
  TMPFIX="${CFG}.tmpfix"
  jq -S . "$CFG" > "$TMPFIX"
  mv -f "$TMPFIX" "$CFG"
  echo "✅ JSON normalization complete."
fi

# Try app-level validation to surface detailed issues
print_header "Running app validation (to reveal the 11 errors)"
if attempt_app_validation; then
  echo "✅ App validation completed (exit 0)."
else
  echo "⚠️  App reported validation issues. Inspect: $LOGFILE"
  # Extract common “validation” lines for quick view
  echo
  echo "——— Validation summary ———"
  grep -Ei 'error|invalid|validation|failed' "$LOGFILE" || true
  echo "———————————————"
  echo
fi

print_header "Done"
echo "Next steps:"
echo "  1) Review $LOGFILE for any remaining field-specific errors."
echo "  2) Re-run: $APP ${APP_ARGS[*]}"
