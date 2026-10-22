/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{vue,js,ts,jsx,tsx}",
    ],
    darkMode: 'class',
    theme: {
        container: {
            center: true,
            padding: {
                DEFAULT: '16px',
                md: '24px',
                xl: '32px',
            },
            screens: {
                sm: '1200px',
                md: '1200px',
                lg: '1200px',
                xl: '1200px',
                '2xl': '1200px',
            },
        },
        extend: {},
    },
    plugins: [],
}
