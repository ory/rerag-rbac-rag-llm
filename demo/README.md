# Demo Directory

This directory contains all demonstration materials for the LLM RAG ReBAC
system.

## Structure

```
demo/
├── demo.sh                   # Interactive demo script
├── load_documents.sh         # Script to load sample documents into the system
├── setup_keto_permissions.sh # Script to configure Keto permissions
└── documents/                # Demo data files
    ├── README.md            # Documentation for demo documents
    ├── sample_documents.json # Sample tax documents for RAG
    └── relation_tuples.json  # Permission configurations for Keto
```

## Demo Scripts

### `demo.sh`

Interactive demonstration script that showcases the RAG system with
permission-based access control. It demonstrates:

- Document uploading
- Permission-aware querying
- Different user access levels (alice, bob, peter)

### `load_documents.sh`

Loads the sample tax documents from `documents/sample_documents.json` into the
system. These documents serve as the knowledge base for RAG queries.

### `setup_keto_permissions.sh`

Configures Ory Keto with the relation tuples from
`documents/relation_tuples.json`. This establishes the permission structure:

- **alice**: Can access John Doe's documents only
- **bob**: Can access ABC Corporation's documents only
- **peter**: Admin access to all documents

## Running the Demo

### Quick Start

```bash
make demo     # Run the interactive demo
```

### Manual Setup

```bash
# 1. Setup permissions in Keto
./demo/setup_keto_permissions.sh

# 2. Load sample documents
./demo/load_documents.sh

# 3. Run the interactive demo
./demo/demo.sh
```

### Using Make Commands

```bash
make setup    # Setup permissions and load documents
make demo     # Run the interactive demonstration
```

## Demo Users and Permissions

The demo includes three users with different access levels:

| User  | Access Level | Can Access                      |
| ----- | ------------ | ------------------------------- |
| alice | Limited      | John Doe's tax documents        |
| bob   | Limited      | ABC Corporation's tax documents |
| peter | Admin        | All documents                   |

## Sample Queries

Here are some example queries to test the permission system:

```bash
# Alice can query John Doe's information
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer alice" \
  -H "Content-Type: application/json" \
  -d '{"question": "What was John Doe's refund amount?"}'

# Bob cannot access John Doe's information
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer bob" \
  -H "Content-Type: application/json" \
  -d '{"question": "What was John Doe's refund amount?"}'

# Peter can access all information
curl -X POST http://localhost:8080/query \
  -H "Authorization: Bearer peter" \
  -H "Content-Type: application/json" \
  -d '{"question": "List all taxpayers and their refund amounts"}'
```

## Customizing the Demo

To add your own demo scenarios:

1. **Add Documents**: Edit `documents/sample_documents.json` to include your
   documents
2. **Set Permissions**: Update `documents/relation_tuples.json` to define access
   rules
3. **Modify Scripts**: Adjust the demo scripts to showcase your specific use
   cases

## Important Notes

- The demo uses simple Bearer token authentication for demonstration purposes
- Sample documents contain fictional tax data for demonstration only
- The system requires Ollama and Keto to be running before starting the demo

For more information about the system architecture and implementation, see the
main [README](../README.md).
