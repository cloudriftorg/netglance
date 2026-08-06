#!/bin/sh
#
# netglance installer — self-contained, runs on the OPNsense box itself.
# The binary and every plugin file are appended to this script as a tar.gz;
# nothing is downloaded and nothing is compiled here.
#
#   sh installer.sh              install or update
#   sh installer.sh uninstall    remove every trace
#
# Afterwards, hard-refresh the OPNsense GUI. Restarting configd and php-fpm
# picks up the plugin's menu entry, MVC classes and ACL most of the time, not
# always — when it doesn't, rebooting the firewall does.
#
# Build it with ./build.sh on your workstation. Keep the file around (or
# rebuild it) if you want the uninstall path later.

set -eu

ACTION="${1:-install}"

case "$ACTION" in
    install|uninstall) ;;
    *) echo "usage: sh $0 [install|uninstall]" >&2; exit 1 ;;
esac

[ "$(id -u)" = "0" ] || { echo "must run as root" >&2; exit 1; }
[ -d /usr/local/opnsense ] || { echo "this does not look like an OPNsense system" >&2; exit 1; }
[ -f "$0" ] || { echo "run this as a file, not from a pipe: sh installer.sh" >&2; exit 1; }

# Unpack the appended payload. `tail -n +N` starts right after the marker
# line; the tar.gz below it is binary but line-counting gets us there fine.
TMP=$(mktemp -d /tmp/netglance-install.XXXXXX)
trap 'rm -rf "$TMP"' EXIT INT TERM
PAYLOAD_LINE=$(awk '/^__PAYLOAD_BELOW__$/ { print NR + 1; exit 0 }' "$0")
tail -n +"$PAYLOAD_LINE" "$0" | tar -xzf - -C "$TMP"

# The payload's src/ mirrors the target layout (src/etc → /usr/local/etc,
# src/opnsense → /usr/local/opnsense), so the install list and the uninstall
# list are both derived from it and can't drift apart.
FILES=$(cd "$TMP/src" && find . -type f | sed 's|^\.|/usr/local|')
DIRS=$(cd "$TMP/src" && find . -type d ! -name . | sed 's|^\.|/usr/local|' | sort -r)

# Pick up new configd actions, MVC classes, ACL and the menu entry. The
# php-fpm rc script isn't always at the same path across OPNsense releases, so
# try the known ways and don't let a failed reload abort the run — a stale GUI
# is fixed by reloading the page, a half-finished install is not.
reload_gui() {
    rm -f /tmp/opnsense_menu_cache.xml
    /usr/local/etc/rc.d/configd restart || true
    if [ -x /usr/local/etc/rc.d/php-fpm ]; then
        /usr/local/etc/rc.d/php-fpm restart || true
    elif service php-fpm restart >/dev/null 2>&1; then
        :
    else
        configctl webgui restart >/dev/null 2>&1 \
            || echo "!! could not restart php-fpm — reload the OPNsense GUI by hand" >&2
    fi
}

# Remove the plugin's own files — whole Netglance/ directories rather than just
# the files this payload lists, so a file dropped in a later version can't
# survive as an orphan the uninstall list would no longer mention. The .inc goes
# first: the GUI loads it on every request and it instantiates the model files
# removed right after. Never touches /var/db/netglance or config.xml.
remove_plugin_files() {
    for f in $FILES; do
        case "$f" in */plugins.inc.d/*) rm -f "$f" ;; esac
    done
    for d in $DIRS; do
        case "$d" in */Netglance) rm -rf "$d" ;; esac
    done
    for f in $FILES; do rm -f "$f"; done
}

