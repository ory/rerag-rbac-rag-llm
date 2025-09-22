#!/bin/bash

API_URL="http://localhost:8080"

echo "Loading sample tax documents into the system..."
echo "Note: Document upload does not require authentication"
echo ""

jq -c '.[]' documents/sample_documents.json | while read doc; do
    echo "Adding document: $(echo $doc | jq -r '.title')"
    curl -sS -X POST "${API_URL}/documents" \
        -H "Content-Type: application/json" \
        -d "$doc"
done

echo ""
echo "All documents loaded successfully!"
echo ""
echo "To query the system, you must authenticate with one of these users:"
echo "  - alice: Can query John Doe's documents only"
echo "  - bob: Can query ABC Corporation's documents only"
echo "  - peter: Can query all documents (admin)"
echo ""
echo "Example query where Alice is allowed to ask questions about Jon Doe\'s tax returns:"
echo "  curl -sS -X POST ${API_URL}/query \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -H \"Authorization: Bearer alice\" \\"
echo "    -d '{\"question\": \"What was John Doe'\"'\"'s refund amount?\"}' | jq"
echo ""
echo "Example query where Bob is prohibited from viewing Jon Doe\'s tax returns:"
echo "  curl -sS -X POST ${API_URL}/query \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -H \"Authorization: Bearer bob\" \\"
echo "    -d '{\"question\": \"What was John Doe'\"'\"'s refund amount?\"}' | jq"