<!--
  Login.vue — Royal Purple restyle + Data Convergence animation (handoff)

  WHAT CHANGED vs the original:
   • The flat <style scoped> block is removed; all visual tokens come from
     theme-royal-purple.css via the existing CSS-variable cascade.
   • A new <style scoped> block references login-animation.css for the
     background scene (aurora, beams, watermark, starfield, halo).
   • The orbit-scene SVG from the original is KEPT (orbit rings, comet arcs)
     and layered behind the new beam scene — both play simultaneously.
   • The brand icon gains the double-spinning energy halo via
     .login-brand-icon-wrap / .login-halo-outer / .login-halo-inner.

  WHAT DID NOT CHANGE:
   • Every auth handler, form binding, error state, 2FA flow, and router
     call is identical to the original. Pure visual layer only.

  IMPORTS:
   In main.js (after style.css and theme-royal-purple.css):
     import './login-animation.css'
-->
<template>
  <div class="login-page">

    <!-- ── Background animation scene ───────────────────── -->
    <div class="orbit-scene" aria-hidden="true">

      <!-- Aurora nebula blobs -->
      <div class="login-aurora login-aurora-1"></div>
      <div class="login-aurora login-aurora-2"></div>
      <div class="login-aurora login-aurora-3"></div>
      <div class="login-aurora login-aurora-4"></div>

      <!-- Ghost shield watermark -->
      <div class="login-watermark">
        <svg width="560" height="560" viewBox="0 0 24 24" fill="rgba(167,139,250,.032)">
          <path d="M12 3L20 7.5V12C20 16.42 16.47 20.5 12 21.5C7.53 20.5 4 16.42 4 12V7.5L12 3Z"/>
        </svg>
      </div>

      <!-- Starfield -->
      <div class="login-stars">
        <span class="login-star"    :style="{left:'7%',  top:'11%', animationDelay:'0s',    animation:'avTwinkle 3.2s ease-in-out infinite'}"></span>
        <span class="login-star lg" :style="{left:'17%', top:'64%', animationDelay:'.6s',   animation:'avTwinkle 4.1s ease-in-out .6s infinite'}"></span>
        <span class="login-star"    :style="{left:'6%',  top:'82%', animationDelay:'1.1s',  animation:'avTwinkle 3.6s ease-in-out 1.1s infinite'}"></span>
        <span class="login-star"    :style="{left:'75%', top:'9%',  animationDelay:'.3s',   animation:'avTwinkle 2.9s ease-in-out .3s infinite'}"></span>
        <span class="login-star lg" :style="{left:'86%', top:'44%', animationDelay:'.9s',   animation:'avTwinkle 4.6s ease-in-out .9s infinite'}"></span>
        <span class="login-star"    :style="{left:'82%', top:'74%', animationDelay:'1.6s',  animation:'avTwinkle 3.4s ease-in-out 1.6s infinite'}"></span>
        <span class="login-star"    :style="{left:'61%', top:'89%', animationDelay:'.4s',   animation:'avTwinkle 3.9s ease-in-out .4s infinite'}"></span>
        <span class="login-star"    :style="{left:'32%', top:'6%',  animationDelay:'1.3s',  animation:'avTwinkle 3.1s ease-in-out 1.3s infinite'}"></span>
        <span class="login-star"    :style="{left:'45%', top:'93%', animationDelay:'.8s',   animation:'avTwinkle 4.3s ease-in-out .8s infinite'}"></span>
        <span class="login-star"    :style="{left:'67%', top:'27%', animationDelay:'1.9s',  animation:'avTwinkle 2.7s ease-in-out 1.9s infinite'}"></span>
        <span class="login-star"    :style="{left:'24%', top:'44%', animationDelay:'2.2s',  animation:'avTwinkle 3.8s ease-in-out 2.2s infinite'}"></span>
        <span class="login-star"    :style="{left:'54%', top:'17%', animationDelay:'.7s',   animation:'avTwinkle 3.3s ease-in-out .7s infinite'}"></span>
        <span class="login-star"    :style="{left:'90%', top:'21%', animationDelay:'1.5s',  animation:'avTwinkle 4.0s ease-in-out 1.5s infinite'}"></span>
        <span class="login-star lg" :style="{left:'11%', top:'37%', animationDelay:'.2s',   animation:'avTwinkle 2.6s ease-in-out .2s infinite'}"></span>
        <span class="login-star"    :style="{left:'71%', top:'54%', animationDelay:'2.0s',  animation:'avTwinkle 3.7s ease-in-out 2.0s infinite'}"></span>
        <span class="login-star"    :style="{left:'39%', top:'71%', animationDelay:'1.1s',  animation:'avTwinkle 4.2s ease-in-out 1.1s infinite'}"></span>
        <span class="login-star"    :style="{left:'93%', top:'62%', animationDelay:'.5s',   animation:'avTwinkle 3.5s ease-in-out .5s infinite'}"></span>
        <span class="login-star"    :style="{left:'3%',  top:'54%', animationDelay:'1.7s',  animation:'avTwinkle 4.4s ease-in-out 1.7s infinite'}"></span>
        <span class="login-star"    :style="{left:'49%', top:'2%',  animationDelay:'2.4s',  animation:'avTwinkle 3.0s ease-in-out 2.4s infinite'}"></span>
        <span class="login-star lg" :style="{left:'79%', top:'91%', animationDelay:'1.0s',  animation:'avTwinkle 2.8s ease-in-out 1.0s infinite'}"></span>
      </div>

      <!-- Original orbit rings (kept from existing Login.vue) -->
      <div class="orbit-rings">
        <svg viewBox="0 0 800 800" class="orbit-svg">
          <circle cx="400" cy="400" r="150" fill="none" stroke="rgba(167,139,250,.14)" stroke-width="1"/>
          <circle cx="400" cy="400" r="230" fill="none" stroke="rgba(167,139,250,.1)"  stroke-width="1"/>
          <circle cx="400" cy="400" r="310" fill="none" stroke="rgba(167,139,250,.07)" stroke-width="1"/>
          <circle cx="400" cy="400" r="390" fill="none" stroke="rgba(167,139,250,.1)"  stroke-width="1"
            stroke-dasharray="2 12" class="orbit-dashed"/>
          <g class="orbit-arm-cw">
            <path d="M 710 400 A 310 310 0 0 1 540 670" fill="none" stroke="#a78bfa" stroke-width="1.5" stroke-linecap="round" opacity=".4"/>
            <circle cx="710" cy="400" r="4" fill="#c4b5fd" filter="url(#glow)"/>
          </g>
          <g class="orbit-arm-ccw">
            <path d="M 400 170 A 230 230 0 0 0 185 330" fill="none" stroke="#8b5cf6" stroke-width="1.5" stroke-linecap="round" opacity=".38"/>
            <circle cx="400" cy="170" r="3.5" fill="#c4b5fd"/>
          </g>
        </svg>
      </div>

      <!-- 12 data-comet beams converging to vault logo -->
      <!-- Logo centre ≈ 50% horizontal, 26% vertical of the viewport -->
      <div class="login-beams">
        <div v-for="beam in beams" :key="beam.angle"
          class="login-beam-wrap"
          :style="{ left:'50%', top:'26%', transform:`rotate(${beam.angle}deg)`, transformOrigin:'0 0' }">
          <div class="login-beam"
            :class="beam.cls"
            :style="{ width: beam.w + 'px', animationDelay: beam.delay + 's' }">
          </div>
        </div>
      </div>
    </div>
    <!-- ── End background scene ───────────────────────────── -->

    <!-- ── Login card ────────────────────────────────────── -->
    <div class="login-shell login-content">

      <!-- Brand with energy halo -->
      <div class="brand-row">
        <div class="login-brand-icon-wrap">
          <div class="login-halo-outer"></div>
          <div class="login-halo-inner"></div>
          <div class="login-brand-icon">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
              <path d="M12 3L20 7.5V12C20 16.42 16.47 20.5 12 21.5C7.53 20.5 4 16.42 4 12V7.5L12 3Z"
                stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
              <path d="M8.5 12L11.5 15L15.5 9.5"
                stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
        </div>
        <span class="brand-name">ArcVault</span>
      </div>

      <!-- Card -->
      <div class="login-card form-card">
        <div class="login-card-header">
          <h1 class="login-title">Sign in</h1>
          <p class="login-sub">Backup orchestrator console</p>
        </div>

        <div v-if="error" class="error">{{ error }}</div>

        <form @submit.prevent="handleLogin" class="login-form">
          <div class="field">
            <label for="username">Username</label>
            <div class="input-wrap">
              <svg class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none">
                <circle cx="7" cy="5" r="2.5" stroke="currentColor" stroke-width="1.2"/>
                <path d="M2 12c0-2.76 2.24-5 5-5s5 2.24 5 5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
              </svg>
              <input id="username" v-model="username" type="text"
                autocomplete="username" required placeholder="Username" />
            </div>
          </div>

          <div class="field">
            <label for="password">Password</label>
            <div class="input-wrap">
              <svg class="input-icon" width="14" height="14" viewBox="0 0 14 14" fill="none">
                <rect x="3" y="6" width="8" height="6" rx="1" stroke="currentColor" stroke-width="1.2"/>
                <path d="M5 6V4.5a2 2 0 0 1 4 0V6" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
                <circle cx="7" cy="9" r=".75" fill="currentColor"/>
              </svg>
              <input id="password" v-model="password" type="password"
                autocomplete="current-password" required placeholder="Password" />
            </div>
          </div>

          <!-- 2FA (unchanged from original) -->
          <div v-if="requiresMfa" class="field">
            <label for="mfa">Authenticator code</label>
            <input id="mfa" v-model="mfaCode" type="text"
              inputmode="numeric" pattern="[0-9]{6}" maxlength="6"
              autocomplete="one-time-code" placeholder="000000" />
          </div>

          <button type="submit" class="btn btn-primary login-submit" :disabled="loading">
            <span v-if="loading" class="spinner"></span>
            <svg v-else width="13" height="13" viewBox="0 0 13 13" fill="none">
              <path d="M4.5 2.5l4 4-4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
            {{ loading ? 'Signing in…' : 'Sign in' }}
          </button>
        </form>
      </div>

      <p class="login-footer">ArcVault — Distributed backup orchestration</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth.js'

