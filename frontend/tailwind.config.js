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
        glass: '0 12px 36px -12px rgb(5 24 54 / 0.18)',
        'glass-sm': '0 6px 20px -8px rgb(5 24 54 / 0.14)',
        glow: '0 0 18px rgb(var(--color-primary-500) / 0.2)',
        'glow-lg': '0 0 44px rgb(var(--color-primary-500) / 0.28)',
        card: '0 8px 28px -20px rgb(5 24 54 / 0.3), inset 0 1px 0 rgb(var(--color-primary-200) / 0.13)',
        'card-hover': '0 12px 32px -18px rgb(var(--color-primary-500) / 0.3)',
        'inner-glow': 'inset 0 1px 0 rgb(var(--color-primary-200) / 0.15)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, rgb(var(--color-primary-500)) 0%, rgb(var(--color-primary-700)) 100%)',
        'gradient-dark': 'linear-gradient(135deg, rgb(var(--color-dark-800)) 0%, rgb(var(--color-dark-950)) 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 78% 8%, rgb(var(--color-primary-500) / 0.12) 0px, transparent 50%), radial-gradient(at 12% 88%, rgb(var(--color-accent-400) / 0.07) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgb(var(--color-primary-500) / 0.25)' },
          '100%': { boxShadow: '0 0 30px rgb(var(--color-primary-500) / 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
