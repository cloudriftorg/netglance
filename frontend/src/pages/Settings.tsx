import { FormEvent, useEffect, useState } from 'react';
import { api, Settings as SettingsT, NetworkConfig, NetInterface, ManagedInfo } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';
import IfacePicker from '../components/IfacePicker';
import { Info, Trash2 } from 'lucide-react';

export default function SettingsPage() {
  const toast = useToast();
  const [s, setS] = useState<SettingsT | null>(null);
  const [saving, setSaving] = useState(false);
  const [ifaces, setIfaces] = useState<NetInterface[]>([]);
  const [resetOpen, setResetOpen] = useState(false);
  const [managed, setManaged] = useState<ManagedInfo>({ managed: false, fields: [] });

  useEffect(() => {
    api.getSettings()
      .then(setS)
      .catch((e) => {
        // In DEV bypass mode there's no backend — render an empty form
        // instead of getting stuck on "Loading…" forever so the layout
        // can still be previewed.
        if (import.meta.env.DEV && localStorage.getItem('netglance.dev.skipAuth') === '1') {
          setS({
            networks: [],
            scanEnabled: true,
            scanEverySeconds: 120,
            scanIfaces: [],
            offlineAfter: 1,
            notify: { newHost: false, offline: false, backOnline: false },
          });
          return;
        }
        toast.error(errMessage(e, 'Load failed'));
      });
    api.listInterfaces().then(setIfaces).catch(() => {
      /* non-fatal: combo just shows the saved value */
    });
    api.managed().then(setManaged).catch(() => {
      /* non-fatal: assume not managed */
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!s) return <p className="text-sm text-slate-500">Loading…</p>;

  const isManaged = (k: string) => managed.managed && managed.fields.includes(k);

  function update<K extends keyof SettingsT>(key: K, value: SettingsT[K]) {
    setS((cur) => (cur ? { ...cur, [key]: value } : cur));
  }
  function updateNet(idx: number, patch: Partial<NetworkConfig>) {
    if (!s) return;
    const next = s.networks.slice();
    next[idx] = { ...next[idx], ...patch };
    update('networks', next);
  }

  async function save() {
    if (!s) return;
    // Drop empty rows the user may have added but never filled in (clicked
    // + Add by mistake or cleared all fields). A row counts as empty when
    // CIDR, VLAN and Name are all blank.
    const trimmedNetworks = s.networks.filter(
      (n) => n.cidr.trim() !== '' || n.vlanId != null || (n.name ?? '').trim() !== ''
    );
    // For non-empty rows, CIDR and VLAN are required.
    for (const n of trimmedNetworks) {
      if (n.cidr.trim() === '') {
        toast.error('Each network row needs an IP/CIDR');
        return;
      }
      if (n.vlanId == null) {
        toast.error('Each network row needs a VLAN id');
        return;
      }
    }
    const cleaned = { ...s, networks: trimmedNetworks };
    setSaving(true);
    try {
      await api.putSettings(cleaned);
      // Reflect the cleanup back into local state so any dropped empty rows
      // disappear from the form even before the next reload.
      setS(cleaned);
      toast.success('Settings saved');
    } catch (e) {
      toast.error(errMessage(e, 'Save failed'));
    } finally {
      setSaving(false);
    }
  }

  async function testEmail() {
    try {
      await api.testSMTP();
      toast.success('Test email sent');
    } catch (e) {
      toast.error(errMessage(e, 'SMTP test failed'));
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-6">
      <div className="flex shrink-0 items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <h2 className="text-lg font-semibold">Settings</h2>
          {managed.managed && <ManagedBadge />}
        </div>
        <button onClick={save} disabled={saving} className="btn-primary">
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-6 overflow-y-auto pb-4 lg:grid-cols-2">
        <Section
          title="Scan"
          desc="Periodic scan runs in the background; the manual button on the Hosts page works regardless."
        >
          <IfacePicker
            value={s.scanIfaces ?? []}
            ifaces={ifaces}
            onChange={(v) => update('scanIfaces', v)}
            disabled={isManaged('scanIfaces')}
          />
          <div className="grid grid-cols-1 items-start gap-3 sm:grid-cols-3">
            <label className="block space-y-1.5 text-sm">
              {/* Invisible label keeps this column the same height as the
                  NumberFields beside it so the checkbox row lines up with
                  their input boxes. */}
              <span className="invisible block font-medium">.</span>
              <span className="flex h-9 items-center gap-2">
                <input
                  type="checkbox"
                  checked={s.scanEnabled}
                  disabled={isManaged('scanEnabled')}
                  onChange={(e) => update('scanEnabled', e.target.checked)}
                />
                Automatic scan
              </span>
            </label>
            <NumberField
              label="Interval (s)"
              min={10}
              value={s.scanEverySeconds}
              disabled={!s.scanEnabled || isManaged('scanEverySeconds')}
              onChange={(v) => update('scanEverySeconds', v)}
            />
            <NumberField
              label="Offline after"
              min={1}
              value={s.offlineAfter}
              onChange={(v) => update('offlineAfter', v)}
            />
          </div>
          <p className="text-xs text-slate-500">
            Minimum interval 10s. Lower values flood the network with ARP broadcasts and cause online/offline flapping. "Offline after" is the number of consecutive missed scans before a host is marked offline.
          </p>
        </Section>

        <Section
          title="Networks"
          desc="One row per CIDR you want scanned. VLAN ID is optional and only affects the badge."
        >
          {s.networks.length === 0 && (
            <p className="text-sm text-slate-500">No networks configured. Add one below.</p>
          )}
          {s.networks.map((n, i) => (
            <div key={i} className="grid grid-cols-12 gap-2">
              {/* JSX order matches visual + tab order: IP → VLAN → Name → ×. */}
              <input
                className="input col-span-11 sm:col-span-3"
                placeholder="192.168.1.0/24"
                value={n.cidr}
                disabled={isManaged('networks')}
                onChange={(e) => updateNet(i, { cidr: e.target.value })}
              />
              <input
                className="input col-span-4 col-start-1 row-start-2 sm:col-span-2 sm:col-start-auto sm:row-start-auto"
                inputMode="numeric"
                placeholder="VLAN"
                value={n.vlanId ?? ''}
                disabled={isManaged('networks')}
                onChange={(e) =>
                  updateNet(i, {
                    vlanId: e.target.value ? Number(e.target.value.replace(/\D/g, '')) : undefined,
                  })
                }
              />
              <input
                className="input col-span-8 row-start-2 sm:col-span-6 sm:row-start-auto"
                placeholder="Name (e.g. trusted)"
                value={n.name}
                disabled={isManaged('networks')}
                onChange={(e) => updateNet(i, { name: e.target.value })}
              />
              <button
                className="col-span-1 col-start-12 row-start-1 inline-flex h-9 w-9 items-center justify-center justify-self-center rounded-full text-slate-400 transition hover:bg-red-100 hover:text-red-600 disabled:opacity-40 dark:hover:bg-red-900 dark:hover:text-red-300 sm:row-start-auto"
                disabled={isManaged('networks')}
                onClick={() => update('networks', s.networks.filter((_, j) => j !== i))}
                aria-label="Remove network"
                title="Remove network"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
          <button
            className="btn-secondary text-sm disabled:opacity-50"
            disabled={isManaged('networks')}
            onClick={() => update('networks', [...s.networks, { name: '', cidr: '' }])}
          >
            + Add network
          </button>
        </Section>

        <Section
          title="SMTP"
          desc="Plain (no auth/no TLS) works for an internal LAN relay. STARTTLS or implicit TLS for external providers."
          action={<TestEmailButton onTest={testEmail} />}
        >
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <Field label="Host">
              <input
                className="input"
                value={s.smtp?.host ?? ''}
                onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), host: e.target.value })}
              />
            </Field>
            <NumberField
              label="Port"
              value={s.smtp?.port ?? 25}
              onChange={(v) => update('smtp', { ...(s.smtp ?? blankSMTP()), port: v })}
            />
            <Field label="From">
              <input
                className="input"
                type="email"
                placeholder="netglance@example.com"
                value={s.smtp?.from ?? ''}
                onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), from: e.target.value })}
              />
            </Field>
            <Field label="Recipients (comma-separated)">
              <input
                className="input"
                value={s.smtp?.recipients?.join(', ') ?? ''}
                onChange={(e) =>
                  update('smtp', {
                    ...(s.smtp ?? blankSMTP()),
                    recipients: e.target.value.split(',').map((x) => x.trim()).filter(Boolean),
                  })
                }
              />
            </Field>
          </div>
          <div className="space-y-2">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={s.smtp?.useTLS ?? false}
                onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), useTLS: e.target.checked })}
              />
              Use TLS (STARTTLS, or implicit TLS on port 465)
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={s.smtp?.useAuth ?? false}
                onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), useAuth: e.target.checked })}
              />
              Use authentication
            </label>
          </div>
          {s.smtp?.useAuth && (
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <Field label="Username">
                <input
                  className="input"
                  value={s.smtp?.username ?? ''}
                  onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), username: e.target.value })}
                />
              </Field>
              <Field label="Password">
                <input
                  className="input"
                  type="password"
                  value={s.smtp?.password ?? ''}
                  onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), password: e.target.value })}
                />
              </Field>
            </div>
          )}
        </Section>

        <div className="flex h-full flex-col gap-6">
          <Section title="Notifications" className="flex-1">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={s.notify.newHost}
                onChange={(e) => update('notify', { ...s.notify, newHost: e.target.checked })}
              />
              New host detected
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={s.notify.offline}
                onChange={(e) => update('notify', { ...s.notify, offline: e.target.checked })}
              />
              Host went offline
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={s.notify.backOnline}
                onChange={(e) => update('notify', { ...s.notify, backOnline: e.target.checked })}
              />
              Host came back online
            </label>
          </Section>

          <section className="space-y-3 rounded-2xl border border-red-200 bg-red-50 p-4 dark:border-red-700 dark:bg-red-950">
            <div>
              <h3 className="text-sm font-semibold text-red-700 dark:text-red-300">Danger zone</h3>
              <p className="mt-0.5 text-xs text-red-700 dark:text-red-300">
                Reset wipes the database (admin user, hosts, history, settings) and restarts the setup wizard. This cannot be undone.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setResetOpen(true)}
              className="rounded-md border border-red-300 bg-white px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-100 dark:border-red-600 dark:bg-transparent dark:text-red-300 dark:hover:bg-red-950"
            >
              Reset application
            </button>
          </section>
        </div>
      </div>

      {resetOpen && (
        <ResetDialog
          onClose={() => setResetOpen(false)}
          // Full reload so App.tsx re-runs its auth/setup probe — without it
          // the in-memory `authed` state would still render the dashboard
          // even though the server now reports setupComplete=false.
          onConfirmed={() => window.location.replace('/setup')}
        />
      )}
    </div>
  );
}

