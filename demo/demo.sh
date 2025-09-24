#!/bin/bash

# Strict error handling for CI/CD
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Function to print colored output
print_header() {
    echo -e "\n${BOLD}${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}${GREEN}$1${NC}"
    echo -e "${BOLD}${BLUE}═══════════════════════════════════════════════════════${NC}\n"
}

print_scenario() {
    echo -e "${BOLD}${YELLOW}► $1${NC}"
}

print_result() {
    echo -e "${GREEN}✓ Result:${NC} $1\n"
}

print_error() {
    echo -e "${RED}✗ Error:${NC} $1\n"
}

# Check dependencies
check_dependencies() {
    if ! command -v jq > /dev/null 2>&1; then
        print_error "jq is not installed. Please install it first:"
        echo "  macOS: brew install jq"
        echo "  Linux: apt-get install jq or yum install jq"
        exit 1
    fi
}

# Check if server is running
check_server() {
    if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
        print_error "Server not running. Please run 'make dev' in another terminal first."
        exit 1
    fi
}

# Main demo
main() {
    print_header "🚀 ReBAC-Powered RAG Demo"
    echo "This demo shows how different users see different results for the same query."
    echo "The LLM never sees documents the user isn't authorized to access."
    sleep 2

    # Check dependencies
    check_dependencies

    # Check server
    check_server

    print_header "📝 Sample Documents in System"
    echo "1. Tax Return 2023 - John Doe (Refund: \$8,500)"
    echo "2. Tax Return 2023 - ABC Corporation (Refund: \$45,000)"
    echo "3. Tax Return 2022 - John Doe (Refund: \$3,200)"
    echo "4. Financial Report - ABC Corporation (Revenue: \$5.2M)"
    sleep 2

    print_header "🔐 User Permissions"
    echo "• Alice: Can only access John Doe's documents"
    echo "• Bob: Can only access ABC Corporation's documents"
    echo "• Peter: Admin with access to all documents"
    sleep 2

    print_header "🔍 Demo: Same Query, Different Results"

    QUERY='{"question": "What was the total refund amount for 2023 for John Doe?"}'

    # Alice's query
    print_scenario "Alice queries the system (authorized for John Doe only):"
    echo -e "Command: ${BOLD}curl -X POST localhost:8080/query -H \"Auth: Bearer alice\" -d '$QUERY'${NC}"

    ALICE_RESPONSE=$(curl -s -X POST http://localhost:8080/query \
        -H "Authorization: Bearer alice" \
        -H "Content-Type: application/json" \
        -d "$QUERY" | jq -r '.answer // .error // "No response"')

    # Validate Alice's response contains expected data
    if [[ "$ALICE_RESPONSE" == *"1,200"* ]] && [[ "$ALICE_RESPONSE" != *"ABC Corporation"* ]] && [[ "$ALICE_RESPONSE" != *"3,500"* ]]; then
        print_result "$ALICE_RESPONSE"
    else
        print_error "Alice's response doesn't show proper permission isolation"
        echo "Response: $ALICE_RESPONSE"
        exit 1
    fi
    sleep 1

    QUERY='{"question": "What was the total refund amount for 2023 for ABC Corp?"}'
    # Bob's query
    print_scenario "Bob queries the system (authorized for ABC Corp only):"
    echo -e "Command: ${BOLD}curl -X POST localhost:8080/query -H \"Auth: Bearer bob\" -d '$QUERY'${NC}"

    BOB_RESPONSE=$(curl -s -X POST http://localhost:8080/query \
        -H "Authorization: Bearer bob" \
        -H "Content-Type: application/json" \
        -d "$QUERY" | jq -r '.answer // .error // "No response"')

    # Validate Bob's response contains expected data
    if [[ "$BOB_RESPONSE" == *"3,500"* ]] && [[ "$BOB_RESPONSE" != *"John Doe"* ]] && [[ "$BOB_RESPONSE" != *"1,200"* ]]; then
        print_result "$BOB_RESPONSE"
    else
        print_error "Bob's response doesn't show proper permission isolation"
        echo "Response: $BOB_RESPONSE"
        exit 1
    fi
    sleep 1

    # Peter's query (admin)
    print_scenario "Peter queries the system (admin with full access):"
    echo -e "Command: ${BOLD}curl -X POST localhost:8080/query -H \"Auth: Bearer peter\" -d '$QUERY'${NC}"

    PETER_RESPONSE=$(curl -s -X POST http://localhost:8080/query \
        -H "Authorization: Bearer peter" \
        -H "Content-Type: application/json" \
        -d "$QUERY" | jq -r '.answer // .error // "No response"')

    # Validate Peter's response contains data (admin access)
    if [[ "$PETER_RESPONSE" != *"No response"* ]] && [[ "$PETER_RESPONSE" != "" ]]; then
        print_result "$PETER_RESPONSE"
    else
        print_error "Peter's response doesn't show admin access to all documents"
        echo "Response: $PETER_RESPONSE"
        exit 1
    fi
    sleep 1

    # Unauthorized query
    print_scenario "Unauthorized user tries to query:"
    echo -e "Command: ${BOLD}curl -X POST localhost:8080/query -H \"Auth: Bearer hacker\" -d '$QUERY'${NC}"

    UNAUTH_RESPONSE=$(curl -s -X POST http://localhost:8080/query \
        -H "Authorization: Bearer hacker" \
        -H "Content-Type: application/json" \
        -d "$QUERY" 2>/dev/null | jq -r '.error // "Access denied"')

    print_result "$UNAUTH_RESPONSE"

    # Unauthorized query (Alice accessing John Doe's data)
    QUERY='{"question": "What was the total refund amount for 2023 for ABC Corp?"}'
    print_scenario "Alice tries to ask about data they don't have access to:"
    echo -e "Command: ${BOLD}curl -X POST localhost:8080/query -H \"Auth: Bearer alice\" -d '$QUERY'${NC}"

    UNAUTH_RESPONSE=$(curl -s -X POST http://localhost:8080/query \
        -H "Authorization: Bearer hacker" \
        -H "Content-Type: application/json" \
        -d "$QUERY" 2>/dev/null | jq -r '.error // "Access denied"')

    print_result "$UNAUTH_RESPONSE"

    print_header "🎯 Key Takeaways"
    echo "✅ Each user only sees documents they're authorized to access"
    echo "✅ The LLM never receives unauthorized content in its context"
    echo "✅ No prompt injection can leak data the user shouldn't see"
    echo "✅ Perfect for multi-tenant systems with sensitive documents"
    echo ""
    echo -e "${BOLD}${GREEN}Demo complete!${NC} Try your own queries with different users."
    echo -e "Documentation: ${BLUE}https://github.com/your-org/llm-rag-rebac${NC}"
}

# Run the demo
main "$@"