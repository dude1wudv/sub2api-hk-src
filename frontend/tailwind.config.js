/** @type {import('tailwindcss').Config} */

const varPalette = (name) => ({
  50: `rgb(var(--color-${name}-50) / <alpha-value>)`,
  100: `rgb(var(--color-${name}-100) / <alpha-value>)`,
  200: `rgb(var(--color-${name}-200) / <alpha-value>)`,
  300: `rgb(var(--color-${name}-300) / <alpha-value>)`,
  400: `rgb(var(--color-${name}-400) / <alpha-value>)`,
  500: `rgb(var(--color-${name}-500) / <alpha-value>)`,
  600: `rgb(var(--color-${name}-600) / <alpha-value>)`,
  700: `rgb(var(--color-${name}-700) / <alpha-value>)`,
  800: `rgb(var(--color-${name}-800) / <alpha-value>)`,
  900: `rgb(var(--color-${name}-900) / <alpha-value>)`,
  950: `rgb(var(--color-${name}-950) / <alpha-value>)`
})

// Foregrounds adapt to the mode independently of solid button and badge fills.
const textPalette = (name) => ({
  400: `rgb(var(--text-${name}) / <alpha-value>)`,
  500: `rgb(var(--text-${name}) / <alpha-value>)`,
  600: `rgb(var(--text-${name}) / <alpha-value>)`
})

export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary: varPalette('primary'),
        accent: varPalette('accent'),
        link: varPalette('link'),
        ok: varPalette('ok'),
        warn: varPalette('warn'),
        err: varPalette('err'),
        amber: varPalette('warn'),
        yellow: varPalette('warn'),
        emerald: varPalette('ok'),
        green: varPalette('ok'),
        lime: varPalette('ok'),
        red: varPalette('err'),
        rose: varPalette('err'),
        orange: varPalette('warn'),
        indigo: varPalette('primary'),
        teal: varPalette('ok'),
        cyan: varPalette('link'),
        sky: varPalette('link'),
        blue: varPalette('link'),
        purple: varPalette('accent'),
        violet: varPalette('accent'),
        fuchsia: varPalette('err'),
        pink: varPalette('err'),
        gray: varPalette('gray'),
        slate: varPalette('gray'),
        zinc: varPalette('gray'),
        neutral: varPalette('gray'),
        dark: varPalette('dark')
      },
      textColor: {
        primary: textPalette('primary'),
        indigo: textPalette('primary'),
        accent: textPalette('accent'),
        purple: textPalette('accent'),
        violet: textPalette('accent'),
        link: textPalette('link'),
        blue: textPalette('link'),
        sky: textPalette('link'),
        cyan: textPalette('link'),
        ok: textPalette('ok'),
        emerald: textPalette('ok'),
        green: textPalette('ok'),
        lime: textPalette('ok'),
        teal: textPalette('ok'),
        warn: textPalette('warn'),
        amber: textPalette('warn'),
        yellow: textPalette('warn'),
        orange: textPalette('warn'),
        err: textPalette('err'),
        red: textPalette('err'),
        rose: textPalette('err'),
        pink: textPalette('err'),
        fuchsia: textPalette('err')
      },
      fontFamily: {
        serif: ['Iowan Old Style', 'Palatino Linotype', 'Songti SC', 'STSong', 'Georgia', 'serif'],
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 1px 0 rgb(var(--line))',
        'glass-sm': '0 1px 0 rgb(var(--line))',
        glow: 'none',
        'glow-lg': 'none',
        card: 'none',
        'card-hover': 'none',
        'inner-glow': 'none'
      },
      backgroundImage: {
        'gradient-radial': 'none',
        'gradient-primary': 'linear-gradient(rgb(var(--color-primary-500)), rgb(var(--color-primary-500)))',
        'gradient-dark': 'linear-gradient(rgb(var(--surface)), rgb(var(--surface)))',
        'gradient-glass': 'none',
        'mesh-gradient': 'none'
      },
      animation: {
        'fade-in': 'fadeIn 0.18s ease-out',
        'slide-up': 'slideUp 0.18s ease-out',
        'slide-down': 'slideDown 0.18s ease-out',
        'slide-in-right': 'slideInRight 0.18s ease-out',
        'scale-in': 'fadeIn 0.18s ease-out',
        'pulse-slow': 'pulse 2s ease-out infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'none'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(4px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        lg: '.5rem',
        xl: '.625rem',
        '2xl': '.75rem',
        '3xl': '.75rem',
        '4xl': '.75rem'
      }
    }
  },
  plugins: []
}
