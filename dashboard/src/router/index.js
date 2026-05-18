import { createRouter, createWebHashHistory } from 'vue-router'
import Agents from '../views/Agents.vue'
import Jobs from '../views/Jobs.vue'
import History from '../views/History.vue'
import Templates from '../views/Templates.vue'

const routes = [
  { path: '/', redirect: '/agents' },
  { path: '/agents', component: Agents },
  { path: '/jobs', component: Jobs },
  { path: '/history', component: History },
  { path: '/templates', component: Templates },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
