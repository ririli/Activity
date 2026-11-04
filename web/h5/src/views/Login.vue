<template>
  <div class="login-page">
    <!-- 藤蔓拱门底纹层（纯 CSS） -->
    <div class="bg-vines">
      <div class="vine-arch arch-center"></div>
      <div class="vine-arch arch-left"></div>
      <div class="vine-arch arch-right"></div>
      <div class="vine-tendril tendril-1"></div>
      <div class="vine-tendril tendril-2"></div>
      <div class="vine-tendril tendril-3"></div>
      <div class="vine-tendril tendril-4"></div>
    </div>

    <!-- 花瓣粒子特效 -->
    <div class="petals-container">
      <span
        v-for="i in 30"
        :key="i"
        class="petal"
        :style="{
          left: randomPct(i, 0) + '%',
          animationDelay: randomPct(i, 1) + 's',
          animationDuration: (3 + (i % 4)) + 's',
          fontSize: (10 + (i % 14)) + 'px',
          opacity: 0.3 + (i % 7) * 0.1
        }"
      >🌸</span>
    </div>

    <!-- 标题区 -->
    <div class="header">
      <div class="title-decoration">✦ ✦ ✦</div>
      <h1 class="title">花开几许 · 集花成羽</h1>
      <p class="subtitle">🌸 许愿池梦幻抽奖活动 🌸</p>
    </div>

    <!-- 表单卡片容器 -->
    <div class="form-wrapper">
      <!-- ========== 登录表单 ========== -->
      <div class="form-card" :class="{ 'form-hidden': currentForm !== 'login' }">
        <!-- 卡片内角装饰 -->
        <div class="rose-deco top-left">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="rose-svg" aria-hidden="true">
            <circle cx="12" cy="8" r="3.5"/>
            <circle cx="7" cy="13" r="3"/>
            <circle cx="17" cy="13" r="3"/>
            <circle cx="10" cy="10" r="2"/>
            <circle cx="14" cy="10" r="2"/>
            <circle cx="12" cy="12" r="1.5"/>
          </svg>
        </div>
        <div class="rose-deco top-right">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="rose-svg" aria-hidden="true">
            <circle cx="12" cy="8" r="3.5"/>
            <circle cx="7" cy="13" r="3"/>
            <circle cx="17" cy="13" r="3"/>
            <circle cx="10" cy="10" r="2"/>
            <circle cx="14" cy="10" r="2"/>
            <circle cx="12" cy="12" r="1.5"/>
          </svg>
        </div>

        <h2 class="form-title">
          <span class="form-title-deco">✦</span> 欢迎回来 <span class="form-title-deco">✦</span>
        </h2>

        <!-- 用户名 -->
        <div class="input-group">
          <span class="input-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="input-svg">
              <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
          </span>
          <input
            v-model="loginForm.username"
            type="text"
            class="glass-input"
            :class="{ 'input-error': loginErrors.username }"
            placeholder="请输入用户名"
            autocomplete="username"
            aria-label="用户名"
            @input="loginErrors.username = ''"
          />
          <span class="input-star" aria-hidden="true">✦</span>
        </div>
        <p v-if="loginErrors.username" class="field-error" role="alert">{{ loginErrors.username }}</p>

        <!-- 密码 -->
        <div class="input-group">
          <span class="input-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="input-svg">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </span>
          <input
            v-model="loginForm.password"
            :type="showLoginPwd ? 'text' : 'password'"
            class="glass-input"
            :class="{ 'input-error': loginErrors.password }"
            placeholder="请输入密码"
            autocomplete="current-password"
            aria-label="密码"
            @keyup.enter="handleLogin"
            @input="loginErrors.password = ''"
          />
          <button
            type="button"
            class="eye-toggle"
            :aria-label="showLoginPwd ? '隐藏密码' : '显示密码'"
            @click="showLoginPwd = !showLoginPwd"
          >
            <!-- 眼睛（可见） -->
            <svg v-if="showLoginPwd" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="eye-svg" aria-hidden="true">
              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
            <!-- 眼睛（隐藏） -->
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="eye-svg" aria-hidden="true">
              <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
              <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
              <path d="m14.12 14.12a3 3 0 1 1-4.24-4.24"/>
              <line x1="1" y1="1" x2="23" y2="23"/>
            </svg>
          </button>
        </div>
        <p v-if="loginErrors.password" class="field-error" role="alert">{{ loginErrors.password }}</p>

        <!-- 提交按钮 -->
        <button
          class="gem-btn"
          :class="{ loading: loginLoading }"
          :disabled="loginLoading"
          :aria-busy="loginLoading"
          @click="handleLogin"
        >
          <span v-if="!loginLoading" class="btn-text">✦ 登 录 ✦</span>
          <span v-else class="btn-loading">
            登录中<span class="loading-dots"><span class="dot">.</span><span class="dot">.</span><span class="dot">.</span></span>
          </span>
          <div class="btn-shine"></div>
        </button>
        <p v-if="loginErrors.general" class="general-error" role="alert">{{ loginErrors.general }}</p>

        <!-- 切换链接 -->
        <p class="switch-text">
          还没有账号？
          <a class="switch-link" @click="switchForm('register')">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="switch-link-icon" aria-hidden="true">
              <path d="M11 20A7 7 0 0 1 9.8 6.9C15.5 4.9 17 3.5 19 2c1 3 0 7.5-4 10"/>
              <path d="M15.5 8H19v3.5"/>
            </svg>
            新用户注册
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="switch-arrow-svg" aria-hidden="true">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </a>
        </p>
      </div>

      <!-- ========== 注册表单 ========== -->
      <div class="form-card" :class="{ 'form-hidden': currentForm !== 'register' }">
        <!-- 卡片内角装饰 -->
        <div class="rose-deco top-left">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="rose-svg" aria-hidden="true">
            <circle cx="12" cy="8" r="3.5"/>
            <circle cx="7" cy="13" r="3"/>
            <circle cx="17" cy="13" r="3"/>
            <circle cx="10" cy="10" r="2"/>
            <circle cx="14" cy="10" r="2"/>
            <circle cx="12" cy="12" r="1.5"/>
          </svg>
        </div>
        <div class="rose-deco top-right">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="rose-svg" aria-hidden="true">
            <circle cx="12" cy="8" r="3.5"/>
            <circle cx="7" cy="13" r="3"/>
            <circle cx="17" cy="13" r="3"/>
            <circle cx="10" cy="10" r="2"/>
            <circle cx="14" cy="10" r="2"/>
            <circle cx="12" cy="12" r="1.5"/>
          </svg>
        </div>

        <h2 class="form-title">
          <span class="form-title-deco">✦</span> 创建账号 <span class="form-title-deco">✦</span>
        </h2>

        <!-- 用户名 -->
        <div class="input-group">
          <span class="input-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="input-svg">
              <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
          </span>
          <input
            v-model="registerForm.username"
            type="text"
            class="glass-input"
            :class="{ 'input-error': registerErrors.username }"
            placeholder="用户名（3-20位字母数字下划线）"
            autocomplete="username"
            aria-label="用户名"
            @input="registerErrors.username = ''"
          />
          <span class="input-star" aria-hidden="true">✦</span>
        </div>
        <p v-if="registerErrors.username" class="field-error" role="alert">{{ registerErrors.username }}</p>

        <!-- 密码 -->
        <div class="input-group">
          <span class="input-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="input-svg">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </span>
          <input
            v-model="registerForm.password"
            :type="showRegPwd ? 'text' : 'password'"
            class="glass-input"
            :class="{ 'input-error': registerErrors.password }"
            placeholder="密码（至少6位）"
            autocomplete="new-password"
            aria-label="密码"
            @input="registerErrors.password = ''"
          />
          <button
            type="button"
            class="eye-toggle"
            :aria-label="showRegPwd ? '隐藏密码' : '显示密码'"
            @click="showRegPwd = !showRegPwd"
          >
            <svg v-if="showRegPwd" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="eye-svg" aria-hidden="true">
              <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
              <circle cx="12" cy="12" r="3"/>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="eye-svg" aria-hidden="true">
              <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/>
              <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/>
              <path d="m14.12 14.12a3 3 0 1 1-4.24-4.24"/>
              <line x1="1" y1="1" x2="23" y2="23"/>
            </svg>
          </button>
        </div>
        <p v-if="registerErrors.password" class="field-error" role="alert">{{ registerErrors.password }}</p>

        <!-- 确认密码 -->
        <div class="input-group">
          <span class="input-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="input-svg">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          </span>
          <input
            v-model="registerForm.confirmPassword"
            :type="showRegPwd ? 'text' : 'password'"
            class="glass-input"
            :class="{ 'input-error': registerErrors.confirmPassword }"
            placeholder="请再次输入密码"
            autocomplete="new-password"
            aria-label="确认密码"
            @keyup.enter="handleRegister"
            @input="registerErrors.confirmPassword = ''"
          />
          <span class="input-star" aria-hidden="true">✧</span>
        </div>
        <p v-if="registerErrors.confirmPassword" class="field-error" role="alert">{{ registerErrors.confirmPassword }}</p>

        <!-- 昵称 -->
        <div class="input-group">
          <span class="input-icon" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="input-svg">
              <circle cx="12" cy="12" r="10"/>
              <path d="M8 14s1.5 2 4 2 4-2 4-2"/>
              <line x1="9" y1="9" x2="9.01" y2="9"/>
              <line x1="15" y1="9" x2="15.01" y2="9"/>
            </svg>
          </span>
          <input
            v-model="registerForm.nickname"
            type="text"
            class="glass-input"
            placeholder="昵称（选填，默认与用户名相同）"
            aria-label="昵称（选填）"
          />
          <span class="input-star" aria-hidden="true">✧</span>
        </div>

        <!-- 提交按钮 -->
        <button
          class="gem-btn"
          :class="{ loading: registerLoading }"
          :disabled="registerLoading"
          :aria-busy="registerLoading"
          @click="handleRegister"
        >
          <span v-if="!registerLoading" class="btn-text">✦ 注 册 ✦</span>
          <span v-else class="btn-loading">
            注册中<span class="loading-dots"><span class="dot">.</span><span class="dot">.</span><span class="dot">.</span></span>
          </span>
          <div class="btn-shine"></div>
        </button>
        <p v-if="registerErrors.general" class="general-error" role="alert">{{ registerErrors.general }}</p>

        <!-- 切换链接 -->
        <p class="switch-text">
          已有账号？
          <a class="switch-link" @click="switchForm('login')">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="switch-link-icon" aria-hidden="true">
              <path d="M11 20A7 7 0 0 1 9.8 6.9C15.5 4.9 17 3.5 19 2c1 3 0 7.5-4 10"/>
              <path d="M15.5 8H19v3.5"/>
            </svg>
            返回登录
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="switch-arrow-svg" aria-hidden="true">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </a>
        </p>
      </div>
    </div>

    <!-- 成功弹窗 -->
    <Teleport to="body">
      <div
        class="success-overlay"
        v-if="showSuccess"
        role="dialog"
        aria-modal="true"
        :aria-label="successType === 'register' ? '注册成功' : '登录成功'"
        @click.self="goHome"
      >
        <div class="success-card">
          <div class="success-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="success-icon-svg" aria-hidden="true">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <polyline points="22 4 12 14.01 9 11.01"/>
            </svg>
          </div>
          <h2 class="success-title" v-if="successType === 'register'">注册成功！</h2>
          <h2 class="success-title" v-else>登录成功！</h2>
          <p class="success-desc">
            {{ successType === 'register' ? '恭喜获得入场资格' : '欢迎回来' }}
          </p>
          <p class="success-sub">正在跳转到许愿池...</p>
          <div class="success-sparkles">✦ ✦ ✦</div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const currentForm = ref<'login' | 'register'>('login')
