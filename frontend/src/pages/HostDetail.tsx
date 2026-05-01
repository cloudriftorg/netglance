import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { api, Host, HostEvent } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';
import Spinner from '../components/Spinner';

export default function HostDetail() {
  const toast = useToast();
  const { mac } = useParams<{ mac: string }>();
  const [data, setData] = useState<{ host: Host; events: HostEvent[] } | null>(null);
  const [customName, setCustomName] = useState('');
  const [customVendor, setCustomVendor] = useState('');
  const [notifyOffline, setNotifyOffline] = useState(true);
  const [isNew, setIsNew] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!mac) return;
    api.getHost(mac)
      .then((d) => {
        setData(d);
        setCustomName(d.host.customName || '');
        setCustomVendor(d.host.customVendor || '');
        setNotifyOffline(d.host.notifyOffline);
        setIsNew(d.host.isNew);
      })
      .catch((e) => toast.error(errMessage(e, 'Load failed')));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mac]);

  async function save() {
    if (!mac) return;
    setSaving(true);
    try {
      const updated = await api.updateHost(mac, { customName, customVendor, notifyOffline, isNew });
      setData((cur) => (cur ? { ...cur, host: updated } : cur));
      toast.success('Host updated');
    } catch (e) {
      toast.error(errMessage(e, 'Save failed'));
    } finally {
      setSaving(false);
    }
  }

  if (!data) return <p className="text-sm text-slate-500">Loading…</p>;

  const { host, events } = data;

  return (
    <div className="space-y-5">
      <Link to="/" className="text-sm text-slate-500 hover:underline">← Back</Link>
      <div className="rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold">{host.customName || host.hostname || host.ip}</h2>
          {host.isNew && (
            <span className="inline-block rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-800 dark:bg-amber-500/20 dark:text-amber-200">
              NEW
            </span>
          )}
        </div>
        <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
          <Row k="MAC" v={host.mac} />
          <Row k="IP" v={host.ip} />
          <Row k="VLAN" v={host.vlanId != null ? String(host.vlanId) : '—'} />
          <Row k="Vendor" v={host.customVendor || host.vendor || '—'} />
          <Row k="Status" v={host.online ? 'Online' : 'Offline'} />
          <Row k="Last seen" v={new Date(host.lastSeen * 1000).toLocaleString()} />
          <Row k="First seen" v={new Date(host.firstSeen * 1000).toLocaleString()} />
        </dl>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <h3 className="text-sm font-semibold">Custom settings</h3>
        <label className="mt-3 block space-y-1.5 text-sm">
          <span className="font-medium text-slate-700 dark:text-slate-300">Custom name</span>
          <input value={customName} onChange={(e) => setCustomName(e.target.value)} className="input" />
        </label>
        <label className="mt-3 block space-y-1.5 text-sm">
          <span className="font-medium text-slate-700 dark:text-slate-300">
            Custom vendor
            {host.vendor && <span className="ml-2 text-xs font-normal text-slate-500">(detected: {host.vendor})</span>}
          </span>
          <input value={customVendor} onChange={(e) => setCustomVendor(e.target.value)} className="input" placeholder={host.vendor || 'e.g. Living room TV'} />
        </label>
        <label className="mt-3 flex items-center gap-2 text-sm">
          <input type="checkbox" checked={notifyOffline} onChange={(e) => setNotifyOffline(e.target.checked)} />
          Notify when this host goes offline
        </label>
        <label className="mt-2 flex items-center gap-2 text-sm">
          <input type="checkbox" checked={isNew} onChange={(e) => setIsNew(e.target.checked)} />
          Mark as new (unacknowledged)
        </label>
        <button onClick={save} disabled={saving} className="btn-primary mt-4">
          {saving && <Spinner className="mr-2 -ml-1" />}
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <h3 className="text-sm font-semibold">Recent events</h3>
        {events.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500">No events recorded.</p>
        ) : (
          <ul className="mt-2 space-y-1 text-sm">
            {events.slice(0, 100).map((e) => (
              <li key={e.id} className="flex justify-between">
                <span>{eventLabel(e.kind)}{e.ip ? ` (${e.ip})` : ''}</span>
                <span className="text-slate-500">{new Date(e.ts * 1000).toLocaleString()}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <>
      <dt className="text-slate-500">{k}</dt>
      <dd className="truncate font-mono text-sm">{v}</dd>
    </>
  );
}

function eventLabel(k: string) {
  switch (k) {
    case 'new': return 'First seen';
    case 'online': return 'Came online';
    case 'offline': return 'Went offline';
    case 'ip_change': return 'IP changed';
    default: return k;
  }
}
