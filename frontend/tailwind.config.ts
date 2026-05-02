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
      },
    },
  },
  plugins: [],
} satisfies Config;
