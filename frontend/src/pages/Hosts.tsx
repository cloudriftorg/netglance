import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import clsx from 'clsx';
import { api, Host } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';
import Spinner from '../components/Spinner';

function relativeTime(unix: number): string {
  const diff = Math.max(0, Math.floor(Date.now() / 1000) - unix);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export default function Hosts() {
  const navigate = useNavigate();
  const toast = useToast();
  const [hosts, setHosts] = useState<Host[]>([]);
  const [q, setQ] = useState('');
  const [filter, setFilter] = useState<'all' | 'online' | 'offline'>('all');
  const [vlan, setVlan] = useState<number | null>(null);
  const [scanning, setScanning] = useState(false);
  const wasScanning = useState({ value: false })[0];

  async function load() {
    try {
      const data = await api.listHosts({
        q: q || undefined,
        online: filter === 'all' ? null : filter === 'online',
        vlan,
      });
      setHosts(data ?? []);
    } catch (err) {
      toast.error(errMessage(err, 'Load failed'));
    }
  }

  async function pollStatus() {
    try {
      const s = await api.scanStatus();
      setScanning(s.running);
      if (wasScanning.value && !s.running) {
        toast.success('Scan complete');
      }
      wasScanning.value = s.running;
    } catch {
      /* network blip — ignore */
    }
  }

  useEffect(() => {
    load();
    pollStatus();
    const id = setInterval(() => {
      load();
      pollStatus();
    }, 5_000);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, filter, vlan]);

  const vlans = useMemo(() => {
    const set = new Set<number>();
    hosts.forEach((h) => h.vlanId != null && set.add(h.vlanId));
    return Array.from(set).sort((a, b) => a - b);
  }, [hosts]);

  async function runScan() {
    try {
      const r = await api.runScan();
      if (r.status === 'already-running') {
        toast.info('A scan is already running');
      } else {
        toast.info('Scan started');
      }
      setScanning(true);
      wasScanning.value = true;
    } catch (err) {
      toast.error(errMessage(err, 'Scan failed to start'));
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search MAC, IP, name…" className="input flex-1" />
        <button onClick={runScan} disabled={scanning} className="btn-primary sm:w-auto">
          {scanning && <Spinner className="mr-2 -ml-1" />}
          {scanning ? 'Scanning…' : 'Scan now'}
        </button>
      </div>

      <div className="flex flex-wrap items-center gap-2 text-sm">
        <FilterChip active={filter === 'all'} onClick={() => setFilter('all')}>All</FilterChip>
        <FilterChip active={filter === 'online'} onClick={() => setFilter('online')}>Online</FilterChip>
        <FilterChip active={filter === 'offline'} onClick={() => setFilter('offline')}>Offline</FilterChip>
        <span className="mx-1 text-slate-300">|</span>
        <FilterChip active={vlan === null} onClick={() => setVlan(null)}>Any VLAN</FilterChip>
        {vlans.map((v) => (
          <FilterChip key={v} active={vlan === v} onClick={() => setVlan(v)}>VLAN {v}</FilterChip>
        ))}
      </div>

      {hosts.length === 0 ? (
        <p className="rounded-lg border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500 dark:border-slate-700">
          No hosts yet. {scanning ? 'A scan is in progress…' : <>Try <button onClick={runScan} className="underline">running a scan</button> or check Settings → Networks.</>}
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900">
          <table className="w-full table-fixed text-sm">
            <colgroup>
              <col />
              <col className="w-36" />
              <col className="hidden w-44 md:table-column" />
              <col className="hidden w-56 lg:table-column" />
              <col className="w-24" />
              <col className="hidden w-28 sm:table-column" />
              <col className="w-28" />
            </colgroup>
            <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:border-slate-800 dark:bg-slate-800/50 dark:text-slate-400">
              <tr>
                <th className="px-3 py-2.5 text-left">Name</th>
                <th className="px-3 py-2.5 text-left">IP</th>
                <th className="hidden px-3 py-2.5 text-left md:table-cell">MAC</th>
                <th className="hidden px-3 py-2.5 text-left lg:table-cell">Vendor</th>
                <th className="px-3 py-2.5 text-left">VLAN</th>
                <th className="hidden px-3 py-2.5 text-left sm:table-cell">Last seen</th>
                <th className="px-3 py-2.5 text-right">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {hosts.map((h) => (
                <tr
                  key={h.mac}
                  onClick={() => navigate(`/h/${h.mac}`)}
                  className="h-12 cursor-pointer align-middle transition-colors hover:bg-slate-50 dark:hover:bg-slate-800/40"
                >
                  <td className="truncate px-3 font-medium">
                    <div className="flex items-center gap-2">
                      <span className="truncate">{h.customName || h.hostname || h.ip}</span>
                      {h.isNew && (
                        <span className="shrink-0 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-800 dark:bg-amber-500/20 dark:text-amber-200">
                          NEW
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="truncate px-3 font-mono text-xs text-slate-700 dark:text-slate-300">{h.ip}</td>
                  <td className="hidden truncate px-3 font-mono text-xs text-slate-700 dark:text-slate-300 md:table-cell">{h.mac}</td>
                  <td className="hidden truncate px-3 text-slate-600 dark:text-slate-400 lg:table-cell" title={h.customVendor || h.vendor || ''}>
                    {h.customVendor || h.vendor || '—'}
                  </td>
                  <td className="px-3">
                    {h.vlanId != null ? (
                      <span className="inline-block rounded-full bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-700/20 dark:text-brand-50">
                        VLAN {h.vlanId}
                      </span>
                    ) : (
                      <span className="text-xs text-slate-400">—</span>
                    )}
                  </td>
                  <td className="hidden truncate px-3 text-xs text-slate-500 sm:table-cell" title={new Date(h.lastSeen * 1000).toLocaleString()}>
                    {relativeTime(h.lastSeen)}
                  </td>
                  <td className="px-3 text-right">
                    <span
                      className={clsx(
                        'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium',
                        h.online
                          ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-200'
                          : 'bg-slate-100 text-slate-600 dark:bg-slate-700/50 dark:text-slate-300'
                      )}
                    >
                      <span className={clsx('h-1.5 w-1.5 rounded-full', h.online ? 'bg-emerald-500' : 'bg-slate-400')} />
                      {h.online ? 'Online' : 'Offline'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function FilterChip({ active, children, onClick }: { active: boolean; children: React.ReactNode; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={clsx(
        'rounded-full border px-3 py-1 text-xs transition',
        active
          ? 'border-brand-500 bg-brand-500 text-white'
          : 'border-slate-300 text-slate-600 hover:border-slate-400 dark:border-slate-700 dark:text-slate-300'
      )}
    >
      {children}
    </button>
  );
}