function ResetDialog({ onClose, onConfirmed }: { onClose: () => void; onConfirmed: () => void }) {
  const toast = useToast();
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.resetApp(password);
      toast.success('Application reset');
      onConfirmed();
    } catch (err) {
      toast.error(errMessage(err, 'Reset failed'));
      setBusy(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 px-4"
      onMouseDown={onClose}
    >
      <form
        onSubmit={submit}
        onMouseDown={(e) => e.stopPropagation()}
        className="w-full max-w-sm space-y-4 rounded-2xl border border-slate-200 bg-white p-5 shadow-xl dark:border-slate-700 dark:bg-slate-900"
      >
        <div>
          <h2 className="text-base font-semibold text-red-700 dark:text-red-300">Reset application</h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
            All data will be permanently deleted and you'll be sent back to the setup wizard. Confirm with your admin password.
          </p>
        </div>
        <label className="block space-y-1.5 text-sm">
          <span className="font-medium text-slate-700 dark:text-slate-300">Password</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoFocus
            className="input"
            autoComplete="current-password"
          />
        </label>
        <div className="flex justify-end gap-2 pt-1">
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-800"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={busy}
            className="rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-60"
          >
            {busy ? 'Resetting…' : 'Reset everything'}
          </button>
        </div>
      </form>
    </div>
  );
}

