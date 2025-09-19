#!/bin/bash

API_URL="http://localhost:8080"

echo "Loading sample tax documents into the system..."

jq -c '.[]' examples/sample_documents.json | while read doc; do
    echo "Adding document: $(echo $doc | jq -r '.title')"
    curl -X POST "${API_URL}/documents" \
        -H "Content-Type: application/json" \
        -d "$doc"
    echo -e "\n"
    sleep 1
done

echo "All documents loaded successfully!"
echo "You can now query the system at ${API_URL}/query"