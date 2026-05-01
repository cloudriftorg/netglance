import { useEffect, useState } from 'react';
import { api, Settings as SettingsT, NetworkConfig } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';

export default function SettingsPage() {
  const toast = useToast();
  const [s, setS] = useState<SettingsT | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.getSettings().then(setS).catch((e) => toast.error(errMessage(e, 'Load failed')));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!s) return <p className="text-sm text-slate-500">Loading…</p>;

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
    setSaving(true);
    try {
      await api.putSettings(s);
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
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Settings</h2>
        <button onClick={save} disabled={saving} className="btn-primary">
          {saving ? 'Saving…' : 'Save settings'}
        </button>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Section
          title="Networks"
          desc="One row per CIDR you want scanned. VLAN ID is optional and only affects the badge."
          className="lg:col-span-2"
        >
          {s.networks.length === 0 && (
            <p className="text-sm text-slate-500">No networks configured. Add one below.</p>
          )}
          {s.networks.map((n, i) => (
            <div key={i} className="grid grid-cols-12 gap-2">
              <input
                className="input col-span-11 sm:col-span-4"
                placeholder="Name (e.g. trusted)"
                value={n.name}
                onChange={(e) => updateNet(i, { name: e.target.value })}
              />
              <button
                className="col-span-1 text-sm text-red-600 sm:order-last"
                onClick={() => update('networks', s.networks.filter((_, j) => j !== i))}
                aria-label="Remove"
              >
                ×
              </button>
              <input
                className="input col-span-8 sm:col-span-5"
                placeholder="192.168.1.0/24"
                value={n.cidr}
                onChange={(e) => updateNet(i, { cidr: e.target.value })}
              />
              <input
                className="input col-span-4 sm:col-span-2"
                inputMode="numeric"
                placeholder="VLAN"
                value={n.vlanId ?? ''}
                onChange={(e) =>
                  updateNet(i, {
                    vlanId: e.target.value ? Number(e.target.value.replace(/\D/g, '')) : undefined,
                  })
                }
              />
            </div>
          ))}
          <button
            className="btn-secondary text-sm"
            onClick={() => update('networks', [...s.networks, { name: '', cidr: '' }])}
          >
            + Add network
          </button>
        </Section>

        <Section
          title="Scan"
          desc="Periodic scan runs in the background; the manual button on the Hosts page works regardless."
        >
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={s.scanEnabled}
              onChange={(e) => update('scanEnabled', e.target.checked)}
            />
            Enable automatic scanning
          </label>
          <NumberField
            label="Scan interval (seconds)"
            min={10}
            value={s.scanEverySeconds}
            disabled={!s.scanEnabled}
            onChange={(v) => update('scanEverySeconds', v)}
            help="Minimum 10s. Lower values flood the network with ARP broadcasts and cause online/offline flapping."
          />
          <NumberField
            label="Mark offline after N missed scans"
            min={1}
            value={s.offlineAfter}
            onChange={(v) => update('offlineAfter', v)}
          />
        </Section>

        <Section title="Notifications">
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
            Watched host went offline
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={s.notify.backOnline}
              onChange={(e) => update('notify', { ...s.notify, backOnline: e.target.checked })}
            />
            Watched host came back online
          </label>
        </Section>

        <Section
          title="SMTP"
          desc="Plain (no auth/no TLS) works for an internal LAN relay. STARTTLS or implicit TLS for external providers."
          className="lg:col-span-2"
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
          <button onClick={testEmail} className="btn-secondary text-sm">
            Send test email
          </button>
        </Section>
      </div>
    </div>
  );
}

function Section({
  title,
  desc,
  className,
  children,
}: {
  title: string;
  desc?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <section
      className={`space-y-3 rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900 ${className ?? ''}`}
    >
      <div>
        <h3 className="text-sm font-semibold">{title}</h3>
        {desc && <p className="mt-0.5 text-xs text-slate-500">{desc}</p>}
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