const loginLoading = ref(false)
const registerLoading = ref(false)
const showLoginPwd = ref(false)
const showRegPwd = ref(false)
const showSuccess = ref(false)
const successType = ref<'login' | 'register'>('login')

const loginForm = reactive({ username: '', password: '' })
const registerForm = reactive({ username: '', password: '', confirmPassword: '', nickname: '' })

// ---- 内联错误状态 ----
const loginErrors = reactive({ username: '', password: '', general: '' })
const registerErrors = reactive({ username: '', password: '', confirmPassword: '', nickname: '', general: '' })

function clearLoginErrors() {
  loginErrors.username = ''
  loginErrors.password = ''
  loginErrors.general = ''
}

function clearRegisterErrors() {
  registerErrors.username = ''
  registerErrors.password = ''
  registerErrors.confirmPassword = ''
  registerErrors.nickname = ''
  registerErrors.general = ''
}

// 花瓣动画随机种子
const seeds = Array.from({ length: 30 }, (_, i) => ({
  x: ((i * 137 + 53) % 100),
  d: ((i * 97 + 19) % 5)
}))
function randomPct(index: number, type: number): number {
  return seeds[index] ? (type === 0 ? seeds[index].x : seeds[index].d) : 0
}

function switchForm(form: 'login' | 'register') {
  currentForm.value = form
  clearLoginErrors()
  clearRegisterErrors()
}

