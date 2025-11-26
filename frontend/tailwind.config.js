/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // CAU Brand Colors
        cau: {
          blue: '#2945C',     // Cyan 90% + Magenta 50%
          red: '#FF0033',      // Magenta 100% + Yellow 100%
          gray: '#666666',     // Black 55%
          'light-gray': '#E5E5E5', // Black 10%
          silver: '#B3B3B3',   // Pantone 877C
          gold: '#A67C52',     // Pantone 873C
        },
        primary: '#5B7FDB',    // Modern blue based on CAU Blue
        secondary: '#FF4757',  // Modern red based on CAU Red
        accent: '#4A90E2',     // Fintech accent blue
        dark: '#1A1A2E',       // Deep dark
        light: '#F8F9FA',      // Light background
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
      },
      boxShadow: {
        'soft': '0 2px 15px rgba(0, 0, 0, 0.08)',
        'medium': '0 4px 20px rgba(0, 0, 0, 0.12)',
        'hard': '0 10px 40px rgba(0, 0, 0, 0.15)',
      },
    },
  },
  plugins: [],
}
