import { createApp } from 'vue'
import App from './App.vue'
import Docs from './Docs.vue'
import './style.css'

createApp(window.location.pathname.replace(/\/+$/, '') === '/docs' ? Docs : App).mount('#app')
