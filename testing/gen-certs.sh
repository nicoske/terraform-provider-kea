#!/bin/sh
# Generates the PKI for the https control socket in kea-dhcp4.conf: a
# self-signed CA, a server certificate for localhost, kea-dhcp4 and 127.0.0.1,
# and a client certificate for mutual TLS. Requires openssl.
#
# The certs service in docker-compose.yml runs it on `up`. To run it by hand:
#
#   sh testing/gen-certs.sh testing/certs
#
# Certificates are kept until they expire within a day; delete the directory
# to force new ones.
set -e

dir=${1:-$(dirname "$0")/certs}
mkdir -p "$dir"
cd "$dir"

if openssl x509 -checkend 86400 -noout -in server.pem 2>/dev/null &&
	openssl x509 -checkend 86400 -noout -in client.pem 2>/dev/null; then
	echo "Certificates in $dir are still valid"
	exit 0
fi

days=825
newkey="-newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes"

openssl req -x509 $newkey -days "$days" -subj "/CN=Kea Test CA" \
	-keyout ca-key.pem -out ca.pem

cat >server.ext <<EXT
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = DNS:localhost, DNS:kea-dhcp4, IP:127.0.0.1
EXT
openssl req $newkey -subj "/CN=kea-dhcp4" -keyout server-key.pem -out server.csr
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
	-days "$days" -extfile server.ext -out server.pem

cat >client.ext <<EXT
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature
extendedKeyUsage = clientAuth
EXT
openssl req $newkey -subj "/CN=terraform" -keyout client-key.pem -out client.csr
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
	-days "$days" -extfile client.ext -out client.pem

rm -f ./*.csr ./*.ext ./*.srl
chmod 644 ./*.pem
# The certs service runs as root
if [ "$(id -u)" = 0 ]; then
	chown "$(stat -c %u:%g ..)" . ./*.pem
fi

echo "Certificates written to $dir"
