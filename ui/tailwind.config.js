/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'hector-red': '#ed4267',
        'hector-green': '#10b981',
        'hector-dark': '#1a1a1a',
        'hector-darker': '#0a0f0d',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'ping-slow': 'ping-slow 1s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        'ping-slow': {
          '0%': { transform: 'scale(1)', opacity: '1' },
          '75%, 100%': { transform: 'scale(1.3)', opacity: '0' },
        },
      },
    },
  },
  plugins: [],
}

