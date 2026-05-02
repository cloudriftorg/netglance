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
          600: '#525252',
          700: '#3a3a3a',
          800: '#262626',
          900: '#1d1d1d',
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
