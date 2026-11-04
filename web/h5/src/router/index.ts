import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from '@/api/request'

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

// 路由守卫：未登录时重定向到登录页
router.beforeEach((to, _from, next) => {
  if (to.meta.requiresAuth) {
    const token = getToken()
    if (!token) {
      next({ name: 'Login', query: { redirect: to.fullPath } })
      return
    }
  }
  // 已登录用户访问登录页 → 重定向到首页
  if (to.name === 'Login') {
    const token = getToken()
    if (token) {
      next({ name: 'Home' })
      return
    }
  }
  next()
})

export default router