do_install() {
    # arp-scan does the actual L2 sweep and isn't in OPNsense's repo, so it
    # travels in the payload and gets added from disk — no repo config, no
    # network.
    if ! command -v arp-scan >/dev/null 2>&1; then
        echo "==> installing arp-scan"
        if pkg add "$TMP/arp-scan.pkg"; then
            # Marker so `uninstall` knows arp-scan came with netglance.
            # It lives in the state dir, so it goes away with it.
            mkdir -p /var/db/netglance
            : > /var/db/netglance/.arp-scan-installed-by-netglance
        else
            echo "!! arp-scan could not be installed — scans will find nothing." >&2
        fi
    fi

    echo "==> installing files"
    remove_plugin_files
    # The .inc is copied last, once the model classes it instantiates are in
    # place: between the two there is a window where a GUI request would fatal
    # on a half-installed plugin, and a fatal in plugins.inc.d takes down the
    # whole OPNsense GUI, not just this page.
    mv "$TMP/src/etc/inc/plugins.inc.d" "$TMP/plugins.inc.d"
    cp -Rp "$TMP/src/." /usr/local/
    mkdir -p /usr/local/etc/inc/plugins.inc.d
    cp -Rp "$TMP/plugins.inc.d/." /usr/local/etc/inc/plugins.inc.d/
    # install(1) writes a temp file and renames, so this works even while the
    # old daemon is still running (a plain cp would hit ETXTBSY).
    install -m 755 "$TMP/netglance" /usr/local/sbin/netglance
    chmod 755 /usr/local/etc/rc.d/netglance

    mkdir -p /usr/local/etc/netglance /var/db/netglance
    chmod 750 /var/db/netglance

    echo "==> reloading OPNsense services"
    reload_gui

    # On an update the plugin is already enabled: re-render the env file and
    # bounce the daemon so it runs the binary we just shipped. On a first
    # install this is skipped — you enable it from the GUI.
    if [ "$(/usr/local/sbin/pluginctl -g OPNsense.netglance.general.enabled 2>/dev/null)" = "1" ]; then
        configctl -d template reload OPNsense/Netglance
        configctl -d netglance restart
    fi

    echo "✓ installed — hard-refresh the OPNsense GUI, then open Services → Netglance"
    echo "  if the menu entry is missing, reboot the firewall"
}

do_uninstall() {
    /usr/local/etc/rc.d/netglance onestop 2>/dev/null || true

    # Drop the plugin's config section first, while its model classes still
    # exist — the legacy config loader instantiates them via plugins.inc.d.
    if [ -f /usr/local/etc/inc/config.inc ]; then
        php -r 'require_once("config.inc");
                require_once("util.inc");
                global $config;
                if (isset($config["OPNsense"]["netglance"])) {
                    unset($config["OPNsense"]["netglance"]);
                    write_config("remove netglance plugin");
                    echo "config.xml: netglance settings removed\n";
                } else {
                    echo "config.xml: no netglance settings present\n";
                }' \
            || echo "!! could not edit config.xml — check for a stale <netglance> section" >&2
    fi

    # Read the marker before the state dir goes away.
    ARP_WAS_OURS=no
    [ -f /var/db/netglance/.arp-scan-installed-by-netglance ] && ARP_WAS_OURS=yes

    echo "==> removing files"
    remove_plugin_files
    # rmdir only, so shared OPNsense dirs (plugins.inc.d, rc.d, actions.d, …)
    # survive and just the now-empty ones we created go away.
    for d in $DIRS; do rmdir "$d" 2>/dev/null || true; done

    rm -f /usr/local/sbin/netglance   # the only file not part of the payload's src/
    rm -f /var/run/netglance.pid /var/log/netglance.log /var/log/netglance.log.*
    rm -rf /var/db/netglance /usr/local/etc/netglance

    # The installer never writes these, but a hand-run `service netglance
    # enable` would have. Clean them so no netglance_* knob survives.
    if [ -f /etc/rc.conf ] && grep -q '^netglance_' /etc/rc.conf; then
        sed -i '' '/^netglance_/d' /etc/rc.conf
    fi
    rm -f /etc/rc.conf.d/netglance

    if [ "$ARP_WAS_OURS" = yes ]; then
        echo "==> removing arp-scan (it came with netglance)"
        pkg delete -y arp-scan || true
    fi

    echo "==> reloading OPNsense services"
    reload_gui

    echo "✓ netglance removed — hard-refresh the OPNsense GUI"
    echo "  if the menu entry is still there, reboot the firewall"
}

if [ "$ACTION" = install ]; then
    do_install
else
    do_uninstall
fi

# Everything below the marker is the tar.gz payload, never executed.
exit 0
