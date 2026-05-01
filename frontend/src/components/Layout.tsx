import { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';
import clsx from 'clsx';

interface Props {
  children: ReactNode;
  username: string;
  onLogout: () => void;
}

export default function Layout({ children, username, onLogout }: Props) {
  return (
    <div className="flex min-h-full flex-col">
      <header className="sticky top-0 z-10 border-b border-slate-200 bg-white/80 backdrop-blur dark:border-slate-800 dark:bg-slate-900/80">
        <div className="flex w-full flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <h1 className="text-lg font-semibold tracking-tight">Netglance</h1>
          <nav className="order-3 flex w-full items-center gap-1 text-sm sm:order-none sm:w-auto">
            <Tab to="/" label="Hosts" />
            <Tab to="/settings" label="Settings" />
          </nav>
          <div className="flex items-center gap-3 text-sm">
            <span className="hidden text-slate-500 sm:inline">{username}</span>
            <button onClick={onLogout} className="text-slate-500 hover:text-slate-900 dark:hover:text-slate-100">
              Logout
            </button>
          </div>
        </div>
      </header>
      <main className="w-full flex-1 px-4 pb-8 pt-4 sm:px-6">{children}</main>
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
            : 'text-slate-500 hover:text-slate-900 dark:hover:text-slate-100'
        )
      }
    >
      {label}
    </NavLink>
  );
}
