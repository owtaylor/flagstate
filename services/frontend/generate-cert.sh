#!/bin/bash
set -e

work=$(mktemp -d)
cleanup() {
    rm -rf $work
}
trap cleanup EXIT

cd $work

# Generate private keys
openssl genrsa -out flagstate_ca.key 2048
openssl genrsa -out flagstate.key 2048

# Generate CSRs
cat > ca.config <<EOF
[req]
prompt=no
distinguished_name=cadn
req_extensions=v3_req

[v3_req]
basicConstraints=critical,CA:TRUE,pathlen:0

[cadn]
CN=Flagstate CA
OU=Flagstate
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
OU=Flagstate
emailAddress=nomail@example.com
EOF

openssl req -new -config ca.config -key flagstate_ca.key -out flagstate_ca.csr
openssl req -new -config cert.config -key flagstate.key -out flagstate_cert.csr

# Generate Root Certificate
openssl x509 -req -in flagstate_ca.csr -days 365 -extfile ca.config -extensions v3_req -signkey flagstate_ca.key -out flagstate_ca.crt

# Generate Server Certificate
openssl x509 -req -in flagstate_cert.csr -days 365 -extfile cert.config -extensions v3_req -CA flagstate_ca.crt -CAkey flagstate_ca.key -CAcreateserial -out flagstate.crt

# Copy the files to the correct locations
cp flagstate.crt flagstate_ca.crt /etc/pki/tls/certs
cp flagstate.key /etc/pki/tls/private/
