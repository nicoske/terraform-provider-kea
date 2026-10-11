#!/bin/sh

set -eu

dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/tls
mkdir -p "$dir"
cd "$dir"

server_ext=$(mktemp)
client_ext=$(mktemp)
trap 'rm -f "$server_ext" "$client_ext" server.csr client.csr ca.srl' EXIT

cat >"$server_ext" <<'EOF'
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:localhost, DNS:kea-dhcp4, IP:127.0.0.1
EOF

cat >"$client_ext" <<'EOF'
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF

openssl req \
	-x509 \
	-newkey rsa:4096 \
	-nodes \
	-days 36500 \
	-keyout ca-key.pem \
	-out ca.pem \
	-subj "/CN=kea-test-ca" \
	-addext "basicConstraints=critical,CA:TRUE" \
	-addext "keyUsage=critical,keyCertSign,cRLSign"

openssl req \
	-newkey rsa:4096 \
	-nodes \
	-keyout server-key.pem \
	-out server.csr \
	-subj "/CN=kea-dhcp4"

openssl x509 \
	-req \
	-in server.csr \
	-CA ca.pem \
	-CAkey ca-key.pem \
	-CAcreateserial \
	-out server.pem \
	-days 36500 \
	-extfile "$server_ext"

openssl req \
	-newkey rsa:4096 \
	-nodes \
	-keyout client-key.pem \
	-out client.csr \
	-subj "/CN=kea-test-client"

openssl x509 \
	-req \
	-in client.csr \
	-CA ca.pem \
	-CAkey ca-key.pem \
	-CAcreateserial \
	-out client.pem \
	-days 36500 \
	-extfile "$client_ext"

chmod 644 ./*-key.pem

echo "Wrote test PKI to $dir"
