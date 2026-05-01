import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import clsx from 'clsx';
import { api, Host, Scan } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';
import { useConfirm } from '../components/Confirm';
import Spinner from '../components/Spinner';

export default function Hosts() {
  const navigate = useNavigate();
  const toast = useToast();
  const confirm = useConfirm();
  const [hosts, setHosts] = useState<Host[]>([]);
  const [q, setQ] = useState('');
  const [filter, setFilter] = useState<'all' | 'online' | 'offline'>('all');
  const [vlan, setVlan] = useState<number | null>(null);
  const [scanning, setScanning] = useState(false);
  const [lastScan, setLastScan] = useState<Scan | null>(null);
  const [nextScanAt, setNextScanAt] = useState<number | null>(null);
  const [sort, setSort] = useState<{ key: SortKey; dir: 'asc' | 'desc' } | null>(loadSortPref);

  // Persist sort preference across reloads. `null` (unsorted) clears the
  // saved key so a fresh load really does come up unsorted.
  useEffect(() => {
    try {
      if (sort) {
        localStorage.setItem('netglance.hosts.sort', JSON.stringify(sort));
      } else {
        localStorage.removeItem('netglance.hosts.sort');
      }
    } catch {
      /* storage disabled / quota — silent */
    }
  }, [sort]);
  const wasScanning = useState({ value: false })[0];
  // Tracks whether the current/most-recent scan was kicked off by the user
  // clicking "Scan now". The completion toast only fires for those — auto
  // scans run silently in the background.
  const manualScan = useState({ value: false })[0];

  async function load() {
    try {
      // online/vlan filters are applied client-side so the chip counters can
      // reflect the full q-scoped list without re-querying when toggling.
      const data = await api.listHosts({ q: q || undefined });
      setHosts(data ?? []);
    } catch (err) {
      toast.error(errMessage(err, 'Load failed'));
    }
  }

  async function pollStatus() {
    try {
      const s = await api.scanStatus();
      if (s.lastScan) setLastScan(s.lastScan);
      setNextScanAt(s.nextScanAt ?? null);
      const justFinished = wasScanning.value && !s.running;
      if (justFinished) {
        // Refresh the host list before clearing the spinner so the user sees
        // the new records appear together with the completion toast.
        await load();
        if (manualScan.value) toast.success('Scan complete');
        manualScan.value = false;
      }
      wasScanning.value = s.running;
      setScanning(s.running);
    } catch {
      /* network blip — ignore */
    }
  }

  useEffect(() => {
    load();
    pollStatus();
    // Poll faster while a scan is running so the transition to "complete"
    // and the refreshed list are reflected promptly in the UI.
    const interval = scanning ? 1_000 : 5_000;
    const id = setInterval(() => {
      load();
      pollStatus();
    }, interval);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, scanning]);

  const counts = useMemo(() => {
    const byVlan = new Map<number, number>();
    let online = 0;
    let offline = 0;
    for (const h of hosts) {
      if (h.online) online++;
      else offline++;
      if (h.vlanId != null) byVlan.set(h.vlanId, (byVlan.get(h.vlanId) ?? 0) + 1);
    }
    return { total: hosts.length, online, offline, byVlan };
  }, [hosts]);

  const vlans = useMemo(
    () => Array.from(counts.byVlan.keys()).sort((a, b) => a - b),
    [counts],
  );

  const filteredHosts = useMemo(() => {
    return hosts.filter((h) => {
      if (filter === 'online' && !h.online) return false;
      if (filter === 'offline' && h.online) return false;
      if (vlan != null && h.vlanId !== vlan) return false;
      return true;
    });
  }, [hosts, filter, vlan]);

  const sortedHosts = useMemo(() => {
    if (!sort) return filteredHosts;
    const arr = filteredHosts.slice();
    arr.sort((a, b) => compareHosts(a, b, sort.key));
    if (sort.dir === 'desc') arr.reverse();
    return arr;
  }, [filteredHosts, sort]);

  // Three-state cycle on a header: unsorted/other → asc → desc → unsorted.
  function clickSort(key: SortKey) {
    setSort((cur) => {
      if (!cur || cur.key !== key) return { key, dir: 'asc' };
      if (cur.dir === 'asc') return { key, dir: 'desc' };
      return null;
    });
  }

  // Click on the NEW badge acknowledges the host: the badge disappears and
  // the host is no longer flagged. There's no UI to set the flag back on —
  // it only flips on automatically when arp-scan first sees a MAC.
  async function acknowledgeNew(h: Host) {
    if (!h.isNew) return;
    try {
      await api.updateHost(h.mac, {
        customName: h.customName || '',
        customVendor: h.customVendor || '',
        notifyOffline: h.notifyOffline,
        isNew: false,
      });
      setHosts((cur) => cur.map((x) => (x.mac === h.mac ? { ...x, isNew: false } : x)));
    } catch (err) {
      toast.error(errMessage(err, 'Update failed'));
    }
  }

  async function deleteHost(h: Host) {
    const label = h.customName || h.ip;
    const ok = await confirm({
      title: `Delete host "${label}"?`,
      message: 'It will reappear flagged as NEW the next time it answers a scan.',
      confirmLabel: 'Delete',
      danger: true,
    });
    if (!ok) return;
    try {
      await api.deleteHost(h.mac);
      setHosts((cur) => cur.filter((x) => x.mac !== h.mac));
      toast.success('Host deleted');
    } catch (err) {
      toast.error(errMessage(err, 'Delete failed'));
    }
  }

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
      <div className="flex items-center gap-2">
        <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search MAC, IP, name…" className="input flex-1" />
        <button
          onClick={runScan}
          disabled={scanning}
          aria-label={scanning ? 'Scan in progress' : 'Scan now'}
          title={scanning ? 'Scan in progress' : 'Scan now'}
          className="btn-primary inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full p-0 sm:h-10 sm:w-10"
        >
          {scanning ? <Spinner className="h-4 w-4" /> : <RefreshIcon className="h-4 w-4" />}
        </button>
      </div>

      {/* Mobile-only: badge on its own row, inheriting parent's space-y gap */}
      <div className="sm:hidden">
        <LastScanBadge scan={lastScan} scanning={scanning} />
        <NextScanBadge nextAt={nextScanAt} scanning={scanning} />
      </div>

      <div className="flex flex-wrap items-center gap-2 text-sm">
        <FilterChip active={filter === 'all'} count={counts.total} onClick={() => setFilter('all')}>All</FilterChip>
        <FilterChip active={filter === 'online'} count={counts.online} onClick={() => setFilter('online')}>Online</FilterChip>
        <FilterChip active={filter === 'offline'} count={counts.offline} onClick={() => setFilter('offline')}>Offline</FilterChip>
        <span className="mx-1 text-slate-300">|</span>
        <FilterChip active={vlan === null} count={counts.total} onClick={() => setVlan(null)}>Any VLAN</FilterChip>
        {vlans.map((v) => (
          <FilterChip key={v} active={vlan === v} count={counts.byVlan.get(v) ?? 0} onClick={() => setVlan(v)}>VLAN {v}</FilterChip>
        ))}
        {/* Desktop-only: badge right-aligned on the filter row */}
        <LastScanBadge scan={lastScan} scanning={scanning} className="ml-auto hidden sm:inline-flex" />
        <NextScanBadge nextAt={nextScanAt} scanning={scanning} className="hidden sm:inline-flex" />
      </div>

      {hosts.length === 0 ? (
        <p className="rounded-lg border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500 dark:border-slate-700">
          No hosts yet. {scanning ? 'A scan is in progress…' : <>Try <button onClick={runScan} className="underline">running a scan</button> or check Settings → Networks.</>}
        </p>
      ) : (
        <>
        {/* Mobile: card list */}
        <ul className="space-y-2 sm:hidden">
          {sortedHosts.map((h) => (
            <li
              key={h.mac}
              className="rounded-xl border border-slate-200 bg-white p-3 shadow-sm dark:border-slate-800 dark:bg-slate-900"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => navigate(`/h/${h.mac}`)}
                      className="min-w-0 truncate text-left font-medium hover:text-brand-600 hover:underline dark:hover:text-brand-400"
                    >
                      {h.customName || h.ip}
                    </button>
                    {h.isNew && <NewBadge onClick={() => acknowledgeNew(h)} />}
                  </div>
                  <div className="mt-0.5 truncate font-mono text-xs text-slate-500 dark:text-slate-400">
                    {h.ip}
                    <span className="mx-1 text-slate-300 dark:text-slate-600">·</span>
                    {h.mac}
                  </div>
                  {(h.customVendor || h.vendor) && (
                    <div className="mt-0.5 truncate text-xs text-slate-500 dark:text-slate-400">
                      {h.customVendor || h.vendor}
                    </div>
                  )}
                </div>
                <button
                  onClick={() => deleteHost(h)}
                  title="Delete host"
                  aria-label="Delete host"
                  className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-slate-400 transition hover:bg-red-100 hover:text-red-600 dark:hover:bg-red-500/20 dark:hover:text-red-300"
                >
                  <TrashIcon />
                </button>
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <span
                  className={clsx(
                    'inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium',
                    h.online
                      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-500/20 dark:text-emerald-200'
                      : 'bg-slate-100 text-slate-600 dark:bg-slate-700/50 dark:text-slate-300',
                  )}
                >
                  <span className={clsx('h-1.5 w-1.5 rounded-full', h.online ? 'bg-emerald-500' : 'bg-slate-400')} />
                  {h.online ? 'Online' : 'Offline'}
                </span>
                {h.vlanId != null && (
                  <span className="inline-block rounded-full bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-700/20 dark:text-brand-50">
                    VLAN {h.vlanId}
                  </span>
                )}
              </div>
            </li>
          ))}
        </ul>

        {/* Desktop: table */}
        <div className="hidden overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-900 sm:block">
          <table className="w-full table-fixed text-sm">
            <colgroup>
              <col className="w-52" />
              <col className="w-32" />
              <col className="w-20" />
              <col className="hidden w-40 md:table-column" />
              <col className="hidden w-60 lg:table-column" />
              <col className="w-24" />
              <col className="w-16" />
            </colgroup>
            <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:border-slate-800 dark:bg-slate-800/50 dark:text-slate-400">
              <tr>
                <SortableTh label="Name" sortKey="name" sort={sort} onClick={clickSort} />
                <SortableTh label="IP" sortKey="ip" sort={sort} onClick={clickSort} />
                <SortableTh label="VLAN" sortKey="vlan" sort={sort} onClick={clickSort} />
                <SortableTh label="MAC" sortKey="mac" sort={sort} onClick={clickSort} className="hidden md:table-cell" />
                <SortableTh label="Vendor" sortKey="vendor" sort={sort} onClick={clickSort} className="hidden lg:table-cell" />
                <SortableTh label="Status" sortKey="status" sort={sort} onClick={clickSort} />
                <th className="px-3 py-2.5"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
              {sortedHosts.map((h) => (
                <tr key={h.mac} className="h-12 align-middle transition-colors hover:bg-slate-50 dark:hover:bg-slate-800/40">
                  <td className="px-3 font-medium">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => navigate(`/h/${h.mac}`)}
                        className="min-w-0 truncate text-left underline-offset-2 hover:text-brand-600 hover:underline dark:hover:text-brand-400"
                      >
                        {h.customName || h.ip}
                      </button>
                      {h.isNew && <NewBadge onClick={() => acknowledgeNew(h)} />}
                    </div>
                  </td>
                  <td className="truncate px-3 font-mono text-xs text-slate-700 dark:text-slate-300">{h.ip}</td>
                  <td className="px-3">
                    {h.vlanId != null ? (
                      <span className="inline-block rounded-full bg-brand-50 px-2 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-700/20 dark:text-brand-50">
                        VLAN {h.vlanId}
                      </span>
                    ) : (
                      <span className="text-xs text-slate-400">—</span>
                    )}
                  </td>
                  <td className="hidden truncate px-3 font-mono text-xs text-slate-700 dark:text-slate-300 md:table-cell">{h.mac}</td>
                  <td className="hidden truncate px-3 text-slate-600 dark:text-slate-400 lg:table-cell" title={h.customVendor || h.vendor || ''}>
                    {h.customVendor || h.vendor || '—'}
                  </td>
                  <td className="px-3 text-left">
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
                  <td className="px-3">
                    <div className="flex justify-end">
                      <button
                        onClick={() => deleteHost(h)}
                        title="Delete host"
                        aria-label="Delete host"
                        className="inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-400 transition hover:bg-red-100 hover:text-red-600 dark:hover:bg-red-500/20 dark:hover:text-red-300"
                      >
                        <TrashIcon />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        </>
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

function NextScanBadge({
  nextAt,
  scanning,
  className,
}: {
  nextAt: number | null;
  scanning: boolean;
  className?: string;
}) {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    if (scanning || nextAt == null) return;
    const id = setInterval(() => setNow(Math.floor(Date.now() / 1000)), 1000);
    return () => clearInterval(id);
  }, [scanning, nextAt]);

  if (scanning || nextAt == null) return null;

  const remaining = Math.max(0, nextAt - now);
  const m = Math.floor(remaining / 60);
  const s = remaining % 60;
  const label = remaining === 0 ? 'imminent' : `${m}:${s.toString().padStart(2, '0')}`;

  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium',
        'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300',
        className,
      )}
      title="Time until next automatic scan"
    >
      <ClockIcon />
      Next in {label}
    </span>
  );
}

function ClockIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 2" />
    </svg>
  );
}

type SortKey = 'name' | 'ip' | 'vlan' | 'mac' | 'vendor' | 'status';
const SORT_KEYS: SortKey[] = ['name', 'ip', 'vlan', 'mac', 'vendor', 'status'];

function loadSortPref(): { key: SortKey; dir: 'asc' | 'desc' } | null {
  try {
    const raw = localStorage.getItem('netglance.hosts.sort');
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { key?: string; dir?: string };
    if (!SORT_KEYS.includes(parsed.key as SortKey)) return null;
    if (parsed.dir !== 'asc' && parsed.dir !== 'desc') return null;
    return { key: parsed.key as SortKey, dir: parsed.dir };
  } catch {
    return null;
  }
}

function nameOf(h: Host): string {
  return (h.customName || h.ip).toLowerCase();
}

function vendorOf(h: Host): string {
  return (h.customVendor || h.vendor || '').toLowerCase();
}

// ipToInt maps an IPv4 string to an integer for proper numeric ordering
// (so 192.168.1.10 sorts after 192.168.1.9, not lexicographically before).
function ipToInt(ip: string): number {
  const p = ip.split('.');
  if (p.length !== 4) return Number.MAX_SAFE_INTEGER;
  let n = 0;
  for (const part of p) {
    const v = Number(part);
    if (!Number.isFinite(v)) return Number.MAX_SAFE_INTEGER;
    n = n * 256 + v;
  }
  return n;
}