const router   = useRouter()
const auth     = useAuth()

const username   = ref('')
const password   = ref('')
const mfaCode    = ref('')
const loading    = ref(false)
const error      = ref(null)
const requiresMfa = ref(false)

// 12 beams: varying angle, width, weight, delay
const beams = [
  { angle: 12,  w: 195, cls: '',      delay: 0    },
  { angle: 42,  w: 168, cls: 'thin',  delay: 0.25 },
  { angle: 72,  w: 215, cls: 'thick', delay: 0.5  },
  { angle: 102, w: 178, cls: '',      delay: 0.75 },
  { angle: 132, w: 200, cls: '',      delay: 1.0  },
  { angle: 162, w: 172, cls: 'thin',  delay: 1.25 },
  { angle: 192, w: 210, cls: 'thick', delay: 1.5  },
  { angle: 222, w: 183, cls: '',      delay: 1.75 },
  { angle: 252, w: 197, cls: '',      delay: 2.0  },
  { angle: 282, w: 172, cls: 'thin',  delay: 2.25 },
  { angle: 312, w: 205, cls: 'thick', delay: 2.5  },
  { angle: 342, w: 180, cls: '',      delay: 2.75 },
]

async function handleLogin() {
  error.value   = null
  loading.value = true
  try {
    const result = await auth.login(username.value, password.value, mfaCode.value || undefined)
    if (result?.requiresMfa) { requiresMfa.value = true; return }
    router.push('/')
  } catch (e) {
    error.value = e.message || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* Only truly view-specific overrides remain.
   Base visual tokens: theme-royal-purple.css
   Animation classes:  login-animation.css
   Components (.btn, .form-card, .field, .input-wrap, .error): style.css */

.login-page {
  min-height: 100vh;
  background: radial-gradient(ellipse 130% 100% at 48% 36%, #1d1240 0%, #0e0820 46%, #050211 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;
}

.login-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  width: 100%;
  max-width: 376px;
  position: relative;
  z-index: 10;
  padding: 24px;
}

.brand-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.brand-name {
  font-family: var(--font-display);
  font-size: 1.45rem;
  font-weight: 800;
  letter-spacing: .04em;
  color: var(--text-primary);
  text-shadow: 0 0 30px rgba(196,181,253,.4);
}

.login-card-header { margin-bottom: 24px; }
.login-title {
  margin: 0 0 4px;
  font-family: var(--font-display);
  font-size: 1.3rem;
  font-weight: 700;
  color: var(--text-primary);
}
.login-sub  { margin: 0; font-size: .85rem; color: var(--text-secondary); }
.login-form { display: flex; flex-direction: column; gap: 17px; }

.login-submit { margin-top: 3px; gap: 8px; }

.login-footer {
  font-size: .74rem;
  color: rgba(167,139,250,.38);
  text-align: center;
  letter-spacing: .01em;
  margin: 0;
}

/* Orbit SVG (kept from original, reduced opacity to complement beams) */
.orbit-scene   { position: absolute; inset: 0; pointer-events: none; }
.orbit-rings   { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; }
.orbit-svg     { width: 880px; height: 880px; flex-shrink: 0; opacity: .6; }
.orbit-dashed  { transform-origin: 400px 400px; animation: avSpin 120s linear infinite; }
.orbit-arm-cw  { transform-origin: 400px 400px; animation: avSpin 60s linear infinite; }
.orbit-arm-ccw { transform-origin: 400px 400px; animation: avSpin 80s linear infinite reverse; }
</style>
