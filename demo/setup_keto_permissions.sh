#!/bin/bash

set -euo pipefail

echo "Setting up Keto permissions for tax document access..."

# Check if Keto server is running
if ! curl -sf http://127.0.0.1:4467/health/ready >/dev/null 2>&1; then
    echo "Keto server is not running. Please start it with:"
    echo "  ./.bin/keto serve --config keto/config.yml &"
    exit 1
fi

echo "Keto server is running. Setting up relation tuples..."

# Load relation tuples from JSON file
./.bin/keto relation-tuple create demo/documents/relation_tuples.json --config keto/config.yml --insecure-disable-transport-security

echo ""
echo "Permission setup complete!"
echo ""
echo "Testing permissions via Keto CLI..."

# Test Alice's permissions
echo -n "Alice can view john-doe:2023: "
./.bin/keto check alice viewer documents john-doe:2023 --config keto/config.yml --insecure-disable-transport-security 2>/dev/null

echo -n "Alice cannot view jane-smith:2023: "
./.bin/keto check alice viewer documents jane-smith:2023 --config keto/config.yml --insecure-disable-transport-security 2>/dev/null

echo -n "Bob can view abc-corporation:2023: "
./.bin/keto check bob viewer documents abc-corporation:2023 --config keto/config.yml --insecure-disable-transport-security 2>/dev/null

echo -n "Peter can view jane-smith:2023: "
./.bin/keto check peter viewer documents jane-smith:2023 --config keto/config.yml --insecure-disable-transport-security 2>/dev/null

echo ""
echo "Testing permissions via HTTP API..."

# Test via HTTP API (same as our Go service will use)
echo -n "Alice can view john-doe:2023 (HTTP): "
curl -s "http://127.0.0.1:4466/relation-tuples/check/openapi?namespace=documents&object=john-doe:2023&relation=viewer&subject_id=alice" | jq -r '.allowed // "error"'

echo -n "Alice cannot view jane-smith:2023 (HTTP): "
curl -s "http://127.0.0.1:4466/relation-tuples/check/openapi?namespace=documents&object=jane-smith:2023&relation=viewer&subject_id=alice" | jq -r '.allowed // "error"'

echo ""
echo "Setup and testing complete!"