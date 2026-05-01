import { useTheme, type Theme } from '../lib/theme';

const ORDER: Theme[] = ['system', 'light', 'dark'];
const LABEL: Record<Theme, string> = {
  system: 'System',
  light: 'Light',
  dark: 'Dark',
};

export default function ThemeToggle() {
  const { theme, setTheme } = useTheme();

  function cycle() {
    const i = ORDER.indexOf(theme);
    setTheme(ORDER[(i + 1) % ORDER.length]);
  }

  return (
    <button
      onClick={cycle}
      title={`Theme: ${LABEL[theme]} (click to change)`}
      aria-label={`Theme: ${LABEL[theme]}`}
      className="flex items-center gap-1.5 rounded-md px-2 py-1 text-sm text-slate-500 hover:bg-slate-100 hover:text-slate-900 dark:hover:bg-slate-800 dark:hover:text-slate-100"
    >
      <Icon theme={theme} />
      <span className="hidden sm:inline">{LABEL[theme]}</span>
    </button>
  );
}

function Icon({ theme }: { theme: Theme }) {
  if (theme === 'light') {
    return (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
        <circle cx="12" cy="12" r="4" />
        <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
      </svg>
    );
  }
  if (theme === 'dark') {
    return (
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
      </svg>
    );
  }
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <rect x="3" y="4" width="18" height="14" rx="2" />
      <path d="M8 21h8M12 18v3" />
    </svg>
  );
}
