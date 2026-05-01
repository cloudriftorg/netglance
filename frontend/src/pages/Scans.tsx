import { useEffect, useState } from 'react';
import { api, Scan } from '../lib/api';
import { errMessage, useToast } from '../components/Toast';

export default function Scans() {
  const toast = useToast();
  const [scans, setScans] = useState<Scan[]>([]);

  useEffect(() => {
    api.listScans()
      .then((s) => setScans(s ?? []))
      .catch((e) => toast.error(errMessage(e, 'Load failed')));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="space-y-3">
      <h2 className="text-lg font-semibold">Recent scans</h2>
      {scans.length === 0 ? (
        <p className="text-sm text-slate-500">No scans yet.</p>
      ) : (
        <ul className="space-y-2">
          {scans.map((s) => {
            const dur = s.endedAt ? `${s.endedAt - s.startedAt}s` : 'running…';
            return (
              <li key={s.id} className="flex items-center justify-between rounded-xl border border-slate-200 bg-white p-3 text-sm dark:border-slate-800 dark:bg-slate-900">
                <div>
                  <div className="font-medium">{new Date(s.startedAt * 1000).toLocaleString()}</div>
                  <div className="text-xs text-slate-500">duration {dur} · {s.hostsFound} hosts</div>
                  {s.error && <div className="mt-0.5 text-xs text-red-600">{s.error}</div>}
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
