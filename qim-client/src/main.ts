import { createApp } from 'vue'
import App from './App.vue'
import './assets/styles/index.css'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import '@fortawesome/fontawesome-free/css/all.css'

createApp(App).use(ElementPlus).mount('#app')
