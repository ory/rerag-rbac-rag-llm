#!/bin/bash

API_URL="http://localhost:8080"
USER="${1:-peter}"

echo "Testing RAG system with sample queries as user: $USER"
echo "Usage: $0 [username]"
echo "Available users: alice (John Doe only), bob (ABC Corp only), peter (all)"
echo "========================================="

echo -e "\nChecking user permissions:"
curl -sS -X GET "${API_URL}/permissions" \
    -H "Authorization: Bearer $USER" | jq '.'

echo -e "\n========================================="
echo -e "Query 1: What was John Doe's refund amount in 2023?"
curl -sS -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $USER" \
    -d '{
        "question": "What was John Doe'\''s refund amount in 2023?",
        "top_k": 3
    }' | jq '.'

echo -e "\n\n========================================="
echo -e "Query 2: Which taxpayers filed as married?"
curl -sS -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $USER" \
    -d '{
        "question": "Which taxpayers filed as married filing jointly?",
        "top_k": 3
    }' | jq '.'

echo -e "\n\n========================================="
echo -e "Query 3: What was ABC Corporation's gross receipts?"
curl -sS -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $USER" \
    -d '{
        "question": "What was ABC Corporation'\''s gross receipts in 2023?",
        "top_k": 3
    }' | jq '.'

echo -e "\n\n========================================="
echo -e "Query 4: Who received child tax credit?"
curl -sS -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $USER" \
    -d '{
        "question": "Which taxpayers received child tax credit and how much?",
        "top_k": 3
    }' | jq '.'

echo -e "\n\n========================================="
echo "Note: Results will vary based on user permissions."
echo "- alice: Can only see John Doe's documents"
echo "- bob: Can only see ABC Corporation's documents"
echo "- peter: Can see all documents"