import { useEffect, useRef, useState } from 'react';
import { NetInterface } from '../lib/api';

export default function IfacePicker({
  value,
  ifaces,
  onChange,
  disabled = false,
  emptyHelpText = 'Empty = no interfaces will be scanned.',
}: {
  value: string[];
  ifaces: NetInterface[];
  onChange: (v: string[]) => void;
  disabled?: boolean;
  emptyHelpText?: string;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onDocClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    document.addEventListener('mousedown', onDocClick);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDocClick);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  function toggle(name: string) {
    if (value.includes(name)) onChange(value.filter((n) => n !== name));
    else onChange([...value, name]);
  }

  // Show any saved iface that's no longer active so the user can drop it.
  const stale = value.filter((n) => !ifaces.some((x) => x.name === n));

  let summary: string;
  if (value.length === 0) summary = 'No interfaces selected';
  else if (value.length <= 3) summary = value.join(', ');
  else summary = `${value.length} selected`;

  return (
    <div className="space-y-1.5">
      <span className="block text-sm font-medium text-slate-700 dark:text-slate-300">
        Interfaces to scan
      </span>
      <div ref={ref} className="relative">
        <button
          type="button"
          onClick={() => !disabled && setOpen((o) => !o)}
          disabled={disabled}
          className="input flex w-full items-center justify-between text-left disabled:cursor-not-allowed disabled:opacity-60"
          aria-haspopup="listbox"
          aria-expanded={open}
        >
          <span className={value.length === 0 ? 'text-slate-500' : ''}>{summary}</span>
          <svg className="ml-2 h-4 w-4 shrink-0 text-slate-400" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.06l3.71-3.83a.75.75 0 111.08 1.04l-4.25 4.39a.75.75 0 01-1.08 0L5.21 8.27a.75.75 0 01.02-1.06z" clipRule="evenodd" />
          </svg>
        </button>
        {open && (
          <div
            className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-900"
            role="listbox"
          >
            {ifaces.length === 0 && stale.length === 0 && (
              <p className="px-3 py-2 text-xs text-slate-500">No active interfaces detected.</p>
            )}
            {ifaces.map((ifc) => {
              const on = value.includes(ifc.name);
              return (
                <label
                  key={ifc.name}
                  className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
                >
                  <input
                    type="checkbox"
                    checked={on}
                    onChange={() => toggle(ifc.name)}
                  />
                  <span className="font-medium">{ifc.name}</span>
                  {ifc.addresses.length > 0 && (
                    <span className="ml-auto truncate text-xs text-slate-500">{ifc.addresses[0]}</span>
                  )}
                </label>
              );
            })}
            {stale.length > 0 && (
              <div className="border-t border-slate-200 dark:border-slate-700">
                {stale.map((name) => (
                  <label
                    key={name}
                    className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm hover:bg-slate-100 dark:hover:bg-slate-800"
                    title="Saved interface — not currently active. Uncheck to remove."
                  >
                    <input type="checkbox" checked onChange={() => toggle(name)} />
                    <span className="font-medium text-amber-700 dark:text-amber-300">{name}</span>
                    <span className="ml-auto text-xs text-amber-700/70 dark:text-amber-300/70">inactive</span>
                  </label>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
      <p className="text-xs text-slate-500">{emptyHelpText}</p>
    </div>
  );
}
