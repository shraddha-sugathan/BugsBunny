<!-- src/views/DashboardView.vue -->
<template>
  <main class="page-container">
    <div class="header-row">
      <div>
        <h1>Dashboard</h1>
        <p class="subtitle">Platform overview and issue statistics</p>
      </div>
      <router-link to="/issues/new" class="create-btn">+ Create Issue</router-link>
    </div>

    <div v-if="loading" class="state-msg">Loading dashboard metrics...</div>
    <div v-else-if="error" class="state-msg error">{{ error }}</div>

    <div v-else class="stats-grid">
      <div class="stat-card">
        <h3>Active Projects</h3>
        <p class="stat-value">{{ stats.total_projects }}</p>
      </div>
      <div class="stat-card highlight">
        <h3>My Active Issues</h3>
        <p class="stat-value">{{ stats.my_active_issues }}</p>
      </div>
      <div class="stat-card warning">
        <h3>Unassigned Bugs</h3>
        <p class="stat-value">{{ stats.unassigned_bugs }}</p>
      </div>
      <div class="stat-card">
        <h3>Updated (7 Days)</h3>
        <p class="stat-value">{{ stats.recently_updated }}</p>
      </div>
    </div>
  </main>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API_URL } from '@/config'

const router = useRouter()
const stats = ref({
  total_projects: 0,
  my_active_issues: 0,
  unassigned_bugs: 0,
  recently_updated: 0
})
const loading = ref(true)
const error = ref('')

const fetchDashboardStats = async () => {
  const token = localStorage.getItem('bunny_token')
  if (!token) {
    router.push('/login')
    return
  }

  try {
    const res = await fetch(`${API_URL}/analytics/dashboard`, {
      headers: {
        Authorization: `Bearer ${token}`
      }
    })

    if (res.status === 401) {
      localStorage.removeItem('bunny_token')
      router.push('/login')
      return
    }

    if (!res.ok) throw new Error('Failed to load dashboard metrics')

    stats.value = await res.json()
  } catch (err) {
    console.error(err)
    error.value = 'Could not load dashboard statistics.'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchDashboardStats()
})
</script>

<style scoped>
.page-container {
  max-width: 900px;
  margin: 40px auto;
  padding: 0 20px;
}
.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
}
.subtitle {
  color: #666;
  margin: 4px 0 0 0;
}
.create-btn {
  background: #18181b;
  color: white;
  padding: 10px 18px;
  border-radius: 6px;
  text-decoration: none;
  font-weight: bold;
  font-size: 0.9em;
  transition: background 0.2s;
}
.create-btn:hover {
  background: #3f3f46;
}
.state-msg {
  padding: 20px;
  text-align: center;
  color: #666;
}
.state-msg.error {
  color: #dc2626;
}
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}
.stat-card {
  background: white;
  border: 1px solid #e4e4e7;
  border-radius: 8px;
  padding: 24px;
  text-align: center;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.04);
}
.stat-card h3 {
  margin: 0 0 8px 0;
  font-size: 0.95em;
  color: #71717a;
  font-weight: 500;
}
.stat-value {
  font-size: 2.2rem;
  font-weight: 700;
  margin: 0;
  color: #18181b;
}
.stat-card.highlight .stat-value {
  color: #2563eb;
}
.stat-card.warning .stat-value {
  color: #dc2626;
}
</style>