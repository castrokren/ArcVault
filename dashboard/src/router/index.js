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

const routes = [
  { path: '/', redirect: '/agents' },
  { path: '/login', component: Login },
  { path: '/agents', component: Agents },
  { path: '/jobs', component: Jobs },
  { path: '/history', component: History },
  { path: '/templates', component: Templates },
  { path: '/federation', component: Federation },
  { path: '/federation/health', component: FederationHealth },
  { path: '/users', component: Users },
  { path: '/groups', component: Groups },
  { path: '/alerts', component: Alerts },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

// Route guards for authentication
router.beforeEach((to, from, next) => {
  const auth = useAuth()

  // Allow login page always
  if (to.path === '/login') {
    next()
    return
  }

  // Check if user is authenticated
  if (!auth.isAuthenticated.value) {
    next('/login')
    return
  }

  next()
})

export default router
