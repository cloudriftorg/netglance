#!/bin/sh

# Ensure the netglance directories exist before the daemon (re)starts —
# install.sh creates them, this re-asserts them in case they were removed.
# The daemon runs as root (arp-scan needs /dev/bpf*), so no chown here.

mkdir -p /var/db/netglance
chmod 750 /var/db/netglance

mkdir -p /usr/local/etc/netglance
chmod 755 /usr/local/etc/netglance
