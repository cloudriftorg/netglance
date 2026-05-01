import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import clsx from 'clsx';
import { api, Host, Scan } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';
import Spinner from '../components/Spinner';

export default function Hosts() {
  const navigate = useNavigate();
  const toast = useToast();
  const [hosts, setHosts] = useState<Host[]>([]);
  const [q, setQ] = useState('');
  const [filter, setFilter] = useState<'all' | 'online' | 'offline'>('all');
  const [vlan, setVlan] = useState<number | null>(null);
  const [scanning, setScanning] = useState(false);
  const [lastScan, setLastScan] = useState<Scan | null>(null);
  const wasScanning = useState({ value: false })[0];
  // Tracks whether the current/most-recent scan was kicked off by the user
  // clicking "Scan now". The completion toast only fires for those — auto
  // scans run silently in the background.
  const manualScan = useState({ value: false })[0];

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
      if (s.lastScan) setLastScan(s.lastScan);
      if (wasScanning.value && !s.running) {
        if (manualScan.value) toast.success('Scan complete');
        manualScan.value = false;
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
        // Don't claim ownership of the in-flight scan — it's likely the auto
        // one, and the completion toast would be misleading.
      } else {
        toast.info('Scan started');
        manualScan.value = true;
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
        <button
          onClick={runScan}
          disabled={scanning}
          aria-label={scanning ? 'Scan in progress' : 'Scan now'}
          title={scanning ? 'Scan in progress' : 'Scan now'}
          className="btn-primary inline-flex h-9 w-9 items-center justify-center rounded-full p-0 sm:h-10 sm:w-10"
        >
          {scanning ? <Spinner className="h-4 w-4" /> : <RefreshIcon className="h-4 w-4" />}
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
        <LastScanBadge scan={lastScan} scanning={scanning} className="ml-auto" />
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
              <col className="w-28" />
            </colgroup>
            <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:border-slate-800 dark:bg-slate-800/50 dark:text-slate-400">
              <tr>
                <th className="px-3 py-2.5 text-left">Name</th>
                <th className="px-3 py-2.5 text-left">IP</th>
                <th className="hidden px-3 py-2.5 text-left md:table-cell">MAC</th>
                <th className="hidden px-3 py-2.5 text-left lg:table-cell">Vendor</th>
                <th className="px-3 py-2.5 text-left">VLAN</th>
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

function LastScanBadge({ scan, scanning, className }: { scan: Scan | null; scanning: boolean; className?: string }) {
  const base = 'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium';
  if (scanning) {
    return (
      <span
        className={clsx(
          base,
          'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200',
          className,
        )}
      >
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
        Scanning…
      </span>
    );
  }
  if (!scan || !scan.endedAt) {
    return (
      <span
        className={clsx(
          base,
          'border-slate-200 bg-slate-50 text-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-400',
          className,
        )}
      >
        No scan yet
      </span>
    );
  }
  const d = new Date(scan.endedAt * 1000);
  const formatted = d.toLocaleString(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
  return (
    <span
      className={clsx(
        base,
        'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300',
        className,
      )}
      title={`${scan.hostsFound} hosts`}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
      Last scan {formatted}
    </span>
  );
}

function RefreshIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M21 12a9 9 0 1 1-3.36-7" />
      <path d="M21 4v6h-6" />
    </svg>
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