function compareHosts(a: Host, b: Host, key: SortKey): number {
  switch (key) {
    case 'name':
      return nameOf(a).localeCompare(nameOf(b));
    case 'ip':
      return ipToInt(a.ip) - ipToInt(b.ip);
    case 'vlan': {
      const av = a.vlanId ?? Number.MAX_SAFE_INTEGER;
      const bv = b.vlanId ?? Number.MAX_SAFE_INTEGER;
      return av - bv;
    }
    case 'mac':
      return a.mac.localeCompare(b.mac);
    case 'vendor':
      return vendorOf(a).localeCompare(vendorOf(b));
    case 'status':
      return Number(b.online) - Number(a.online); // online first
    default:
      return 0;
  }
}

function SortableTh({
  label,
  sortKey,
  sort,
  onClick,
  align,
  className,
}: {
  label: string;
  sortKey: SortKey;
  sort: { key: SortKey; dir: 'asc' | 'desc' } | null;
  onClick: (k: SortKey) => void;
  align?: 'left' | 'center' | 'right';
  className?: string;
}) {
  const active = sort?.key === sortKey;
  const justify = align === 'center' ? 'justify-center' : align === 'right' ? 'justify-end' : 'justify-start';
  return (
    <th className={clsx('px-3 py-2.5 text-left', className)}>
      <button
        onClick={() => onClick(sortKey)}
        className={clsx(
          'inline-flex w-full items-center gap-1 text-xs font-semibold uppercase tracking-wide transition-colors hover:text-slate-900 dark:hover:text-slate-200',
          justify,
          active ? 'text-slate-900 dark:text-slate-100' : 'text-slate-500 dark:text-slate-400',
        )}
      >
        <span>{label}</span>
        <SortIndicator dir={active ? sort!.dir : null} />
      </button>
    </th>
  );
}

