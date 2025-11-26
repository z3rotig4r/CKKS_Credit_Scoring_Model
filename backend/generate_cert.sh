#!/bin/bash
# Generate self-signed SSL certificate for development

echo "Generating self-signed SSL certificate for CKKS Credit Backend..."

openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt \
    -days 365 -nodes \
    -subj "/C=KR/ST=Seoul/L=Seoul/O=CKKS Credit/OU=Development/CN=localhost"

if [ $? -eq 0 ]; then
    echo "✓ Certificate generated successfully:"
    echo "  - server.crt (certificate)"
    echo "  - server.key (private key)"
    echo ""
    echo "⚠ WARNING: This is a self-signed certificate for development only!"
    echo "   Browsers will show security warnings."
    echo ""
    echo "To use with Go backend:"
    echo "  http.ListenAndServeTLS(\":8080\", \"server.crt\", \"server.key\", router)"
else
    echo "✗ Certificate generation failed"
    exit 1
fi
