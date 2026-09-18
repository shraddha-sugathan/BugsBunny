<!-- src/views/DashboardView.vue -->
<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API_URL } from '@/config'

const router = useRouter()
const form = ref({ project_id: '', title: '', body: '', priority: 'low', component: '', module: '', release: '' }) 
const isSubmitting = ref(false)
const projects = ref([])
const stagedFiles = ref([])
const submitStatus = ref('Submit Issue')

const handleFileSelect = (event) => {
  const newFiles = Array.from(event.target.files)
  stagedFiles.value.push(...newFiles)
  event.target.value = '' 
}

const removeFile = (index) => {
  stagedFiles.value.splice(index, 1)
}

const submitIssue = async () => {
  isSubmitting.value = true
  submitStatus.value = 'Creating Issue...'
  
  const selectedProject = projects.value.find(p => p.id === form.value.project_id)
  const projectKey = selectedProject ? selectedProject.project_key : 'UNK'
  const randomId = Math.floor(Math.random() * 1000)
  const newIssueKey = `${projectKey}-${randomId}`
  
  const token = localStorage.getItem('bunny_token')
  if (!token) {
    router.push('/login')
    return
  }

  try {
    // 👈 NEW: Use API_URL
    const response = await fetch(`${API_URL}/issues`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
      body: JSON.stringify({
        issue_key: newIssueKey,
        project_id: form.value.project_id,
        title: form.value.title,
        body: form.value.body,
        priority: form.value.priority,
        component: form.value.component,
        module: form.value.module,
        release: form.value.release
      })
    })

    if (!response.ok) throw new Error("Failed to create issue")

    if (stagedFiles.value.length > 0) {
      const uploadPromises = stagedFiles.value.map(file => {
        const formData = new FormData()
        formData.append('file', file)
        // 👈 NEW: Use API_URL
        return fetch(`${API_URL}/issues/${newIssueKey}/attachments`, {
          method: 'POST',
          headers: { 'Authorization': `Bearer ${token}` }, 
          body: formData
        })
      })
      await Promise.all(uploadPromises) 
    }

    submitStatus.value = 'Redirecting...'
    router.push(`/issue/${newIssueKey}`)
  } catch (error) {
    console.error(error)
  } finally {
    isSubmitting.value = false
    submitStatus.value = 'Submit Issue'
  }
}

onMounted(async () => {
  const token = localStorage.getItem('bunny_token')
  
  // 👈 NEW: Redirect immediately if no token exists
  if (!token) {
    router.push('/login')
    return
  }

  try {
    // 👈 NEW: Use API_URL and handle 401 properly
    const res = await fetch(`${API_URL}/projects`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    
    if (res.status === 401) {
      localStorage.removeItem('bunny_token')
      router.push('/login')
      return
    }

    if (!res.ok) throw new Error("Failed to fetch projects")

    projects.value = await res.json()
    
    if (projects.value.length > 0) {
      form.value.project_id = projects.value[0].id
    }
  } catch (e) {
    console.error("Failed to load projects", e)
  }
})
</script>

<template>
  <main class="page-container">
    <h1>Project Dashboard</h1>
    <p class="subtitle">Welcome to Bugsbunny. Create a new issue to get started.</p>
    
    <form @submit.prevent="submitIssue" class="create-form">
      <div class="form-group">
        <label>Project:</label>
        <select v-model="form.project_id" required>
          <option v-for="p in projects" :key="p.id" :value="p.id">
            {{ p.name }} ({{ p.project_key }})
          </option>
        </select>
      </div>
      <div class="form-group"><label>Title:</label><input v-model="form.title" required /></div>
      <div class="form-group"><label>Description:</label><textarea v-model="form.body" rows="4" required></textarea></div>
      <div class="form-group">
        <label>Priority:</label>
        <select v-model="form.priority">
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="critical">Critical</option>
        </select>
      </div>
      <div class="form-group">
        <label>Component:</label>
        <input v-model="form.component" placeholder="e.g. Database, UI, Auth" />
      </div>
      <div class="form-group">
        <label>Module:</label>
        <input v-model="form.module" placeholder="e.g. Login Screen, API" />
      </div>
      <div class="form-group">
        <label>Target Release:</label>
        <input v-model="form.release" placeholder="e.g. v1.2.0" />
      </div>
      
      <div class="form-group attachments-staging">
        <label>Attach Files (Logs, Dumps, Screenshots):</label>
        <input type="file" @change="handleFileSelect" multiple />
        
        <div v-if="stagedFiles.length > 0" class="staged-list">
          <div v-for="(file, index) in stagedFiles" :key="index" class="staged-item">
            📎 {{ file.name }} ({{ (file.size / 1024).toFixed(1) }} KB)
            <button type="button" @click="removeFile(index)" class="btn-remove">❌</button>
          </div>
        </div>
      </div>
      
      <button type="submit" :disabled="isSubmitting" class="submit-btn">
        {{ submitStatus }}
      </button>
    </form>
  </main>
</template>

<style scoped>
.page-container { max-width: 600px; margin: 40px auto; padding: 0 20px; }
.subtitle { color: #666; margin-bottom: 30px; }
.create-form { background: white; padding: 30px; border: 1px solid #e4e4e7; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.05); }
.form-group { margin-bottom: 20px; display: flex; flex-direction: column; gap: 8px; }
label { font-weight: 500; font-size: 0.9em; }
input, textarea, select { padding: 12px; border: 1px solid #ddd; border-radius: 6px; font-family: inherit; font-size: 1em; }
button { background: #18181b; color: white; border: none; padding: 12px; border-radius: 6px; cursor: pointer; font-weight: bold; font-size: 1em; transition: background 0.2s; }
button:hover:not(:disabled) { background: #3f3f46; }
button:disabled { opacity: 0.5; cursor: not-allowed; }
/* --- File Staging Styles --- */
.attachments-staging { margin-top: 10px; }
.staged-list { margin-top: 10px; display: flex; flex-direction: column; gap: 5px; }

.staged-item { 
  display: flex; 
  justify-content: space-between; 
  align-items: center; 
  background: #fffbeb; 
  border: 1px solid #fcd34d; 
  padding: 8px 12px; 
  border-radius: 4px; 
  font-size: 0.85em; 
  gap: 15px; 
}

.btn-remove { 
  background: transparent !important; 
  padding: 0 !important; 
  border: none; 
  cursor: pointer; 
  color: #dc2626; 
  font-size: 1.1em;
  line-height: 1;
  min-width: auto;
}

.btn-remove:hover {
  transform: scale(1.1); 
}
</style>
