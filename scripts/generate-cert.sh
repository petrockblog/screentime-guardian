#!/bin/bash
set -e

CERT_DIR="${1:-/etc/screentime-guardian}"
DAYS="${2:-3650}"  # 10 years

echo "Generating self-signed certificate for Screentime Guardian..."
echo "Certificate directory: $CERT_DIR"
echo "Valid for: $DAYS days"
echo ""

# Create directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Generate private key and certificate
openssl req -x509 -newkey rsa:4096 -nodes \
    -keyout "$CERT_DIR/server.key" \
    -out "$CERT_DIR/server.crt" \
    -days "$DAYS" \
    -subj "/CN=screentime-guardian.local/O=Screentime Guardian/C=US" \
    -addext "subjectAltName=DNS:screentime-guardian.local,DNS:localhost,IP:127.0.0.1"

# Set permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

echo ""
echo "✅ Certificate generated successfully!"
echo "   Certificate: $CERT_DIR/server.crt"
echo "   Private key: $CERT_DIR/server.key"
echo ""
echo "To enable HTTPS, add to /etc/screentime-guardian/config.yaml:"
echo ""
echo "enable_tls: true"
echo "tls_cert_file: $CERT_DIR/server.crt"
echo "tls_key_file: $CERT_DIR/server.key"
echo ""
echo "⚠️  Note: Browsers will show a security warning for self-signed certificates."
echo "   Click 'Advanced' and 'Proceed to site' to continue."
