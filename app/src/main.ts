import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import 'primeicons/primeicons.css'

import App from './App.vue'
import router from './router'
import { getThemePreset } from './themes/presets'
import './styles.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(PrimeVue, {
  ripple: true,
  inputVariant: 'filled',
  theme: {
    preset: getThemePreset('aura'),
    options: {
      darkModeSelector: '.app-dark',
    },
  },
})

app.mount('#app')
