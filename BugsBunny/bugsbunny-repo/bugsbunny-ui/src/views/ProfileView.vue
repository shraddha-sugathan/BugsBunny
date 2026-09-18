<script setup>
import { ref, onMounted } from 'vue'
import { API_URL } from '@/config'

const profile = ref(null)
const loading = ref(true)

const fetchProfile = async () => {
  const token = localStorage.getItem('bunny_token')
  try {
    const response = await fetch(`${API_URL}/profile`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (response.ok) {
      profile.value = await response.json()
    }
  } catch (error) {
    console.error("Failed to load profile", error)
  } finally {
    loading.value = false
  }
}

onMounted(fetchProfile)
</script>

<template>
  <main class="page-container">
    <div v-if="loading" class="loading">Loading metrics...</div>
    
    <div v-else-if="profile">
      <div class="profile-header">
        <div class="avatar-large">👤</div>
        <h2>{{ profile.email }}</h2>
        <p class="role-badge">Bugsbunny User</p>
      </div>

      <div class="stats-grid">
        <div class="stat-card">
          <h3>Total Issues Assigned</h3>
          <div class="stat-number">{{ profile.total_issues }}</div>
        </div>
        <div class="stat-card success">
          <h3>Issues Resolved</h3>
          <div class="stat-number">{{ profile.resolved_count }}</div>
        </div>
      </div>

      <h3 class="section-title">Your Assigned Queue</h3>
      <div v-if="!profile.issues || profile.issues.length === 0" class="no-data">
        No issues found.
      </div>
      
      <div v-else class="issue-list">
        <router-link 
          v-for="issue in profile.issues" 
          :key="issue.issue_key" 
          :to="'/issue/' + issue.issue_key" 
          class="issue-row"
        >
          <span class="issue-id">{{ issue.issue_key }}</span>
          <span class="issue-title">{{ issue.title }}</span>
          <span :class="['status-badge', issue.status]">{{ issue.status }}</span>
        </router-link>
      </div>
    </div>
  </main>
</template>

<style scoped>
.page-container { max-width: 800px; margin: 40px auto; padding: 0 20px; }
.profile-header { text-align: center; margin-bottom: 40px; }
.avatar-large { font-size: 4rem; margin-bottom: 10px; }
.profile-header h2 { margin: 0; color: #18181b; }
.role-badge { display: inline-block; background: #e0e7ff; color: #4338ca; padding: 4px 12px; border-radius: 20px; font-size: 0.85em; font-weight: bold; margin-top: 10px; }

.stats-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; margin-bottom: 40px; }
.stat-card { background: white; border: 1px solid #e4e4e7; padding: 30px; border-radius: 8px; text-align: center; box-shadow: 0 2px 4px rgba(0,0,0,0.02); }
.stat-card h3 { margin: 0 0 10px 0; color: #71717a; font-size: 1em; }
.stat-number { font-size: 3rem; font-weight: bold; color: #18181b; }
.stat-card.success .stat-number { color: #16a34a; }

.section-title { border-bottom: 2px solid #f4f4f5; padding-bottom: 10px; margin-bottom: 20px; }
.issue-list { display: flex; flex-direction: column; gap: 10px; }
.issue-row { display: flex; align-items: center; padding: 15px; background: white; border: 1px solid #e4e4e7; border-radius: 6px; text-decoration: none; color: inherit; transition: box-shadow 0.2s; }
.issue-row:hover { box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); }
.issue-id { font-weight: bold; width: 100px; color: #646cff; }
.issue-title { flex: 1; }
.status-badge { padding: 4px 8px; border-radius: 4px; font-size: 0.8em; font-weight: bold; text-transform: uppercase; background: #f4f4f5; color: #3f3f46; }
.status-badge.resolved { background: #dcfce7; color: #166534; }
.status-badge.open { background: #fee2e2; color: #991b1b; }
.status-badge.in_progress { background: #fef08a; color: #854d0e; }
</style>
