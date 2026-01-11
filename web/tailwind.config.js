/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Plava - primarna (zastava Srbije)
        primary: {
          50: '#e6f0fa',
          100: '#cce1f5',
          200: '#99c3eb',
          300: '#66a5e0',
          400: '#3387d6',
          500: '#0C4076',
          600: '#0a3663',
          700: '#082c50',
          800: '#06223d',
          900: '#04182a',
        },
        // Crvena - akcent (zastava Srbije)
        accent: {
          50: '#fef2f2',
          100: '#fee2e2',
          200: '#fecaca',
          300: '#f8a5a5',
          400: '#e86d6d',
          500: '#C6363C',
          600: '#a62d32',
          700: '#862428',
          800: '#661b1e',
          900: '#461214',
        },
        // Bela i siva za pozadine
        serbia: {
          red: '#C6363C',
          blue: '#0C4076',
          white: '#FFFFFF',
          gold: '#C9A227',
        },
      },
    },
  },
  plugins: [],
}
