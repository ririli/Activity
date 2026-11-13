import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { showToast } from 'vant'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'Home',
      component: () => import('@/views/Home.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue')
    },
    {
      path: '/leaderboard',
      name: 'Leaderboard',
      component: () => import('@/views/Leaderboard.vue'),
      meta: { requiresAuth: true }
    }
  ]
})

// Access lives in memory, so a page reload must restore it from the HttpOnly
// Refresh cookie before deciding whether a route is authorized.
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  try {
    if (to.meta.requiresAuth) {
      const ok = await auth.initFromStorage()
      if (!ok) {
        return { name: 'Login', query: { redirect: to.fullPath } }
      }
    }
    if (to.name === 'Login') {
      const ok = await auth.initFromStorage()
      if (ok) return { name: 'Home' }
    }
    return true
  } catch (error) {
    showToast(error instanceof Error ? error.message : '服务暂时不可用，请稍后重试')
    return false
  }
})

export default router
