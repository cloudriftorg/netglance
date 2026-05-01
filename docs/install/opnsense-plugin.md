# Install netglance as an OPNsense plugin (`os-netglance`)

This puts a `Services → Netglance` tab in the OPNsense GUI, ties the daemon
into OPNsense's service lifecycle (start at boot, restart on save, status
in the dashboard), and gets you future updates from the OPNsense Plugin
Manager just like any official plugin.

The web UI itself remains accessible directly on the configured port (default
`8473`) so you can use it from any device on the LAN, including phones.

## 1. Add the pkg repo

This is a one-time setup. SSH into OPNsense as `root` (enable SSH first under
**System → Settings → Administration**).

```sh
# Trust the netglance signing key
fetch -o /usr/local/etc/pkg/keys/netglance.pub \
  https://netglance.github.io/netglance/repo.pub

# Tell pkg about the repo
cat > /usr/local/etc/pkg/repos/netglance.conf <<'EOF'
netglance: {
  url: "https://netglance.github.io/netglance/${ABI}",
  signature_type: "pubkey",
  pubkey: "/usr/local/etc/pkg/keys/netglance.pub",
  enabled: yes
}
EOF

pkg update
```

## 2. Install the plugin

```sh
pkg install os-netglance
```

This pulls in the `netglance` daemon as a dependency. After install, refresh
your OPNsense web UI — a new menu item **Services → Netglance** appears.

## 3. Configure

In **Services → Netglance**:

1. Pick the **listen port** (default 8473) and **listen address** (default
   `0.0.0.0`, i.e. all interfaces — adjust if you want to firewall it).
2. Pick one or more **scan interfaces** (LAN, OPT1, etc.).
3. Add the **networks** you want inventoried — one row per CIDR. VLAN ID
   is optional and only affects the badge in the netglance UI.
4. Tick **Enable**, click **Save**.

OPNsense renders the env file and restarts the daemon. Click **Open Netglance
UI** to access the netglance web UI in a new tab — it has its own admin
password (separate from your OPNsense login), set on first launch.

## 4. Updates

Future versions show up in **System → Firmware → Plugins** alongside the
official OPNsense plugins. Click **Update** to bump.

The `pkg upgrade` is also available from the shell:

```sh
pkg update
pkg upgrade os-netglance netglance
```

State (`/var/db/netglance/netglance.db`) is preserved across updates.

## 5. Uninstall

From the GUI: **System → Firmware → Plugins → os-netglance → trash icon**.

From shell:
```sh
pkg delete os-netglance
# Optional: also remove the daemon and state
pkg delete netglance
rm -rf /var/db/netglance      # wipes inventory and admin password
```

The repo entry stays in `/usr/local/etc/pkg/repos/netglance.conf` until you
remove it manually.

## Troubleshooting

**Plugin tab doesn't appear after install.**  
Hard-refresh your browser; the OPNsense menu cache lives in the page.

**"Open Netglance UI" link goes nowhere.**  
Check that the firewall lets your management host reach the chosen port.
By default OPNsense blocks new ports on LAN if you've added explicit rules.

**Logs**: `tail -F /var/log/netglance.log /var/log/configd.log`

**Reset to factory state**: stop the service, delete `/var/db/netglance/*`,
restart. The setup wizard reappears on next visit to `:8473`.
