<!-- src/views/VersionManagerView.vue -->
<script setup>
import { ref, onMounted, watch } from 'vue'
import { API_URL } from '@/config'

const projects = ref([])
const selectedProject = ref('')
const versions = ref([])

const newVersion = ref({ name: '', version_type: 'release' })

const fetchProjects = async () => {
  const token = localStorage.getItem('bunny_token')
  try {
    const res = await fetch(`${API_URL}/projects`, { headers: { 'Authorization': `Bearer ${token}` } })
    if (res.ok) {
      projects.value = await res.json()
      if (projects.value.length > 0) selectedProject.value = projects.value[0].id
    }
  } catch (e) {
    console.error("Failed to load projects", e)
  }
}

const loadVersions = async () => {
  if (!selectedProject.value) return
  const token = localStorage.getItem('bunny_token')
  try {
    const res = await fetch(`${API_URL}/projects/${selectedProject.value}/versions`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      versions.value = Array.isArray(data) ? data : []
    }
  } catch (e) {
    console.error("Failed to load versions", e)
  }
}

watch(selectedProject, loadVersions)
onMounted(fetchProjects)

const addVersion = async () => {
  if (!selectedProject.value) return
  const token = localStorage.getItem('bunny_token')
  
  try {
    const res = await fetch(`${API_URL}/projects/${selectedProject.value}/versions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(newVersion.value)
    })
    
    if (!res.ok) throw new Error(await res.text())
    
    const saved = await res.json()
    versions.value.unshift(saved) // Add to top of list
    newVersion.value.name = '' // Reset input
  } catch (e) {
    alert("Error adding version: " + e.message)
  }
}

const deleteVersion = async (id) => {
  if (!confirm("Are you sure you want to delete this version?")) return
  const token = localStorage.getItem('bunny_token')
  
  try {
    const res = await fetch(`${API_URL}/versions/${id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${token}` }
    })
    
    if (!res.ok) throw new Error(await res.text())
    
    versions.value = versions.value.filter(v => v.id !== id)
  } catch (e) {
    alert("Error deleting version: " + e.message)
  }
}
</script>

<template>
  <main class="manager-container">
    <div class="header">
      <h1>Release & Build Manager</h1>
      <select v-model="selectedProject" class="project-selector">
        <option v-for="p in projects" :key="p.id" :value="p.id">
          {{ p.name }} ({{ p.project_key }})
        </option>
      </select>
    </div>

    <!-- ADD FORM -->
    <section class="panel add-panel">
      <h3>Create New Version</h3>
      <form @submit.prevent="addVersion" class="inline-form">
        <input v-model="newVersion.name" placeholder="e.g. v2.1.0 or Build-592" required />
        <select v-model="newVersion.version_type">
          <option value="release">Target Release</option>
          <option value="build">Build Version</option>
        </select>
        <button type="submit" class="btn-primary">Add</button>
      </form>
    </section>

    <!-- LISTS -->
    <div class="lists-grid">
      <section class="panel">
        <h2>Target Releases</h2>
        <ul class="version-list">
          <li v-for="v in (versions || []).filter(v => v.version_type === 'release')" :key="v.id">
            <span class="badge-release">{{ v.name }}</span>
            <button @click="deleteVersion(v.id)" class="btn-delete">❌</button>
          </li>
          <li v-if="!versions.some(v => v.version_type === 'release')" class="empty">No releases found.</li>
        </ul>
      </section>

      <section class="panel">
        <h2>Builds</h2>
        <ul class="version-list">
          <li v-for="v in (versions || []).filter(v => v.version_type === 'build')" :key="v.id">
            <span class="badge-build">{{ v.name }}</span>
            <button @click="deleteVersion(v.id)" class="btn-delete">❌</button>
          </li>
          <li v-if="!versions.some(v => v.version_type === 'build')" class="empty">No builds found.</li>
        </ul>
      </section>
    </div>
  </main>
</template>

<style scoped>
.manager-container { max-width: 900px; margin: 40px auto; padding: 0 20px; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.project-selector { padding: 10px; border-radius: 6px; font-size: 1.1em; border: 1px solid #ddd; }
.panel { background: white; padding: 25px; border-radius: 8px; border: 1px solid #e4e4e7; box-shadow: 0 2px 10px rgba(0,0,0,0.05); }
.add-panel { margin-bottom: 30px; background: #f8fafc; }
.inline-form { display: flex; gap: 15px; margin-top: 15px; }
.inline-form input { flex: 2; padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
.inline-form select { flex: 1; padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
.btn-primary { background: #18181b; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
.lists-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 30px; }
.version-list { list-style: none; padding: 0; margin: 0; }
.version-list li { display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #eee; }
.version-list li:last-child { border-bottom: none; }
.badge-release { background: #dbeafe; color: #1e40af; padding: 4px 10px; border-radius: 4px; font-weight: 500; }
.badge-build { background: #f3e8ff; color: #6b21a8; padding: 4px 10px; border-radius: 4px; font-weight: 500; }
.btn-delete { background: none; border: none; cursor: pointer; font-size: 1.1em; opacity: 0.6; }
.btn-delete:hover { opacity: 1; }
.empty { color: #94a3b8; font-style: italic; }
</style>