async function handleLogin() {
  clearLoginErrors()
  let valid = true

  if (!loginForm.username.trim()) {
    loginErrors.username = '请输入用户名'
    valid = false
  }
  if (!loginForm.password) {
    loginErrors.password = '请输入密码'
    valid = false
  }
  if (!valid) return

  loginLoading.value = true
  try {
    await authStore.login({ account: loginForm.username.trim(), password: loginForm.password })
    successType.value = 'login'; showSuccess.value = true
    setTimeout(() => goHome(), 1200)
  } catch (err: any) {
    loginErrors.general = err.message || '登录失败，请稍后重试'
  } finally { loginLoading.value = false }
}

async function handleRegister() {
  clearRegisterErrors()
  let valid = true

  if (!registerForm.username.trim()) {
    registerErrors.username = '请输入用户名'
    valid = false
  } else if (!/^[a-zA-Z0-9_]{3,20}$/.test(registerForm.username.trim())) {
    registerErrors.username = '用户名格式：3-20位字母、数字、下划线'
    valid = false
  }
  if (registerForm.password.length < 6) {
    registerErrors.password = '密码长度不能少于6位'
    valid = false
  }
  if (registerForm.password !== registerForm.confirmPassword) {
    registerErrors.confirmPassword = '两次输入的密码不一致'
    valid = false
  }
  if (!valid) return

  registerLoading.value = true
  try {
    await authStore.register({
      account: registerForm.username.trim(),
      password: registerForm.password,
      nickname: registerForm.nickname.trim() || undefined
    })
    successType.value = 'register'; showSuccess.value = true
    setTimeout(() => goHome(), 1200)
  } catch (err: any) {
    registerErrors.general = err.message || '注册失败，请稍后重试'
  } finally { registerLoading.value = false }
}