// ManagedBadge sits beside the Settings page title and surfaces the
// "managed by OPNsense" notice as a tooltip-style popover instead of a
// page-wide banner. The badge stays compact when the user already
// knows their netglance is OPNsense-managed; hover (desktop) or tap
// (touch) reveals the full message.
// TestEmailButton fires the SMTP self-test and surfaces a hover/tap
// hint reminding the user that the test reads the *persisted* config
// — unsaved form edits won't be picked up until they hit Save.
function TestEmailButton({ onTest }: { onTest: () => void }) {
  const [open, setOpen] = useState(false);
  return (
    <span
      className="relative inline-flex"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type="button"
        onClick={onTest}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        className="btn-secondary text-sm"
      >
        Test email
      </button>
      {open && (
        <div className="absolute right-0 top-full z-20 mt-2 w-64 rounded-lg border border-slate-200 bg-white p-3 text-xs text-slate-600 shadow-lg dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200">
          Save your settings first — the test uses the saved SMTP config, not the unsaved form values.
        </div>
      )}
    </span>
  );
}

function ManagedBadge() {
  const [open, setOpen] = useState(false);
  return (
    <span className="relative inline-flex">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        className="inline-flex items-center gap-1 rounded-full border border-amber-300 bg-amber-50 px-2 py-0.5 text-[11px] font-medium text-amber-800 dark:border-amber-600 dark:bg-amber-950 dark:text-amber-200"
        aria-label="Managed by OPNsense — details"
      >
        <Info className="h-3 w-3" />
        <span className="hidden sm:inline">Managed by OPNsense</span>
        <span className="sm:hidden">OPNsense</span>
      </button>
      {open && (
        <div className="absolute left-0 top-full z-20 mt-2 w-72 rounded-lg border border-amber-300 bg-amber-50 p-3 text-xs text-amber-900 shadow-lg dark:border-amber-600 dark:bg-amber-950 dark:text-amber-100">
          Scan interfaces, networks, interval and listen port are read-only here — edit them from <em>Services → Netglance</em> in your OPNsense panel. Notifications, SMTP and per-host preferences remain editable.
        </div>
      )}
    </span>
  );
}