function SortIndicator({ dir }: { dir: 'asc' | 'desc' | null }) {
  if (!dir) {
    return (
      <svg className="h-3 w-3 opacity-30" viewBox="0 0 12 12" fill="currentColor" aria-hidden="true">
        <path d="M3 5l3-3 3 3z" />
        <path d="M3 7l3 3 3-3z" />
      </svg>
    );
  }
  return (
    <svg className="h-3 w-3" viewBox="0 0 12 12" fill="currentColor" aria-hidden="true">
      {dir === 'asc' ? <path d="M3 7l3-4 3 4z" /> : <path d="M3 5l3 4 3-4z" />}
    </svg>
  );
}

function NewBadge({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      title="Acknowledge — clears the NEW flag"
      aria-label="Acknowledge new host"
      className="shrink-0 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-800 transition hover:bg-amber-200 dark:bg-amber-500/20 dark:text-amber-200 dark:hover:bg-amber-500/30"
    >
      NEW
    </button>
  );
}

function TrashIcon() {
  return (
    <svg
      className="h-4 w-4"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M3 6h18" />
      <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
      <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
      <path d="M10 11v6" />
      <path d="M14 11v6" />
    </svg>
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

function FilterChip({
  active,
  count,
  children,
  onClick,
}: {
  active: boolean;
  count?: number;
  children: React.ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs transition',
        active
          ? 'border-brand-500 bg-brand-500 text-white'
          : 'border-slate-300 text-slate-600 hover:border-slate-400 dark:border-slate-700 dark:text-slate-300'
      )}
    >
      <span>{children}</span>
      {count != null && (
        <span
          className={clsx(
            'rounded-full px-1.5 py-0.5 text-[10px] font-semibold leading-none',
            active
              ? 'bg-white/20 text-white'
              : 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300',
          )}
        >
          {count}
        </span>
      )}
    </button>
  );
}
