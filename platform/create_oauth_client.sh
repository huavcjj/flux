#!/bin/bash
set -e

PROJECT_ID=$1
SUPPORT_EMAIL=$2
OUTPUT_DIR=$3

if [ ! -f "$OUTPUT_DIR/.oauth_client_id" ]; then
  echo "Creating OAuth client..."

  # Create OAuth consent screen if not exists
  gcloud iap oauth-brands create \
    --application_title="Gmail-LINE Bot" \
    --support_email="$SUPPORT_EMAIL" \
    --project="$PROJECT_ID" 2>/dev/null || true

  # Get brand name
  BRAND=$(gcloud iap oauth-brands list --project="$PROJECT_ID" --format="value(name)" | head -1)

  # Create OAuth client
  OUTPUT=$(gcloud iap oauth-clients create "$BRAND" \
    --display_name="Gmail-LINE Bot OAuth Client" \
    --project="$PROJECT_ID" \
    --format=json)

  # Extract client ID and secret
  echo "$OUTPUT" | jq -r '.name' | awk -F'/' '{print $NF}' > "$OUTPUT_DIR/.oauth_client_id"
  echo "$OUTPUT" | jq -r '.secret' > "$OUTPUT_DIR/.oauth_client_secret"

  echo "OAuth client created successfully"
else
  echo "OAuth client already exists"
fi
