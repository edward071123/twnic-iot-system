import { createRouter, createWebHistory } from 'vue-router'
import WardFloorView from '../views/WardFloorView.vue'
import EngineerLoginView from '../views/EngineerLoginView.vue'
import { isEngineerAuthed } from '../auth/engineerAuth'

const routes = [
  {
    path: '/',
    name: 'ward-public',
    component: WardFloorView,
    props: { showThermal: false, engineerMode: false },
  },
  {
    path: '/engineer/login',
    name: 'ward-engineer-login',
    component: EngineerLoginView,
  },
  {
    path: '/engineer',
    name: 'ward-engineer',
    component: WardFloorView,
    props: { showThermal: true, engineerMode: true },
    meta: { requiresEngineerAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to) => {
  if (to.meta.requiresEngineerAuth && !isEngineerAuthed()) {
    return {
      name: 'ward-engineer-login',
      query: { redirect: to.fullPath },
    }
  }
  if (to.name === 'ward-engineer-login' && isEngineerAuthed()) {
    const redirect = typeof to.query.redirect === 'string' ? to.query.redirect : '/engineer'
    return { path: redirect }
  }
  return true
})

export default router
