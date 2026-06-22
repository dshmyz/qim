import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import './styles/main.css'
import App from './App.vue'
import router from './router'
import { permissionDirective } from './directives/permission'
import { roleDirective } from './directives/role'
import { setupPermissionGuard } from './router/guards'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn })

app.directive('permission', permissionDirective)
app.directive('role', roleDirective)
router.beforeEach(setupPermissionGuard())

app.mount('#app')