function Section({
  title,
  desc,
  className,
  action,
  children,
}: {
  title: string;
  desc?: string;
  className?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section
      className={`space-y-3 rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900 ${className ?? ''}`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold">{title}</h3>
          {desc && <p className="mt-0.5 text-xs text-slate-500">{desc}</p>}
        </div>
        {action && <div className="shrink-0">{action}</div>}
      </div>
      <div className="space-y-3">{children}</div>
    </section>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1.5 text-sm">
      <span className="font-medium text-slate-700 dark:text-slate-300">{label}</span>
      {children}
    </label>
  );
}

// NumberField uses a text input with numeric inputMode instead of `type="number"`
// to drop the platform spinner buttons (which got in the way of free typing on
// some browsers) while still surfacing a numeric keypad on mobile. The `min`
// constraint is applied only at blur, so users can transiently type intermediate
// values like "3" while reaching for "30".
function NumberField({
  label,
  value,
  min,
  disabled,
  onChange,
  help,
}: {
  label: string;
  value: number;
  min?: number;
  disabled?: boolean;
  onChange: (v: number) => void;
  help?: string;
}) {
  return (
    <Field label={label}>
      <input
        className="input"
        type="text"
        inputMode="numeric"
        pattern="[0-9]*"
        disabled={disabled}
        value={value}
        onChange={(e) => {
          const cleaned = e.target.value.replace(/\D/g, '');
          onChange(cleaned === '' ? 0 : Number(cleaned));
        }}
        onBlur={(e) => {
          const n = Number(e.target.value);
          if (min != null && (Number.isNaN(n) || n < min)) onChange(min);
        }}
      />
      {help && <p className="text-xs text-slate-500">{help}</p>}
    </Field>
  );
}

function blankSMTP() {
  return { host: '', port: 25, useTLS: false, useAuth: false, from: '', recipients: [] };
}
