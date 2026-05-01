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

---

# Maintainer workflow — iterating on the plugin

This section is for the maintainer (or anyone forking) who wants to push
fixes and new versions through the same pkg repo end users will install
from. It uses only `pkg` on the OPNsense side — nothing extra to install.

## One-time setup

### 1. Generate the repo signing key (locally)

```sh
mkdir -p ~/.netglance-signing && cd ~/.netglance-signing
openssl genrsa -out repo.key 2048
openssl rsa -in repo.key -pubout > repo.pub
chmod 600 repo.key
```

The private `repo.key` stays on your workstation. If lost, you can
generate a new one but every existing user has to re-trust it.

### 2. Add the key as a GitHub secret

Repo on GitHub → **Settings → Secrets and variables → Actions →
New repository secret**:
- Name: `PKG_SIGN_KEY`
- Value: the full contents of `repo.key` (including the BEGIN/END lines)

### 3. Create the empty `gh-pages` branch

```sh
git checkout --orphan gh-pages
git rm -rf .
echo "# netglance pkg repo" > README.md
git add README.md && git commit -m "init: gh-pages for pkg repo"
git push origin gh-pages
git checkout main
```

### 4. Enable GitHub Pages

Repo → **Settings → Pages → Source: Deploy from a branch →
Branch: gh-pages, /(root) → Save**.

The published URL appears at the top of that page once the first deploy
runs (typically `https://<user>.github.io/netglance/`).

## Cutting a release-candidate

Each iteration is a new tag. `pkg` keys upgrades on version, so reusing
the same tag won't be picked up.

```sh
# develop and validate locally first
make local                            # full app in Docker
# (test the change in your browser)

git commit -am "fix: ..."
git tag v0.1.0-rc.1
git push && git push --tags
```

Watch the **Actions** tab — the `Build & publish FreeBSD pkg repo`
workflow takes ~7-10 minutes. When green, `https://<user>.github.io/netglance/repo.pub`
serves the public key.

## Installing or upgrading on your OPNsense

First time only — on the OPNsense (replace `<user>` with the GitHub user/org
hosting the repo):

```sh
ssh root@<opnsense>
fetch -o /usr/local/etc/pkg/keys/netglance.pub \
  https://<user>.github.io/netglance/repo.pub
cat > /usr/local/etc/pkg/repos/netglance.conf <<'EOF'
netglance: {
  url: "https://<user>.github.io/netglance/${ABI}",
  signature_type: "pubkey",
  pubkey: "/usr/local/etc/pkg/keys/netglance.pub",
  enabled: yes
}
EOF
pkg update && pkg install -y os-netglance
```

For every subsequent rc tag, on OPNsense:
- **GUI**: System → Firmware → Plugins → row `os-netglance` → Update
  (the row appears as "update available" after `pkg update` runs nightly)
- **Shell** (immediate): `pkg update && pkg upgrade -y os-netglance netglance`

## What you can test locally vs what needs OPNsense

| Change to... | Testable locally? | How |
|---|---|---|
| Backend Go code | ✅ | `make local` (Docker) |
| Frontend React code | ✅ | `make ui` (Vite HMR) |
| OPNsense plugin (`.volt` / `.xml` / `.php`) | ❌ | needs OPNsense |

For the third row, you have two options:

1. **Tag → CI → upgrade** (clean, slow): the loop above. Right for
   anything you're confident about, or that touches the OPNsense
   config schema (`General.xml`).
2. **VM-based fast loop** (sandbox-only): a throwaway OPNsense VM you
   `rsync` files into. ~3 seconds per iteration, but never on your
   real OPNsense — it leaves files outside the pkg database.
   See [dev/opnsense-vm/README.md](../../dev/opnsense-vm/README.md).

Use #2 while iterating heavily on the PHP/Volt scaffolding, then once
it's stable bump an rc tag and validate via #1 on the real box.

## Cutting the stable release

When the rc dance settles:

```sh
git tag v0.1.0       # no -rc suffix
git push --tags
```

Same workflow runs, publishes `0.1.0` to `Latest/`. Existing users
upgrade through Plugin Manager as usual.
