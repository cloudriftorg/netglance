# Multi-VLAN scanning

Netglance uses `arp-scan` for host discovery, run from inside the container
(which itself runs in `network_mode: host`). ARP requests are L2 broadcasts:
they don't traverse routers, so the scanner only sees hosts in the broadcast
domain of the interface it's launched on.

To scan multiple VLANs, the Docker host must have a network interface
(typically a VLAN sub-interface) **with an IP** in each target VLAN. Once
an interface exists, netglance auto-detects it from the configured CIDR —
no per-network interface name required in the UI.

This guide walks through a typical Proxmox VM + Debian setup with three
VLANs.

## Example topology

| VLAN | CIDR              | Role             |
|------|-------------------|------------------|
| 1    | 192.168.1.0/24    | LAN (untagged)   |
| 20   | 192.168.20.0/24   | IoT              |
| 30   | 192.168.30.0/24   | Guest            |

Docker host: a VM on Proxmox with bridge `vmbr0` configured as VLAN-aware
(`VLAN IDs: 2-4094`), starting with one untagged NIC in VLAN 1.

## 1. Add virtual NICs in Proxmox

For each VLAN, on the VM:

```
Hardware → Add → Network Device
  Bridge:    vmbr0
  VLAN Tag:  <id, e.g. 20>
  Model:     VirtIO (paravirtualized)
```

After a reboot (or hot-plug), the new interfaces appear in the guest:

```bash
ip -br link show
# ens18  UP    bc:24:11:7a:89:b0
# ens19  DOWN  bc:24:11:33:71:ff
# ens20  DOWN  bc:24:11:7b:a7:d2
```

## 2. Assign static IPs

Each new interface needs an IP **inside the matching VLAN**, with **no
default gateway** (the default gateway must remain on the primary VLAN 1
interface). Pick an IP outside any DHCP pool.

### Debian / `ifupdown` (drop-in, no edit to existing files)

```bash
sudo tee /etc/network/interfaces.d/vlan-extras > /dev/null <<'EOF'
auto ens19
iface ens19 inet static
    address 192.168.20.2/24

auto ens20
iface ens20 inet static
    address 192.168.30.2/24
EOF
```

Verify the main `interfaces` file already sources drop-ins:

```bash
grep -E "^source" /etc/network/interfaces
# expected: source /etc/network/interfaces.d/*
```

Apply without reboot:

```bash
sudo ifup ens19
sudo ifup ens20
```

### Ubuntu / netplan

```yaml
# /etc/netplan/99-vlan-extras.yaml
network:
  version: 2
  ethernets:
    ens19:
      addresses: [192.168.20.2/24]
      dhcp4: false
    ens20:
      addresses: [192.168.30.2/24]
      dhcp4: false
```

```bash
sudo netplan apply
```

## 3. Verify L2 presence

```bash
ip -4 -br addr show
# ens18    UP    192.168.1.21/24
# ens19    UP    192.168.20.2/24
# ens20    UP    192.168.30.2/24

ping -c1 192.168.20.1   # gateway VLAN 20
ping -c1 192.168.30.1   # gateway VLAN 30
```

## 4. Configure networks in netglance

In the web UI → **Settings** → **Networks**:

| Name  | CIDR              | VLAN ID |
|-------|-------------------|---------|
| LAN   | 192.168.1.0/24    | 1       |
| IoT   | 192.168.20.0/24   | 20      |
| Guest | 192.168.30.0/24   | 30      |

Save and wait for the next scan cycle (or trigger one manually). Hosts on
the new VLANs appear with real MACs and proper vendor lookup.

## Undo

- Remove IP persistence: `sudo rm /etc/network/interfaces.d/vlan-extras`, then `sudo ifdown ens19 ens20` or reboot.
- Remove NICs: Proxmox UI → VM → Hardware → select NIC → Remove.
- Remove networks from netglance: Settings → Networks → delete the rows.

## Notes

- The added sub-interfaces serve only L2: they don't NAT, don't route, and
  exist only to populate the kernel's ARP cache via `arp-scan`.
- Keep the default gateway on the primary VLAN 1 interface only. Adding a
  second gateway on a sub-interface breaks the guest's routing.
- For very populated VLANs, raise `Scan interval (seconds)` in Settings to
  avoid CPU/network pressure.
