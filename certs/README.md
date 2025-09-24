# TLS Certificates

This directory contains TLS certificates for HTTPS support in the LLM RAG ReBAC
OSS application.

## ⚠️ Important Security Notice

The certificates in this directory (`cert.pem` and `key.pem`) are **example
files for development and testing purposes only**. They should **never be used
in production**.

## Creating Your Own Certificates

### For Development/Testing

To generate new self-signed certificates for development:

```bash
# Generate a private key
openssl req -new -newkey rsa:2048 -days 365 -nodes -x509 \
    -keyout certs/key.pem \
    -out certs/cert.pem \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
```

This creates:

- `key.pem`: Private key file
- `cert.pem`: Self-signed certificate (365 days validity)

### For Production

For production deployments, you should:

1. **Use certificates from a trusted Certificate Authority (CA)** such as:
   - Let's Encrypt (free)
   - DigiCert
   - GlobalSign
   - Your organization's internal CA

2. **Replace the example files** with your production certificates:

   ```bash
   # Backup example files
   mv certs/cert.pem certs/cert.pem.example
   mv certs/key.pem certs/key.pem.example

   # Copy your production certificates
   cp /path/to/your/cert.pem certs/cert.pem
   cp /path/to/your/key.pem certs/key.pem

   # Set proper permissions
   chmod 600 certs/key.pem
   chmod 644 certs/cert.pem
   ```

## Configuration

The application uses these certificates when TLS is enabled via configuration:

```yaml
# config.yml
server:
  tls:
    enabled: true
    cert_file: 'certs/cert.pem'
    key_file: 'certs/key.pem'
    min_tls: '1.2'
```

Or via environment variables:

```bash
SERVER_TLS_ENABLED=true
SERVER_TLS_CERT_FILE=certs/cert.pem
SERVER_TLS_KEY_FILE=certs/key.pem
```

## Testing HTTPS

Once TLS is enabled, you can test the HTTPS endpoint:

```bash
# With self-signed certificates (ignore certificate warnings)
curl -k https://localhost:8080/health

# With valid certificates
curl https://localhost:8080/health
```

## Security Best Practices

1. **Never commit private keys to version control**
2. **Use proper file permissions** (`600` for private keys)
3. **Rotate certificates regularly** (before expiration)
4. **Use strong cipher suites** (configured in the application)
5. **Monitor certificate expiration dates**
6. **Use certificates from trusted CAs in production**

## File Permissions

Ensure proper permissions are set:

```bash
chmod 600 certs/key.pem    # Private key - owner read/write only
chmod 644 certs/cert.pem   # Certificate - world readable
```
