/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      ringColor: {
        'blue': {
          500: '#3B82F6', // This is Tailwind's default blue-500
        }
      }
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
  variants: {
    extend: {
      ringWidth: ['focus'],
      ringColor: ['focus'],
      ringOpacity: ['focus'],
    },
  },
}