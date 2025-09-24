# Demo Documents

This directory contains sample data files used for demonstrating the LLM RAG
ReBAC system.

## Files

### `sample_documents.json`

Contains sample tax documents for testing the RAG functionality. These documents
represent different taxpayers and tax returns:

- **John Doe**: Personal tax returns (2022 and 2023) with standard deduction
- **Jane Smith**: Married filing jointly with spouse Robert Smith (2023)
- **ABC Corporation**: Corporate tax return (Form 1120, 2023)
- **Michael Johnson**: Personal tax return (2023)

Each document includes:

- `id`: Unique identifier (UUID)
- `title`: Human-readable document title
- `content`: Full document text content
- `metadata`: Structured metadata including taxpayer, year, and form type

### `relation_tuples.json`

Contains Ory Keto relation tuples that define access permissions for the sample
documents. These tuples establish which users can view which documents:

- **alice**: Can access John Doe's documents
- **bob**: Can access ABC Corporation's documents
- **peter**: Admin access to all documents

Each relation tuple defines:

- `namespace`: The permission namespace ("documents")
- `object`: Document ID that the permission applies to
- `relation`: Type of permission ("viewer")
- `subject_id`: Username who has the permission

## Usage

These files are loaded during system setup to provide:

1. **Sample data** for testing RAG queries and retrieval
2. **Permission scenarios** for demonstrating ReBAC access control
3. **End-to-end testing data** for validating the complete system workflow

## Loading Data

The sample data is loaded automatically when running:

```bash
make setup    # Load documents and permissions
make demo     # Interactive demo with sample data
```

You can also load the data manually using the loading scripts in the `/demo`
directory.

## Customization

To add your own test documents:

1. Add entries to `sample_documents.json` following the existing format
2. Add corresponding permission tuples to `relation_tuples.json`
3. Update user permissions as needed for your test scenarios

**Note**: These are demo documents for demonstration purposes only. Do not use
real sensitive data in these files.
