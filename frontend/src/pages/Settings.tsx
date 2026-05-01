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
      <h2 className="text-lg font-semibold">Settings</h2>

      <Section title="Networks" desc="One row per CIDR you want scanned. VLAN ID is optional and only affects the badge.">
        {s.networks.length === 0 && <p className="text-sm text-slate-500">No networks configured. Add one below.</p>}
        {s.networks.map((n, i) => (
          <div key={i} className="grid grid-cols-12 gap-2">
            <input className="input col-span-4" placeholder="Name (e.g. trusted)" value={n.name} onChange={(e) => updateNet(i, { name: e.target.value })} />
            <input className="input col-span-5" placeholder="192.168.1.0/24" value={n.cidr} onChange={(e) => updateNet(i, { cidr: e.target.value })} />
            <input className="input col-span-2" placeholder="VLAN" type="number" value={n.vlanId ?? ''} onChange={(e) => updateNet(i, { vlanId: e.target.value ? Number(e.target.value) : undefined })} />
            <button className="col-span-1 text-sm text-red-600" onClick={() => update('networks', s.networks.filter((_, j) => j !== i))} aria-label="Remove">×</button>
          </div>
        ))}
        <button className="btn-secondary text-sm" onClick={() => update('networks', [...s.networks, { name: '', cidr: '' }])}>
          + Add network
        </button>
      </Section>

      <Section title="Scan" desc="Periodic scan runs in the background; the manual button on the Hosts page works regardless.">
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={s.scanEnabled} onChange={(e) => update('scanEnabled', e.target.checked)} />
          Enable automatic scanning
        </label>
        <Field label="Scan interval (seconds)">
          <input className="input" type="number" min={30} disabled={!s.scanEnabled} value={s.scanEverySeconds} onChange={(e) => update('scanEverySeconds', Number(e.target.value))} />
        </Field>
        <Field label="Mark offline after N missed scans">
          <input className="input" type="number" min={1} value={s.offlineAfter} onChange={(e) => update('offlineAfter', Number(e.target.value))} />
        </Field>
      </Section>

      <Section title="SMTP" desc="Plain (no auth/no TLS) works for an internal LAN relay. STARTTLS or implicit TLS for external providers.">
        <Field label="Host">
          <input className="input" value={s.smtp?.host ?? ''} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), host: e.target.value })} />
        </Field>
        <Field label="Port">
          <input className="input" type="number" value={s.smtp?.port ?? 25} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), port: Number(e.target.value) })} />
        </Field>
        <Field label="From">
          <input className="input" value={s.smtp?.from ?? ''} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), from: e.target.value })} />
        </Field>
        <Field label="Recipients (comma-separated)">
          <input className="input" value={s.smtp?.recipients?.join(', ') ?? ''} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), recipients: e.target.value.split(',').map((x) => x.trim()).filter(Boolean) })} />
        </Field>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={s.smtp?.useTLS ?? false} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), useTLS: e.target.checked })} />
          Use TLS (STARTTLS, or implicit TLS on port 465)
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={s.smtp?.useAuth ?? false} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), useAuth: e.target.checked })} />
          Use authentication
        </label>
        {s.smtp?.useAuth && (
          <>
            <Field label="Username">
              <input className="input" value={s.smtp?.username ?? ''} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), username: e.target.value })} />
            </Field>
            <Field label="Password">
              <input className="input" type="password" value={s.smtp?.password ?? ''} onChange={(e) => update('smtp', { ...(s.smtp ?? blankSMTP()), password: e.target.value })} />
            </Field>
          </>
        )}
        <button onClick={testEmail} className="btn-secondary text-sm">Send test email</button>
      </Section>

      <Section title="Notifications">
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={s.notify.newHost} onChange={(e) => update('notify', { ...s.notify, newHost: e.target.checked })} />
          New host detected
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={s.notify.offline} onChange={(e) => update('notify', { ...s.notify, offline: e.target.checked })} />
          Watched host went offline
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={s.notify.backOnline} onChange={(e) => update('notify', { ...s.notify, backOnline: e.target.checked })} />
          Watched host came back online
        </label>
      </Section>

      <div className="flex items-center gap-3">
        <button onClick={save} disabled={saving} className="btn-primary">
          {saving ? 'Saving…' : 'Save settings'}
        </button>
      </div>
    </div>
  );
}

function Section({ title, desc, children }: { title: string; desc?: string; children: React.ReactNode }) {
  return (
    <section className="space-y-3 rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
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

function blankSMTP() {
  return { host: '', port: 25, useTLS: false, useAuth: false, from: '', recipients: [] };
}
