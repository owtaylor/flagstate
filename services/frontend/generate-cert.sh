#!/bin/bash
set -e

work=$(mktemp -d)
cleanup() {
    rm -rf $work
}
trap cleanup EXIT

cd $work

# Generate private keys
openssl genrsa -out metastore_ca.key 2048
openssl genrsa -out metastore.key 2048

# Generate CSRs
cat > ca.config <<EOF
[req]
prompt=no
distinguished_name=cadn
req_extensions=v3_req

[v3_req]
basicConstraints=critical,CA:TRUE,pathlen:0

[cadn]
CN=Metastore CA
OU=Metastore
emailAddress=nomail@example.com
EOF

cat > cert.config <<EOF
[req]
prompt=no
distinguished_name=certdn
req_extensions=v3_req

[v3_req]
subjectAltName=DNS:registry.local.fishsoup.net
basicConstraints=critical,CA:FALSE

[certdn]
CN=registry.local.fishsoup.net
OU=Metastore
emailAddress=nomail@example.com
EOF

openssl req -new -config ca.config -key metastore_ca.key -out metastore_ca.csr
openssl req -new -config cert.config -key metastore.key -out metastore_cert.csr

# Generate Root Certificate
openssl x509 -req -in metastore_ca.csr -days 365 -extfile ca.config -extensions v3_req -signkey metastore_ca.key -out metastore_ca.crt

# Generate Server Certificate
openssl x509 -req -in metastore_cert.csr -days 365 -extfile cert.config -extensions v3_req -CA metastore_ca.crt -CAkey metastore_ca.key -CAcreateserial -out metastore.crt

# Copy the files to the correct locations
cp metastore.crt metastore_ca.crt /etc/pki/tls/certs
cp metastore.key /etc/pki/tls/private/
