# Developing the OPNsense plugin

Iterating on the plugin (`deploy/opnsense-plugin/`) requires running OPNsense
somewhere — its PHP/Volt/configd stack only runs on FreeBSD. The fastest setup
is a local VM you can `rsync` files into.

## 1. Install OPNsense in a VM

Pick whatever hypervisor you already use:

| Host | Recommended |
|---|---|
| macOS (Apple Silicon) | [UTM](https://mac.getutm.app/) — free, native arm64 |
| macOS (Intel) | UTM, VirtualBox, or VMware Fusion |
| Linux | KVM/virt-manager, VirtualBox |
| Windows | VirtualBox, VMware Workstation |

Then:

1. Download the OPNsense ISO (latest stable amd64) from <https://opnsense.org/download/>.
2. Create a VM with **2 GB RAM**, **20 GB disk**, **2 NICs** (one bridged to your LAN
   so the plugin can actually scan something, one NAT for outbound updates).
3. Boot the ISO and follow the installer (default answers are fine).
4. After reboot, log in on the console and run `Configure interfaces` (option 1)
   to assign the two NICs as `WAN` (NAT) and `LAN` (bridged).
5. Note the LAN IP; you'll SSH into that.
6. Enable SSH from the web UI: **System → Settings → Administration → Secure Shell
   → Enable Secure Shell** + **Permit root user login**. Add your public key
   under **System → Access → Users → root → authorized keys**.

## 2. Quick sanity check

```sh
ssh root@<vm-ip> 'opnsense-version'
```

## 3. Iterating on the plugin

From the repo root, with `VM_HOST` pointing at your VM:

```sh
# one-time: cross-build and deploy the netglance binary
make dev-vm-deploy VM_HOST=root@10.0.0.42

# every time you edit a .volt / .xml / .php / .env in deploy/opnsense-plugin/
make dev-plugin-sync VM_HOST=root@10.0.0.42

# tail logs in another terminal
make dev-plugin-logs VM_HOST=root@10.0.0.42
```

`dev-plugin-sync` rsyncs the plugin tree into `/usr/local/opnsense/` on the VM
and restarts `configd`. The OPNsense web UI picks up XML/Volt changes on the
next page load — no full reload needed.

## 4. Resetting the plugin's config

If you want to wipe the Netglance plugin settings without reinstalling:

```sh
ssh root@<vm-ip> 'configctl netglance stop; rm -rf /var/db/netglance/*'
```

The `<netglance>` block in `/conf/config.xml` is removed by uninstalling
`os-netglance`; reinstall to start fresh.

## 5. Caveats

- **macOS file metadata**: `rsync` may copy `.DS_Store` and AppleDouble files.
  The `--exclude` flags in `sync.sh` handle this.
- **Permissions**: files end up owned by `root:wheel` on the VM, which is what
  OPNsense expects. No `chmod` dance needed.
- **PHP errors**: surface in `/var/log/configd.log` (the plugin's controller
  logs there) and `/var/log/audit.log` (PHP fatals). Watch both via
  `make dev-plugin-logs`.
- **Native daemon iteration**: when you only changed Go code, run
  `make dev-vm-deploy` (skips the plugin sync). When you only changed plugin
  files, run `make dev-plugin-sync` (skips the binary copy).
