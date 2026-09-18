<!-- src/views/QueueView.vue -->
<script setup>
import { ref, onMounted, watch } from 'vue'
import { API_URL } from '@/config'

const issues = ref([]) 
const loading = ref(true)
const users = ref([])

// Filter State
const searchQuery = ref('')
const priorityFilter = ref('all')
const statusFilter = ref('all')
const assigneeFilter = ref('all')
let debounceTimeout = null

const fetchUsers = async () => {
  const token = localStorage.getItem('bunny_token')
  const res = await fetch(`${API_URL}/users`, { 
    headers: { 'Authorization': `Bearer ${token}` } 
  })
  if (res.ok) {
    users.value = await res.json()
  } else {
    console.error("Failed to fetch users")
  }
}

const fetchAllIssues = async () => {
  loading.value = true
  try {
    const params = new URLSearchParams()
    if (searchQuery.value) params.append('search', searchQuery.value)
    if (priorityFilter.value && priorityFilter.value !== 'all') params.append('priority', priorityFilter.value)
    if (statusFilter.value && statusFilter.value !== 'all') params.append('status', statusFilter.value)
    if (assigneeFilter.value && assigneeFilter.value !== 'all') params.append('assignee', assigneeFilter.value)

    const token = localStorage.getItem('bunny_token')
    
    // 👇 Build the URL cleanly, omitting the '?' if parameters are empty
    const queryString = params.toString()
    const url = queryString ? `${API_URL}/issues?${queryString}` : `${API_URL}/issues`

    const response = await fetch(url, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    
    if (!response.ok) {
      const errText = await response.text()
      throw new Error(errText)
    }

    issues.value = await response.json()
  } catch (error) {
    console.error("Failed to fetch queue:", error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchUsers()
  fetchAllIssues()
})

// Watch ALL dropdowns and trigger an instant fetch when any of them change
watch([priorityFilter, statusFilter, assigneeFilter], () => {
  fetchAllIssues()
})

watch(searchQuery, () => {
  clearTimeout(debounceTimeout)
  debounceTimeout = setTimeout(() => { fetchAllIssues() }, 300)
})
</script>

<template>
  <main class="page-container">
    <div class="header">
      <h1>Issue Queue</h1>
      
      <!-- The activated filter bar! -->
      <div class="filter-bar">
  <input 
    v-model="searchQuery" 
    type="text" 
    placeholder="🔍 Search titles..." 
  />
  
  <select v-model="statusFilter">
    <option value="all">All Statuses</option>
    <option value="open">Open</option>
    <option value="in_progress">In Progress</option>
    <option value="resolved">Resolved</option>
    <option value="closed">Closed</option>
  </select>

  <select v-model="priorityFilter">
    <option value="all">All Priorities</option>
    <option value="critical">Critical</option>
    <option value="high">High</option>
    <option value="medium">Medium</option>
    <option value="low">Low</option>
  </select>
  
  <select v-model="assigneeFilter">
    <option value="all">All Assignees</option>
    <option value="unassigned">Unassigned</option>
    <!-- Dynamically render the team members -->
    <option v-for="user in users" :key="user.id" :value="user.id">
      {{ user.email.split('@')[0] }}
    </option>
  </select>
</div>
    </div>

    <div v-if="loading" class="loading">Loading queue...</div>
    
    <!-- Empty State UI -->
    <div v-else-if="issues.length === 0" class="empty-state">
      No issues found matching your filters.
    </div>

    <!-- The New Data Table -->
    <table v-else class="queue-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Title</th>
          <th>Priority</th>
          <th>Status</th>
          <th>Assignee</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="issue in issues" :key="issue.issue_key">
          
          <td class="id-col">
            <router-link :to="`/issue/${issue.issue_key}`">{{ issue.issue_key }}</router-link>
          </td>
          
          <td class="title-col">
            <router-link :to="`/issue/${issue.issue_key}`">{{ issue.title }}</router-link>
          </td>
          
          <td>
            <!-- Using issue.priority from our updated Go struct -->
            <span :class="['badge-priority', issue.priority || 'none']">
              {{ (issue.priority || 'none').toUpperCase() }}
            </span>
          </td>
          
          <td>
            <span :class="['badge-status', issue.status]">
              {{ (issue.status || 'open').replace('_', ' ').toUpperCase() }}
            </span>
          </td>
          
          <td class="assignee-col">
            {{ issue.assignee_email ? issue.assignee_email.split('@')[0] : 'Unassigned' }}
          </td>
          
        </tr>
      </tbody>
    </table>
  </main>
</template>

<style scoped>
.page-container { max-width: 1000px; margin: 40px auto; padding: 0 20px; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.header h1 { margin: 0; color: #18181b; font-size: 1.8em; }

.filter-bar { display: flex; gap: 10px; }
.filter-bar input { padding: 10px 15px; border: 1px solid #ddd; border-radius: 6px; width: 300px; font-size: 0.95em; }
.filter-bar select { padding: 10px 15px; border: 1px solid #ddd; border-radius: 6px; background: white; font-size: 0.95em; cursor: pointer; }

/* Table Styles */
.queue-table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); }
.queue-table th { background: #f4f4f5; text-align: left; padding: 12px 15px; color: #71717a; font-size: 0.85em; text-transform: uppercase; border-bottom: 2px solid #e4e4e7; }
.queue-table td { padding: 12px 15px; border-bottom: 1px solid #e4e4e7; color: #3f3f46; transition: background 0.1s; }
.queue-table tr:hover td { background: #fafafa; }
.queue-table tr:last-child td { border-bottom: none; }

.id-col a { font-weight: bold; color: #646cff; text-decoration: none; }
.id-col a:hover { text-decoration: underline; }

.title-col a { color: #18181b; text-decoration: none; font-weight: 500; font-size: 1.05em; }
.title-col a:hover { color: #646cff; text-decoration: underline; }

.assignee-col { color: #71717a; font-size: 0.9em; text-transform: capitalize; }

/* Status Badges */
.badge-status { padding: 4px 8px; border-radius: 4px; font-size: 0.75em; font-weight: bold; background: #f4f4f5; }
.badge-status.resolved, .badge-status.closed { background: #dcfce7; color: #166534; }
.badge-status.open, .badge-status.rejected { background: #fee2e2; color: #991b1b; }
.badge-status.in_progress, .badge-status.accepted { background: #fef08a; color: #854d0e; }
.badge-status.on_hold, .badge-status.deferred { background: #e0e7ff; color: #4338ca; }

/* Priority Badges */
.badge-priority { font-size: 0.75em; padding: 4px 8px; border-radius: 4px; font-weight: bold; }
.badge-priority.critical { background: #fee2e2; color: #991b1b; }
.badge-priority.high { background: #ffedd5; color: #c2410c; }
.badge-priority.medium { background: #e0e7ff; color: #4338ca; }
.badge-priority.low { background: #f3f4f6; color: #4b5563; }
.badge-priority.none { background: #f4f4f5; color: #a1a1aa; }

.loading { text-align: center; color: #71717a; padding: 40px; font-style: italic; }
.empty-state { padding: 40px; text-align: center; color: #a1a1aa; font-style: italic; background: white; border: 1px dashed #e4e4e7; border-radius: 8px; }
</style>
