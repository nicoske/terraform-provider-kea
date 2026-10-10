#!/bin/sh
set -e

echo "Creating dummy network interfaces..."

ip link add dev dummy0 type dummy
ip addr add 10.67.0.1/24 dev dummy0
ip link set dev dummy0 up

ip link add dev dummy1 type dummy
ip addr add 10.67.1.1/24 dev dummy1
ip link set dev dummy1 up

ip link add dev dummy2 type dummy
ip addr add 10.67.2.1/24 dev dummy2
ip link set dev dummy2 up

echo "Network interfaces created successfully:"
ip addr show dummy0
ip addr show dummy1
ip addr show dummy2

# config-write needs a writable directory holding the startup config file
cp /usr/local/etc/kea-dhcp4.conf /etc/kea/kea-dhcp4.conf

echo "Starting Kea DHCP4 server..."
exec /usr/sbin/kea-dhcp4 -X -c /etc/kea/kea-dhcp4.conf
