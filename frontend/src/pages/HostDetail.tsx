import { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { api, Host, HostEvent } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';
import { useConfirm } from '../components/Confirm';
import Spinner from '../components/Spinner';

export default function HostDetail() {
  const toast = useToast();
  const confirm = useConfirm();
  const navigate = useNavigate();
  const { mac } = useParams<{ mac: string }>();
  const [data, setData] = useState<{ host: Host; events: HostEvent[] } | null>(null);
  const [customName, setCustomName] = useState('');
  const [customVendor, setCustomVendor] = useState('');
  const [notifyOffline, setNotifyOffline] = useState(true);
  const [isNew, setIsNew] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);

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
    if (!mac || !data) return;
    setSaving(true);
    try {
      await api.updateHost(mac, { customName, customVendor, notifyOffline, isNew });
      toast.success('Host updated');
      navigate(-1);
    } catch (e) {
      toast.error(errMessage(e, 'Save failed'));
      setSaving(false);
    }
  }

  async function deleteHost() {
    if (!mac) return;
    const ok = await confirm({
      title: 'Delete this host?',
      message: 'It will reappear flagged as NEW the next time it answers a scan.',
      confirmLabel: 'Delete',
      danger: true,
    });
    if (!ok) return;
    setDeleting(true);
    try {
      await api.deleteHost(mac);
      toast.success('Host deleted');
      navigate('/');
    } catch (e) {
      toast.error(errMessage(e, 'Delete failed'));
      setDeleting(false);
    }
  }

  if (!data) return <p className="text-sm text-slate-500">Loading…</p>;

  const { host, events } = data;
  const displayName = host.customName || host.ip;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto pb-4">
      <div className="flex items-center justify-between">
        <Link to="/" className="text-sm text-slate-500 hover:underline">← Back</Link>
        <button onClick={save} disabled={saving} className="btn-primary">
          {saving && <Spinner className="mr-2 -ml-1" />}
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>

      <div className="rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-lg font-semibold">{displayName}</h2>
          <span
            className={clsx(
              'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium',
              host.online
                ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-200'
                : 'bg-slate-100 text-slate-600 dark:bg-slate-700/50 dark:text-slate-300',
            )}
          >
            <span className={clsx('h-1.5 w-1.5 rounded-full', host.online ? 'bg-emerald-500' : 'bg-slate-400')} />
            {host.online ? 'Online' : 'Offline'}
          </span>
          {host.isNew && (
            <span className="rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-800 dark:bg-amber-500/20 dark:text-amber-200">
              NEW
            </span>
          )}
        </div>
        <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-3 lg:grid-cols-4">
          <Field k="IP" v={host.ip} mono />
          <Field k="VLAN" v={host.vlanId != null ? String(host.vlanId) : '—'} />
          <Field k="MAC" v={host.mac} mono />
          <Field k="Vendor" v={host.customVendor || host.vendor || '—'} />
          <Field k="First seen" v={new Date(host.firstSeen * 1000).toLocaleString()} />
          <Field k="Last seen" v={new Date(host.lastSeen * 1000).toLocaleString()} />
        </dl>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <section className="space-y-3 rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
          <h3 className="text-sm font-semibold">Custom settings</h3>
          <label className="block space-y-1.5 text-sm">
            <span className="font-medium text-slate-700 dark:text-slate-300">Custom name</span>
            <input value={customName} onChange={(e) => setCustomName(e.target.value)} className="input" />
          </label>
          <label className="block space-y-1.5 text-sm">
            <span className="font-medium text-slate-700 dark:text-slate-300">
              Custom vendor
              {host.vendor && <span className="ml-2 text-xs font-normal text-slate-500">(detected: {host.vendor})</span>}
            </span>
            <input
              value={customVendor}
              onChange={(e) => setCustomVendor(e.target.value)}
              className="input"
              placeholder={host.vendor || 'e.g. Living room TV'}
            />
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={notifyOffline} onChange={(e) => setNotifyOffline(e.target.checked)} />
            Notify when this host goes offline
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={isNew} onChange={(e) => setIsNew(e.target.checked)} />
            Flag as new
          </label>
        </section>

        <div className="flex h-full flex-col gap-4">
          <section className="flex-1 rounded-2xl border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
            <h3 className="text-sm font-semibold">Recent events</h3>
            {events.length === 0 ? (
              <p className="mt-2 text-sm text-slate-500">No events recorded.</p>
            ) : (
              <ul className="mt-2 max-h-80 divide-y divide-slate-100 overflow-y-auto text-sm dark:divide-slate-800">
                {events.slice(0, 100).map((e) => (
                  <li key={e.id} className="flex items-center justify-between gap-4 py-1.5">
                    <span className="truncate">
                      {eventLabel(e.kind)}
                      {e.ip ? <span className="ml-1 font-mono text-xs text-slate-500">({e.ip})</span> : null}
                    </span>
                    <span className="shrink-0 text-xs text-slate-500">
                      {new Date(e.ts * 1000).toLocaleString()}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="rounded-2xl border border-red-200 bg-red-50/50 p-4 dark:border-red-500/30 dark:bg-red-500/5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold text-red-700 dark:text-red-300">Danger zone</h3>
                <p className="mt-0.5 text-xs text-red-700/80 dark:text-red-300/80">
                  Removes the host and its history. It will reappear flagged as NEW the next time it answers a scan.
                </p>
              </div>
              <button
                type="button"
                onClick={deleteHost}
                disabled={deleting}
                className="rounded-md border border-red-300 bg-white px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-100 disabled:opacity-60 dark:border-red-500/40 dark:bg-transparent dark:text-red-300 dark:hover:bg-red-500/10"
              >
                {deleting ? 'Deleting…' : 'Delete this host'}
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

function Field({ k, v, mono }: { k: string; v: string; mono?: boolean }) {
  return (
    <div className="min-w-0">
      <dt className="text-xs uppercase tracking-wide text-slate-500">{k}</dt>
      <dd className={clsx('truncate text-sm', mono && 'font-mono text-xs')} title={v}>
        {v}
      </dd>
    </div>
  );
}

function eventLabel(k: string) {
  switch (k) {
    case 'new':
      return 'First seen';
    case 'online':
      return 'Came online';
    case 'offline':
      return 'Went offline';
    case 'ip_change':
      return 'IP changed';
    default:
      return k;
  }
}
