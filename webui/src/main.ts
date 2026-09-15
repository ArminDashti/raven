import { createApp } from 'vue'
import { registerSW } from 'virtual:pwa-register'
import App from './App.vue'
import router from './router'
import { initTheme } from '@/lib/theme'
import '@fontsource/vazirmatn/400.css'
import '@fontsource/vazirmatn/500.css'
import '@fontsource/vazirmatn/700.css'
import '@fontsource/share-tech-mono/400.css'
import './assets/main.css'

initTheme()

registerSW({
  immediate: true,
})

createApp(App).use(router).mount('#app')
