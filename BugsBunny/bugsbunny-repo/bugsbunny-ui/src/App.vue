<!-- src/App.vue -->
<script setup>
import { ref } from 'vue'

// Check if a token exists
const isAuthenticated = ref(!!localStorage.getItem('bunny_token'))
const role = ref(localStorage.getItem('bunny_role'))
const userEmail = ref(localStorage.getItem('bunny_email'))

const showProfilePopup = () => {
  alert("🚀 User Profile & Assigned Bugs view is coming soon!")
}

const logout = () => {
  localStorage.removeItem('bunny_token')
  localStorage.removeItem('bunny_role')
  localStorage.removeItem('bunny_email')
  window.location.href = '/login' // Hard reload to clear state
}
</script>

<template>
  <div id="app">
    <nav class="top-nav">
      <div class="logo">Bugsbunny 🥕</div>
      
      <!-- Only show navigation if logged in -->
      <div v-if="isAuthenticated" class="nav-links">
        <router-link to="/">Dashboard</router-link>
        <router-link to="/queue">Issue Queue</router-link>
        
        <!-- Future Admin Console Link -->
        <router-link v-if="role === 'admin'" to="/admin" style="color: #fbbf24; margin-left: 10px;">🛡️ Admin</router-link>
        <!--<span v-if="role === 'admin'" style="color: #fbbf24; margin-left: 10px;">🛡️ Admin</span>-->
        <!-- Replace your old user-profile div with this -->
<router-link to="/profile" class="user-profile" style="text-decoration: none;">
  <span class="avatar">👤</span>
  <span class="email" style="color: white;">{{ userEmail }}</span>
</router-link>
        <button @click="logout" class="logout-btn">Log Out</button>
      </div>
    </nav>
    
    <router-view /> 
  </div>
</template>

<style>
/* Global styles for the app shell */
body { margin: 0; font-family: system-ui, sans-serif; background: #f4f4f5; }
.top-nav { background: #18181b; color: white; padding: 15px 40px; display: flex; justify-content: space-between; align-items: center; }
.logo { font-weight: bold; font-size: 1.2em; }
.nav-links { display: flex; gap: 20px; }
.nav-links a { color: #a1a1aa; text-decoration: none; font-weight: 500; transition: color 0.2s; }
.nav-links a:hover, .nav-links a.router-link-active { color: white; }
.admin-link { color: #fbbf24 !important; margin-left: 10px; }
.user-profile { display: flex; align-items: center; gap: 8px; margin-left: auto; padding-left: 20px; border-left: 1px solid #3f3f46; cursor: pointer; transition: opacity 0.2s; }
.user-profile:hover { opacity: 0.8; }
.avatar { font-size: 1.2em; }
.email { color: #e4e4e7; font-size: 0.9em; font-weight: 500; }
</style>
