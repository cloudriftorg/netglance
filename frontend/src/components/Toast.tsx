import { createContext, ReactNode, useCallback, useContext, useRef, useState } from 'react';
import clsx from 'clsx';

export type ToastKind = 'info' | 'success' | 'error';

interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

interface Ctx {
  toast: (message: string, kind?: ToastKind) => void;
  success: (message: string) => void;
  error: (message: string) => void;
  info: (message: string) => void;
}

const ToastCtx = createContext<Ctx | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Toast[]>([]);
  const idRef = useRef(0);

  const dismiss = useCallback((id: number) => {
    setItems((cur) => cur.filter((t) => t.id !== id));
  }, []);

  const toast = useCallback(
    (message: string, kind: ToastKind = 'info') => {
      const id = ++idRef.current;
      setItems((cur) => [...cur, { id, kind, message }]);
      const ttl = kind === 'error' ? 6000 : 3500;
      setTimeout(() => dismiss(id), ttl);
    },
    [dismiss]
  );

  const value: Ctx = {
    toast,
    success: (m) => toast(m, 'success'),
    error: (m) => toast(m, 'error'),
    info: (m) => toast(m, 'info'),
  };

  return (
    <ToastCtx.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed bottom-0 right-0 z-50 flex flex-col items-end gap-2 px-3 pb-3">
        {items.map((t) => (
          <div
            key={t.id}
            role="status"
            onClick={() => dismiss(t.id)}
            className={clsx(
              'pointer-events-auto flex w-full max-w-sm cursor-pointer items-start gap-2.5 rounded-lg px-4 py-2.5 text-sm font-medium shadow-lg ring-1 transition-all backdrop-blur',
              t.kind === 'success' &&
                'bg-emerald-600 text-white ring-emerald-700/20 dark:bg-emerald-500/15 dark:text-emerald-100 dark:ring-emerald-400/30',
              t.kind === 'error' &&
                'bg-red-600 text-white ring-red-700/20 dark:bg-red-500/15 dark:text-red-100 dark:ring-red-400/30',
              t.kind === 'info' &&
                'bg-sky-600 text-white ring-sky-700/20 dark:bg-sky-500/15 dark:text-sky-100 dark:ring-sky-400/30'
            )}
          >
            <ToastIcon kind={t.kind} />
            <span className="flex-1">{t.message}</span>
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  );
}

export function useToast(): Ctx {
  const ctx = useContext(ToastCtx);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
}

export function errMessage(e: unknown, fallback = 'Something went wrong'): string {
  if (e instanceof Error) return e.message || fallback;
  if (typeof e === 'string') return e;
  return fallback;
}

function ToastIcon({ kind }: { kind: ToastKind }) {
  if (kind === 'success') {
    return (
      <svg className="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
        <path fillRule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm3.7-9.7a1 1 0 0 0-1.4-1.4L9 10.2 7.7 8.9a1 1 0 0 0-1.4 1.4l2 2a1 1 0 0 0 1.4 0l4-4Z" clipRule="evenodd" />
      </svg>
    );
  }
  if (kind === 'error') {
    return (
      <svg className="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
        <path fillRule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16ZM10 6a1 1 0 0 1 1 1v3a1 1 0 1 1-2 0V7a1 1 0 0 1 1-1Zm0 8a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z" clipRule="evenodd" />
      </svg>
    );
  }
  return (
    <svg className="mt-0.5 h-4 w-4 shrink-0" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
      <path fillRule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16ZM9 9a1 1 0 0 1 2 0v4a1 1 0 1 1-2 0V9Zm1-3.25a1.25 1.25 0 1 0 0 2.5 1.25 1.25 0 0 0 0-2.5Z" clipRule="evenodd" />
    </svg>
  );
}
