import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../features/projects/DashboardView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/projects' },
    { path: '/projects', name: 'projects', component: DashboardView },
    { path: '/:pathMatch(.*)*', redirect: '/projects' },
  ],
})

export default router