function goHome() {
  showSuccess.value = false
  router.replace((route.query.redirect as string) || '/')
}
</script>

<style scoped>
/* ============================================================
   页面容器
   ============================================================ */
.login-page {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 20px 0;
  background: linear-gradient(180deg,
    #e8f4f8 0%, #fef0f3 20%, #fce4ec 40%,
    #f8bbd0 65%, #f48fb1 85%, #e1bee7 100%
  );
}

/* ============================================================
   藤蔓拱门底纹
   ============================================================ */
.bg-vines {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  overflow: hidden;
}

/* 中央大拱门 */
.vine-arch {
  position: absolute;
  border-radius: 50%;
  border: 1.5px solid rgba(180,130,140,0.2);
  box-shadow:
    inset 0 0 40px rgba(210,160,170,0.15),
    0 0 60px rgba(210,160,170,0.1);
}

.arch-center {
  width: 140%;
  height: 110%;
  top: -65%;
  left: -20%;
  border-bottom: none;
  border-color: rgba(180,140,150,0.25);
  box-shadow:
    inset 0 0 60px rgba(210,160,170,0.2),
    0 0 80px rgba(210,160,170,0.12);
}

.arch-left {
  width: 80%;
  height: 70%;
  top: -45%;
  left: -30%;
  border-bottom: none;
  border-right: 1px solid rgba(180,140,150,0.15);
}

.arch-right {
  width: 80%;
  height: 70%;
  top: -45%;
  right: -30%;
  border-bottom: none;
  border-left: 1px solid rgba(180,140,150,0.15);
}

/* 藤蔓卷须 */
.vine-tendril {
  position: absolute;
  width: 60px;
  height: 120px;
  border: 1px solid rgba(170,140,150,0.15);
  border-radius: 50%;
  border-bottom: none;
  border-right: none;
}

