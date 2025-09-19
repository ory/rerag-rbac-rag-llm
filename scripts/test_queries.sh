#!/bin/bash

API_URL="http://localhost:8080"

echo "Testing RAG system with sample queries..."
echo "========================================="

echo -e "\nQuery 1: What was John Doe's refund amount in 2023?"
curl -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -d '{
        "question": "What was John Doe'\''s refund amount in 2023?",
        "top_k": 3
    }' | jq '.'

echo -e "\n\n========================================="
echo -e "Query 2: Which taxpayers filed as married?"
curl -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -d '{
        "question": "Which taxpayers filed as married filing jointly?",
        "top_k": 3
    }' | jq '.'

echo -e "\n\n========================================="
echo -e "Query 3: What was ABC Corporation's gross receipts?"
curl -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -d '{
        "question": "What was ABC Corporation'\''s gross receipts in 2023?",
        "top_k": 3
    }' | jq '.'

echo -e "\n\n========================================="
echo -e "Query 4: Who received child tax credit?"
curl -X POST "${API_URL}/query" \
    -H "Content-Type: application/json" \
    -d '{
        "question": "Which taxpayers received child tax credit and how much?",
        "top_k": 3
    }' | jq '.'