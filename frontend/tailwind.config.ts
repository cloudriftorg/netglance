import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // OPNsense-inspired orange. Matches the firmware's primary accent
        // so the bundled web UI feels at home next to the host firewall.
        brand: {
          50: '#fff5ec',
          500: '#f5681c',
          600: '#d94f00',
          700: '#b14000',
          // 800 / 900 added after the opaque-theme sweep mapped
          // `dark:bg-brand-700/20` → `dark:bg-brand-900` — without
          // these the dark override resolved to nothing and the VLAN
          // badge stayed `bg-brand-50` with `text-brand-50` on top
          // (same colour, invisible label).
          800: '#7a2c00',
          900: '#4a1a00',
        },
        // Override Tailwind's default slate dark shades to match OPNsense's
        // near-black firewall theme (slate's defaults have a navy-blue tint
        // that clashes when the netglance UI sits in an OPNsense iframe or
        // is opened straight from the firewall menu). Keep the light-mode
        // shades as Tailwind ships them.
        slate: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          // Dark shades: pure neutral grays, no blue/red bias. Panels at
          // slate-900 (#1a1a1a) sit just above the body's pure black so
          // they read as cards on a dark backdrop.
          600: '#525252',
          700: '#333333',
          800: '#222222',
          900: '#1a1a1a',
          // The page <body> uses `dark:bg-slate-950`. Sits one notch below
          // the slate-900 panels (#1a1a1a) so cards lift cleanly without
          // the harshness of pure black.
          950: '#0f0f0f',
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