.tendril-1 { top: 8%;  left: 5%;  transform: rotate(-30deg); }
.tendril-2 { top: 12%; right: 5%;  transform: rotate(30deg) scaleX(-1); }
.tendril-3 { top: 22%; left: 12%; transform: rotate(-50deg) scale(0.7); }
.tendril-4 { top: 20%; right: 10%; transform: rotate(50deg) scaleX(-1) scale(0.8); }

/* ============================================================
   花瓣粒子
   ============================================================ */
.petals-container {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 1;
  overflow: hidden;
}

.petal {
  position: absolute;
  top: -30px;
  animation: petal-fall linear infinite;
}

@keyframes petal-fall {
  0%   { transform: translateY(-30px) rotate(0deg) scale(1); opacity: 0; }
  10%  { opacity: 0.8; }
  90%  { opacity: 0.5; }
  100% { transform: translateY(105vh) rotate(720deg) scale(0.3); opacity: 0; }
}

/* ============================================================
   标题区
   ============================================================ */
.header {
  position: relative;
  z-index: 10;
  text-align: center;
  margin-bottom: 20px;
}

.title-decoration {
  font-size: 16px;
  color: #d4af37;
  letter-spacing: 12px;
  margin-bottom: 8px;
  animation: sparkle 2s ease-in-out infinite;
}

@keyframes sparkle {
  0%, 100% { opacity: 0.4; transform: scale(1); }
  50%      { opacity: 1;   transform: scale(1.1); }
}

.title {
  font-family: 'Ma Shan Zheng', 'ZCOOL KuaiLe', 'STKaiti', 'KaiTi', cursive;
  font-size: 30px;
  font-weight: 400;
  color: #5d4037;
  text-shadow:
    0 0 16px rgba(255,183,197,0.9),
    0 0 32px rgba(255,183,197,0.5),
    2px 3px 6px rgba(255,255,255,0.7);
  letter-spacing: 4px;
  margin: 0 0 6px;
  line-height: 1.5;
}

.subtitle {
  font-size: 13px;
  color: #8d6e63;
  margin: 0;
  letter-spacing: 2px;
}

/* ============================================================
   表单卡片容器
   ============================================================ */
.form-wrapper {
  position: relative;
  z-index: 10;
  width: 88%;
  max-width: 360px;
  min-height: 340px;
}

/* ============================================================
   磨砂玻璃表单卡片
   ============================================================ */
.form-card {
  position: relative;
  background: rgba(255, 255, 255, 0.22);
  backdrop-filter: blur(18px);
  -webkit-backdrop-filter: blur(18px);
  border: 1px solid rgba(212, 175, 55, 0.3);
  border-radius: 26px;
  padding: 34px 24px 26px;
  box-shadow:
    0 10px 40px rgba(200, 130, 150, 0.25),
    0 0 0 1px rgba(255,255,255,0.4) inset,
    0 1px 2px rgba(255,255,255,0.6) inset;
  transition: all 0.45s cubic-bezier(0.4, 0, 0.2, 1);
  opacity: 1;
  transform: translateY(0) scale(1);
  animation: card-glow 4s ease-in-out infinite;
}

@keyframes card-glow {
  0%, 100% { border-color: rgba(212, 175, 55, 0.3); }
  50%      { border-color: rgba(212, 175, 55, 0.48); }
}

.form-card.form-hidden {
  position: absolute;
  top: 0;
  left: 50%;
  translate: -50% 0;
  width: 100%;
  opacity: 0;
  transform: translateY(-24px) scale(0.95);
  pointer-events: none;
}

/* ---- 卡片内角装饰 ---- */
.rose-deco {
  position: absolute;
  opacity: 0.55;
}

.rose-deco.top-left  { top: -4px;  left: 18px; }
.rose-deco.top-right { top: -4px; right: 18px; }

.rose-svg {
  width: 22px;
  height: 22px;
  color: rgba(200, 130, 155, 0.55);
  filter: drop-shadow(0 1px 2px rgba(180,120,140,0.2));
}

.form-title {
  text-align: center;
  font-size: 17px;
  color: #5d4037;
  margin: 0 0 26px;
  font-weight: 600;
  letter-spacing: 2px;
}

.form-title-deco {
  color: #d4af37;
  font-size: 14px;
}

/* ============================================================
   输入框组
   ============================================================ */
.input-group {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 16px;
}

.input-icon {
  position: absolute;
  left: 14px;
  z-index: 2;
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: center;
}

