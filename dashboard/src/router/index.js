import { createRouter, createWebHashHistory } from 'vue-router'
import { useAuth } from '../composables/useAuth.js'
import Login from '../views/Login.vue'
import Agents from '../views/Agents.vue'
import Jobs from '../views/Jobs.vue'
import History from '../views/History.vue'
import Templates from '../views/Templates.vue'
import Federation from '../views/Federation.vue'
import FederationHealth from '../views/FederationHealth.vue'
import Users from '../views/Users.vue'
import Groups from '../views/Groups.vue'
import Alerts from '../views/Alerts.vue'
import Credentials from '../views/admin/Credentials.vue'

const routes = [
  { path: '/', redirect: '/agents' },
  { path: '/login', component: Login },
  { path: '/agents', component: Agents },
  { path: '/jobs', component: Jobs },
  { path: '/history', component: History },
  { path: '/templates', component: Templates },
  { path: '/federation', component: Federation, meta: { requiresRole: 'admin' } },
  { path: '/federation/health', component: FederationHealth, meta: { requiresRole: 'admin' } },
  { path: '/users', component: Users, meta: { requiresRole: 'admin' } },
  { path: '/groups', component: Groups, meta: { requiresRole: 'admin' } },
  { path: '/alerts', component: Alerts, meta: { requiresRole: 'admin' } },
  { path: '/admin/credentials', component: Credentials, meta: { requiresRole: 'admin' } },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// Route guards for authentication
router.beforeEach((to, from, next) => {
  const auth = useAuth()

  // Login page: only for unauthenticated users — otherwise the nav header
  // (gated on isAuthenticated in App.vue) renders on top of the login form.
  if (to.path === '/login') {
    if (auth.isAuthenticated.value) {
      next('/agents')
    } else {
      next()
    }
    return
  }

  // Check if user is authenticated
  if (!auth.isAuthenticated.value) {
    next('/login')
    return
  }

  // Role-based route guard
  if (to.meta && to.meta.requiresRole && !auth.hasRole(to.meta.requiresRole)) {
    next('/agents')
    return
  }

  next()
})

export default router
