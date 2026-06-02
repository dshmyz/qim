import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import Home from './views/Home.vue'
import './styles/main.css'

const app = createApp(Home)
app.use(ElementPlus, { locale: zhCn })
app.mount('#app')
