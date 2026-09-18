<!-- src/views/LoginView.vue -->
<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { API_URL } from '@/config'

const router = useRouter()
const email = ref('admin@bugsbunny.local') // Defaulting to make testing fast
const password = ref('admin123')
const errorMsg = ref('')
const isLoggingIn = ref(false)

const handleLogin = async () => {
  isLoggingIn.value = true
  errorMsg.value = ''

  try {
    const response = await fetch(`${API_URL}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email.value, password: password.value })
    })

    if (!response.ok) {
      throw new Error('Invalid email or password')
    }

    const data = await response.json()
    
    // Save the JWT token and Role to the browser
    localStorage.setItem('bunny_token', data.token)
    localStorage.setItem('bunny_role', data.role)
    localStorage.setItem('bunny_email', data.email)

    // Force a hard reload to update the App shell navigation, 
    // then route to the queue
    window.location.href = '/queue'
    
  } catch (error) {
    errorMsg.value = error.message
  } finally {
    isLoggingIn.value = false
  }
}
</script>

<template>
  <main class="login-container">
    <div class="login-box">
      <h2>Sign In to Bugsbunny</h2>
      
      <form @submit.prevent="handleLogin" class="login-form">
        <div class="form-group">
          <label>Email</label>
          <input type="email" v-model="email" required />
        </div>
        
        <div class="form-group">
          <label>Password</label>
          <input type="password" v-model="password" required />
        </div>

        <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>

        <button type="submit" :disabled="isLoggingIn">
          {{ isLoggingIn ? 'Authenticating...' : 'Sign In' }}
        </button>
      </form>
    </div>
  </main>
</template>

<style scoped>
.login-container { display: flex; justify-content: center; align-items: center; min-height: 80vh; }
.login-box { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); width: 100%; max-width: 400px; }
h2 { text-align: center; margin-top: 0; margin-bottom: 25px; color: #18181b; }
.login-form { display: flex; flex-direction: column; gap: 20px; }
.form-group { display: flex; flex-direction: column; gap: 8px; }
label { font-size: 0.9em; font-weight: bold; color: #3f3f46; }
input { padding: 12px; border: 1px solid #ddd; border-radius: 6px; font-size: 1em; }
button { background: #646cff; color: white; border: none; padding: 12px; border-radius: 6px; font-weight: bold; font-size: 1em; cursor: pointer; transition: background 0.2s; }
button:hover:not(:disabled) { background: #535bf2; }
.error-msg { color: #dc2626; font-size: 0.9em; text-align: center; background: #fef2f2; padding: 10px; border-radius: 4px; }
</style>