.input-svg {
  width: 18px;
  height: 18px;
  color: #bcaaa4;
}

.input-star {
  position: absolute;
  right: 14px;
  font-size: 12px;
  color: #d4af37;
  z-index: 2;
}

/* 密码可见性切换按钮 */
.eye-toggle {
  position: absolute;
  right: 6px;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  color: #d4af37;
  border-radius: 8px;
  transition: color 0.2s, background 0.2s;
  -webkit-tap-highlight-color: transparent;
}

.eye-toggle:active {
  background: rgba(212, 175, 55, 0.12);
}

.eye-svg {
  width: 20px;
  height: 20px;
}

.glass-input {
  width: 100%;
  height: 48px;
  padding: 0 38px 0 42px;
  font-size: 15px;
  color: #4a3b3b;
  background: rgba(255, 255, 255, 0.32);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  border: 1px solid rgba(212, 175, 55, 0.35);
  border-radius: 14px;
  outline: none;
  transition: all 0.35s ease;
  box-shadow: inset 0 2px 4px rgba(0,0,0,0.03);
}

.glass-input::placeholder {
  color: rgba(139, 115, 115, 0.45);
  font-size: 13px;
}

.glass-input:focus {
  border-color: rgba(212, 175, 55, 0.75);
  box-shadow:
    0 0 0 4px rgba(255, 183, 197, 0.22),
    0 0 14px rgba(212, 175, 55, 0.18),
    inset 0 2px 4px rgba(0,0,0,0.03);
  background: rgba(255, 255, 255, 0.48);
}

/* ---- 输入框错误态 ---- */
.glass-input.input-error {
  border-color: rgba(231, 76, 60, 0.6);
  box-shadow: 0 0 0 3px rgba(231, 76, 60, 0.1);
}

/* ============================================================
   内联错误提示
   ============================================================ */
.field-error {
  margin: -10px 0 14px 4px;
  font-size: 12px;
  color: #e74c3c;
  line-height: 1.4;
  animation: shake-in 0.3s ease;
}

.general-error {
  margin: 10px 0 0 0;
  font-size: 13px;
  color: #e74c3c;
  text-align: center;
  animation: shake-in 0.3s ease;
}

@keyframes shake-in {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-4px); }
  40% { transform: translateX(4px); }
  60% { transform: translateX(-3px); }
  80% { transform: translateX(2px); }
}

/* ============================================================
   宝石按钮
   ============================================================ */
.gem-btn {
  position: relative;
  width: 100%;
  height: 50px;
  margin-top: 30px;
  border: none;
  border-radius: 25px;
  background: linear-gradient(135deg, #ffb7c5 0%, #f48fb1 30%, #d4af37 100%);
  color: #fff;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 6px;
  cursor: pointer;
  overflow: hidden;
  box-shadow:
    0 8px 24px rgba(220, 130, 155, 0.45),
    0 2px 6px rgba(0,0,0,0.1);
  transition: all 0.3s ease;
  -webkit-tap-highlight-color: transparent;
}

.gem-btn:active:not(.loading) {
  transform: scale(0.96);
  box-shadow:
    0 3px 12px rgba(220, 130, 155, 0.35),
    0 1px 2px rgba(0,0,0,0.1);
}

.gem-btn.loading {
  opacity: 0.7;
  cursor: not-allowed;
}

.btn-shine {
  position: absolute;
  top: 0;
  left: -100%;
  width: 60%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.3), transparent);
  transform: skewX(-25deg);
  animation: shine 3s ease-in-out infinite;
}

@keyframes shine {
  0%   { left: -100%; }
  50%  { left: 140%; }
  100% { left: 140%; }
}

.btn-text, .btn-loading {
  position: relative;
  z-index: 1;
  text-shadow: 0 1px 2px rgba(0,0,0,0.15);
}

/* 加载中省略号动画 */
.loading-dots {
  display: inline;
}

.loading-dots .dot {
  animation: dot-blink 1.4s ease-in-out infinite;
}

.loading-dots .dot:nth-child(1) { animation-delay: 0s; }
.loading-dots .dot:nth-child(2) { animation-delay: 0.2s; }
.loading-dots .dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes dot-blink {
  0%, 80%, 100% { opacity: 0; }
  40% { opacity: 1; }
}

