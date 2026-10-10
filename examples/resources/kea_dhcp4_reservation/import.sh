# Reservation identified by {subnet_id}/{identifier_type}/{identifier}, where
# identifier_type is one of circuit-id, client-id, duid, flex-id or hw-address
terraform import kea_dhcp4_reservation.server01 1/hw-address/aa:bb:cc:dd:ee:01

# Reservation identified by its reserved address
terraform import kea_dhcp4_reservation.server02 1/ip-address/192.0.2.11
