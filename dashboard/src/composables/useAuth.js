import { ref, computed } from 'vue'

// SECURITY: JWT tokens stored in localStorage — violates OWASP ASVS Session Management guidelines.
// localStorage is accessible via XSS, has no expiration isolation, and is shared across all
// subdirectories on the same origin. For production hardening, migrate to:
//   - httpOnly SameSite=Strict cookie (recommended) OR
//   - in-memory token with refresh token rotation using a secure cookie
// Tracked as: tech-debt/auth-storage-hardening
// ponytail: Full refactor deferred — token flow is functional and scoped to dashboard alone.

// Role hierarchy: admin > operator > viewer
const ROLE_HIERARCHY = { admin: 3, operator: 2, viewer: 1 }

const currentUser = ref(null)
const isAuthenticated = ref(false)
const rememberMe = ref(false)

// Singleton instance
let authInstance = null

export function useAuth() {
  // Only create once
  if (authInstance) return authInstance

  const userRole = computed(() => currentUser.value?.role || null)

  // Check if token exists and is valid
  function hasValidToken() {
    const token = getToken()
    return !!token
  }

  // Get token from storage (localStorage or memory)
  function getToken() {
    return localStorage.getItem('arcvault_jwt') || localStorage.getItem('arcvault_token') || null
  }

  // Save token to appropriate storage — always persist
  function saveToken(token, remember) {
    rememberMe.value = remember
    localStorage.setItem('arcvault_remember_me', '1')
    localStorage.setItem('arcvault_jwt', token)
    localStorage.setItem('arcvault_token', token)
  }

  // Clear all auth state
  function clearAuth() {
    currentUser.value = null
    isAuthenticated.value = false
    rememberMe.value = false
    localStorage.removeItem('arcvault_jwt')
    localStorage.removeItem('arcvault_token')
    localStorage.removeItem('arcvault_remember_me')
    localStorage.removeItem('arcvault_user')
  }

  // Check role hierarchy - admin > operator > viewer
  function hasRole(requiredRole) {
    if (!currentUser.value) return false
    const userHierarchy = ROLE_HIERARCHY[currentUser.value.role] || 0
    const requiredHierarchy = ROLE_HIERARCHY[requiredRole] || 0
    return userHierarchy >= requiredHierarchy
  }

  // Check if user can access a feature
  function canAccess(feature) {
    if (!isAuthenticated.value) return false

    // Feature access map: feature -> minimum required role
    const accessMap = {
      'users': 'admin',
      'groups': 'admin',
      'change-password': 'operator', // All authenticated users
      'jobs-create': 'operator',
      'jobs-delete': 'operator',
      'templates-create': 'operator',
      'templates-edit': 'operator',
      'agents-view': 'viewer',
      'history-view': 'viewer',
    }

    const required = accessMap[feature]
    if (!required) return true // If feature not in map, allow it

    return hasRole(required)
  }

  // Login with username/password
  async function login(username, password, remember = false) {
    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })

      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Login failed')
      }

      const data = await response.json()
      const { token, role, must_change_password } = data

      // Build user object from login response fields
      const user = { username, role, must_change_password }

      // Always save token (remember me is always on per UX requirement)
      saveToken(token, true)

      // Set user
      currentUser.value = user
      isAuthenticated.value = true
      localStorage.setItem('arcvault_user', JSON.stringify(user))

      return { success: true, user, mustChangePassword: must_change_password }
    } catch (err) {
      return { success: false, error: err.message }
    }
  }

  // Logout
  async function logout() {
    try {
      // Try to notify server
      const token = getToken()
      if (token) {
        await fetch('/api/auth/logout', {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }).catch(() => {}) // Ignore errors
      }
    } finally {
      clearAuth()
    }
  }

  // Change password
  async function changePassword(currentPassword, newPassword) {
    try {
      const token = getToken()
      if (!token) throw new Error('Not authenticated')

      const response = await fetch('/api/auth/change-password', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          old_password: currentPassword,
          new_password: newPassword,
        }),
      })

      if (!response.ok) {
        const error = await response.text()
        throw new Error(error || 'Password change failed')
      }

      return { success: true }
    } catch (err) {
      return { success: false, error: err.message }
    }
  }

  // Refresh token (silent)
  async function refreshToken() {
    try {
      const token = getToken()
      if (!token) return false

      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      })

      if (response.ok) {
        const data = await response.json()
        const { token: newToken } = data

        // Save new token
        const remember = localStorage.getItem('arcvault_remember_me') === '1'
        saveToken(newToken, remember)

        return true
      }

      return false
    } catch (err) {
      console.error('Token refresh failed:', err)
      return false
    }
  }

  // Initialize on first use
  function initialize() {
    const token = getToken()
    const rememberFlag = localStorage.getItem('arcvault_remember_me') === '1'
    rememberMe.value = rememberFlag

    if (token) {
      const userJson = localStorage.getItem('arcvault_user')
      if (userJson) {
        try {
          currentUser.value = JSON.parse(userJson)
          isAuthenticated.value = true
        } catch (err) {
          console.error('Failed to parse stored user:', err)
          clearAuth()
        }
      }
    }
  }

  // Initialize immediately
  initialize()

  authInstance = {
    currentUser,
    isAuthenticated,
    rememberMe,
    userRole,
    login,
    logout,
    changePassword,
    refreshToken,
    hasRole,
    canAccess,
    getToken,
    saveToken,
    clearAuth,
  }

  return authInstance
}
