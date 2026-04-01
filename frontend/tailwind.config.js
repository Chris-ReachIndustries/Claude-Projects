/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
      },
      colors: {
        // Surface system: warm dark grays (Linear-inspired)
        surface: {
          ground: '#16161a',
          base: '#1c1d24',
          raised: '#24252e',
          overlay: '#2c2d38',
          spotlight: '#343542',
        },
        // Accent — purple/violet, used sparingly
        accent: {
          400: '#9b6dfa',
          500: '#8b5cf6',
          600: '#7c3aed',
          700: '#6d28d9',
        },
        // Status
        status: {
          active: '#34d399',
          working: '#60a5fa',
          idle: '#fbbf24',
          waiting: '#fb923c',
          error: '#f87171',
        },
        // Legacy compat
        lumi: {
          400: '#9333ff',
          500: '#7c1fff',
          600: '#6b00f0',
        },
        dark: {
          50: '#f0f0f2',
          100: '#e6e6e8',
          200: '#d0d0d4',
          300: '#aeaeb5',
          400: '#86868f',
          500: '#6b6b74',
          600: '#5b5b63',
          700: '#4d4d54',
          800: '#434348',
          850: '#333340',
          900: '#1e1e22',
          925: '#18181b',
          950: '#16161a',
        },
      },
    },
  },
  plugins: [],
};
