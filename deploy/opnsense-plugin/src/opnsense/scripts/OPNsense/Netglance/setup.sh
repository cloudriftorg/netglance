#!/bin/sh

# Ensure the netglance state directory exists with correct ownership.
# The netglance pkg already creates /var/db/netglance, but we re-assert here
# in case the pkg was upgraded or the directory was removed.

mkdir -p /var/db/netglance
chown netglance:netglance /var/db/netglance
chmod 750 /var/db/netglance

mkdir -p /usr/local/etc/netglance
chmod 755 /usr/local/etc/netglance
