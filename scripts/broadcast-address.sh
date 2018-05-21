#!/bin/bash

udp_payload="tank-odroid"
udp_port=6666

function broadcast_address() {
    echo "Sending UDP packet to all UDP broadcast addresses on port $udp_port"
    echo "$udp_payload" | socat - UDP-DATAGRAM:255.255.255.255:$udp_port,broadcast
}

while true; do
    broadcast_address
    sleep 5
done

