import { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';
import clsx from 'clsx';
import { RefreshCw } from 'lucide-react';
import ThemeToggle from './ThemeToggle';

interface Props {
  children: ReactNode;
  onLogout: () => void;
}

export default function Layout({ children, onLogout }: Props) {
  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header
        className="z-10 shrink-0 border-b border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900"
        // Respect the iOS status bar / notch when installed as a PWA on
        // the home screen. viewport-fit=cover (set in index.html) extends
        // the page under the notch; without this padding the title and
        // tabs slid under the carrier/battery indicators.
        style={{ paddingTop: 'env(safe-area-inset-top)' }}
      >
        <div className="flex w-full flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <div className="flex items-center gap-2 sm:flex-1">
            <h1 className="text-lg font-semibold tracking-tight">Netglance</h1>
            {/* Mobile-only page refresh — installed-PWA users have no
                browser reload button, so the title bar gets one shaped
                like the Logout / theme toggle for visual consistency.
                Hard reload bypasses the SW cache so a stuck stale UI
                comes back fresh. */}
            <span aria-hidden className="h-5 w-px bg-slate-200 dark:bg-slate-700 sm:hidden" />
            <button
              type="button"
              onClick={() => window.location.reload()}
              aria-label="Refresh page"
              title="Refresh page"
              className="rounded-md px-2 py-1 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 dark:hover:bg-slate-800 dark:hover:text-slate-100 sm:hidden"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>
          <nav className="order-3 flex w-full items-center justify-center gap-1 text-sm sm:order-none sm:w-auto">
            <Tab to="/" label="Hosts" />
            <Tab to="/settings" label="Settings" />
          </nav>
          <div className="flex items-center gap-2 sm:flex-1 sm:justify-end">
            <ThemeToggle />
            <span aria-hidden className="h-5 w-px bg-slate-200 dark:bg-slate-700" />
            <button
              onClick={onLogout}
              className="rounded-md px-2 py-1 text-sm text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 dark:hover:bg-slate-800 dark:hover:text-slate-100"
            >
              Logout
            </button>
          </div>
        </div>
      </header>
      <main className="flex w-full min-h-0 flex-1 flex-col px-4 pb-4 pt-4 sm:px-6">{children}</main>
    </div>
  );
}

function Tab({ to, label }: { to: string; label: string }) {
  return (
    <NavLink
      to={to}
      end={to === '/'}
      className={({ isActive }) =>
        clsx(
          'rounded-md px-3 py-1.5 transition-colors',
          isActive
            ? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-800 dark:text-slate-100'
            : 'text-slate-500 hover:bg-slate-100 hover:text-slate-900 dark:hover:bg-slate-800 dark:hover:text-slate-100'
        )
      }
    >
      {label}
    </NavLink>
  );
}
