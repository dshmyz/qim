import { createApp } from 'vue'
import App from './App.vue'
import './assets/styles/index.css'
import QMessage from './components/shared/QMessage.vue'
import QMessageBox from './components/shared/QMessageBox.vue'
import '@fortawesome/fontawesome-free/css/all.css'
import { pinia } from './stores'

const app = createApp(App)

app.use(pinia)

app.component('QMessage', QMessage)
app.component('QMessageBox', QMessageBox)

if (navigator.userAgent.includes('Linux')) {
  document.body.classList.add('linux-platform')
}

app.mount('#app')
