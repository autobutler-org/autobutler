#!/bin/bash
# Renders headscale ARM parameters by:
#   1. Substituting $HEADSCALE_DOMAIN and $ADMIN_EMAIL into setup-headscale.bash
#   2. Base64-encoding the result
#   3. Writing headscale.rendered.parameters.json for upload to Azure
#
# Called by: make render/headscale HEADSCALE_DOMAIN=ts.autobutler.org
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SETUP_SCRIPT="${SCRIPT_DIR}/setup-headscale.bash"
OUTPUT="${SCRIPT_DIR}/headscale.rendered.parameters.json"
TEMPLATE="${SCRIPT_DIR}/headscale.json"

: "${HEADSCALE_DOMAIN:?HEADSCALE_DOMAIN is required. Example: make render/headscale HEADSCALE_DOMAIN=ts.autobutler.org}"

echo "[render] Substituting variables into setup-headscale.bash..."
RENDERED_SCRIPT=$(
  sed \
    -e "s|\${HEADSCALE_DOMAIN}|${HEADSCALE_DOMAIN}|g" \
    -e "s|\${ADMIN_EMAIL:-admin@autobutler.org}|${ADMIN_EMAIL}|g" \
    "${SETUP_SCRIPT}"
)

echo "[render] Base64-encoding script..."
if [[ "$OSTYPE" == "darwin"* ]]; then
  SCRIPT_B64=$(printf '%s' "${RENDERED_SCRIPT}" | base64)
else
  SCRIPT_B64=$(printf '%s' "${RENDERED_SCRIPT}" | base64 -w 0)
fi

echo "[render] Writing ${OUTPUT}..."
python3 - <<PYTHON
import json

params = {
    "\$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "vmName":          {"value": "autobutler-headscale"},
        "adminUsername":   {"value": "autobutler"},
        "adminPublicKey":  {"value": "REPLACE_WITH_YOUR_SSH_PUBLIC_KEY"},
        "vmSize":          {"value": "Standard_B1s"},
        "headscaleDomain": {"value": "${HEADSCALE_DOMAIN}"},
        "adminEmail":      {"value": "${ADMIN_EMAIL}"},
        "allowedSSHCidr":  {"value": "*"},
        "scriptBase64":    {"value": """${SCRIPT_B64}"""},
    }
}

with open("${OUTPUT}", "w") as f:
    json.dump(params, f, indent=2)

print(f"[render] Written to ${OUTPUT}")
print()
print("Next: deploy with")
print("  az deployment group create \\\\")
print("    --resource-group <your-rg> \\\\")
print("    --template-file ${TEMPLATE} \\\\")
print("    --parameters ${OUTPUT} \\\\")
print('    --parameters adminPublicKey="\$(cat ~/.ssh/id_ed25519.pub)"')
PYTHON