/* ============================================================
   切换链接
   ============================================================ */
.switch-text {
  text-align: center;
  margin-top: 24px;
  margin-bottom: 0;
  font-size: 14px;
  color: #8d6e63;
}

.switch-link {
  color: #c9a84c;
  font-weight: 600;
  cursor: pointer;
  text-decoration: none;
  transition: color 0.2s;
  -webkit-tap-highlight-color: transparent;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.switch-link:active {
  color: #f48fb1;
}

.switch-link-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.switch-arrow-svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  animation: arrow-bounce 1.5s ease-in-out infinite;
}

@keyframes arrow-bounce {
  0%, 100% { transform: translateX(0); }
  50%      { transform: translateX(4px); }
}

/* ============================================================
   成功弹窗
   ============================================================ */
.success-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  animation: fade-in 0.3s ease;
}

@keyframes fade-in {
  from { opacity: 0; }
  to   { opacity: 1; }
}

.success-card {
  background: rgba(255,255,255,0.93);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  border: 2px solid rgba(212,175,55,0.5);
  border-radius: 24px;
  padding: 36px 32px;
  text-align: center;
  box-shadow:
    0 24px 64px rgba(200,130,150,0.35),
    0 0 0 1px rgba(255,255,255,0.8);
  animation: pop-in 0.45s cubic-bezier(0.175, 0.885, 0.32, 1.275);
  max-width: 300px;
}

@keyframes pop-in {
  from { transform: scale(0.7); opacity: 0; }
  to   { transform: scale(1);   opacity: 1; }
}

.success-icon {
  margin-bottom: 12px;
  animation: bounce 0.6s ease infinite alternate;
  color: #4caf50;
}

.success-icon-svg {
  width: 52px;
  height: 52px;
}

@keyframes bounce {
  from { transform: translateY(0); }
  to   { transform: translateY(-8px); }
}

.success-title {
  font-size: 20px;
  color: #5d4037;
  margin: 0 0 8px;
  letter-spacing: 2px;
}

.success-desc {
  font-size: 14px;
  color: #8d6e63;
  margin: 0 0 4px;
}

.success-sub {
  font-size: 12px;
  color: #bcaaa4;
  margin: 0 0 12px;
}

.success-sparkles {
  font-size: 18px;
  color: #d4af37;
  letter-spacing: 8px;
}

/* ============================================================
   焦点可见性（无障碍）
   ============================================================ */
.glass-input:focus-visible {
  outline: 2px solid rgba(212, 175, 55, 0.8);
  outline-offset: 1px;
}

.gem-btn:focus-visible {
  outline: 2px solid rgba(212, 175, 55, 0.9);
  outline-offset: 2px;
}

.switch-link:focus-visible {
  outline: 2px solid rgba(201, 168, 76, 0.7);
  outline-offset: 2px;
  border-radius: 4px;
}

.eye-toggle:focus-visible {
  outline: 2px solid rgba(212, 175, 55, 0.8);
  outline-offset: 2px;
}

/* ============================================================
   桌面端悬停态
   ============================================================ */
@media (hover: hover) {
  .gem-btn:hover:not(.loading) {
    transform: translateY(-1px);
    box-shadow:
      0 10px 28px rgba(220, 130, 155, 0.5),
      0 2px 6px rgba(0,0,0,0.1);
  }

  .switch-link:hover {
    color: #e0b84c;
  }
}

/* ============================================================
   无障碍：减少动画
   ============================================================ */
@media (prefers-reduced-motion: reduce) {
  .petal {
    animation: none !important;
    display: none;
  }

  .btn-shine {
    animation: none !important;
    opacity: 0;
  }

  .switch-arrow-svg {
    animation: none !important;
  }

  .title-decoration {
    animation: none !important;
    opacity: 0.7;
  }

  .success-icon {
    animation: none !important;
  }

  .success-card {
    animation: fade-in 0.1s ease !important;
  }

  .form-card {
    animation: none !important;
    transition: opacity 0.1s ease !important;
  }

  .form-card.form-hidden {
    transform: none !important;
  }

  .gem-btn:active:not(.loading) {
    transform: scale(0.98) !important;
    transition: transform 0.05s ease !important;
  }

  .field-error,
  .general-error {
    animation: none !important;
  }
}
</style>
