<!-- src/views/AdminView.vue -->
<script setup>
import { ref, onMounted } from 'vue'
import { API_URL } from '@/config'
import { useRouter } from 'vue-router'

const projectForm = ref({ name: '', project_key: '', description: '' })
const userForm = ref({ email: '', password: '', role: 'developer' })
const message = ref('')

const users = ref([])
const projects = ref([])

const router = useRouter()

const roles = ref([])
const selectedRole = ref('')
const newRoleName = ref('')
const showAddRoleModal = ref(false)

const loadRoles = async () => {
  const token = localStorage.getItem('bunny_token')
  try {
    const res = await fetch(`${API_URL}/roles`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) {
      const data = await res.json()
      roles.value = Array.isArray(data) ? data : []
      if (roles.value.length > 0 && !selectedRole.value) {
        selectedRole.value = roles.value[0].name // or role.id depending on your user table
      }
    }
  } catch (e) {
    console.error('Failed to load roles', e)
  }
}

const createRole = async () => {
  const name = newRoleName.value.trim()
  if (!name) {
    alert('Please enter a role name.')
    return
  }

  // Pre-check against current list
  const isDuplicate = roles.value.some(
    (r) => r.name.toLowerCase() === name.toLowerCase()
  )
  if (isDuplicate) {
    alert(`Role "${name}" already exists.`)
    return
  }

  const token = localStorage.getItem('bunny_token')
  try {
    const res = await fetch(`${API_URL}/roles`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`
      },
      body: JSON.stringify({ name, description: '' })
    })

    if (!res.ok) {
      const errorMsg = await res.text()
      alert(errorMsg || 'Failed to create role')
      return
    }

    const created = await res.json()
    newRoleName.value = ''
    showAddRoleModal.value = false
    await loadRoles()
    userForm.value.role = created.name
  } catch (err) {
    console.error('Error creating role:', err)
    alert('Network error while creating role.')
  }
}

const deleteRole = async (roleId) => {
  if (!confirm('Are you sure you want to delete this role?')) return

  const token = localStorage.getItem('bunny_token')
  try {
    const res = await fetch(`${API_URL}/roles/${roleId}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })

    if (!res.ok) {
      const errorText = await res.text()
      alert(errorText || 'Failed to delete role')
      return
    }

    // Refresh role list after deletion
    await loadRoles()
    
    // Fallback if the selected role was deleted
    if (roles.value.length > 0) {
      userForm.value.role = roles.value[0].name
    }
  } catch (err) {
    console.error('Error deleting role:', err)
  }
}

const fetchUsers = async () => {
  const token = localStorage.getItem('bunny_token')
  const response = await fetch(`${API_URL}/admin/users`, {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  if (response.ok) users.value = await response.json()
}

const fetchProjects = async () => {
  const token = localStorage.getItem('bunny_token')
  const response = await fetch(`${API_URL}/admin/projects`, {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  if (response.ok) projects.value = await response.json()
}

// Load data when the page opens
onMounted(() => {
  fetchUsers()
  fetchProjects()
  loadRoles()
})

const toggleUserStatus = async (id) => {
  const token = localStorage.getItem('bunny_token')
  await fetch(`${API_URL}/admin/users/${id}/toggle`, {
    method: 'PUT',
    headers: { 'Authorization': `Bearer ${token}` }
  })
  fetchUsers()
}

const resetPassword = async (id) => {
  const newPassword = prompt("Enter new password for this user:")
  if (!newPassword) return

  const token = localStorage.getItem('bunny_token')
  await fetch(`${API_URL}/admin/users/${id}/password`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify({ password: newPassword })
  })
  alert("Password updated successfully!")
}

const archiveProject = async (id) => {
  if (!confirm("Are you sure you want to archive this project? It will no longer accept new bugs.")) return
  
  const token = localStorage.getItem('bunny_token')
  await fetch(`${API_URL}/admin/projects/${id}/archive`, {
    method: 'PUT',
    headers: { 'Authorization': `Bearer ${token}` }
  })
  fetchProjects()
}

const createProject = async () => {
  const token = localStorage.getItem('bunny_token')
  const res = await fetch(`${API_URL}/projects`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify(projectForm.value)
  })
  if (res.ok) {
    message.value = `Project ${projectForm.value.project_key} created successfully!`
    projectForm.value = { name: '', project_key: '', description: '' }
    await fetchProjects() 
    
    setTimeout(() => message.value = '', 3000)
  }
}

const createUser = async () => {
  const token = localStorage.getItem('bunny_token')
  const res = await fetch(`${API_URL}/users`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify(userForm.value)
  })
  if (res.ok) {
    message.value = `User ${userForm.value.email} created successfully!`
    userForm.value = { email: '', password: '', role: 'developer' }
    await fetchUsers() 
    
    setTimeout(() => message.value = '', 3000)
  }
}
</script>

<template>
  <main class="admin-container">
    <h1>🛡️ Platform Administration</h1>
    <div v-if="message" class="success-msg">{{ message }}</div>

    <div class="admin-grid">
      <!-- Create Project Panel -->
      <div class="admin-panel">
        <h2>Create New Project</h2>
        <form @submit.prevent="createProject" class="admin-form">
          <label>Project Name</label>
          <input v-model="projectForm.name" placeholder="e.g. Mobile App" required />

          <label>Project Key (Prefix)</label>
          <input v-model="projectForm.project_key" placeholder="e.g. MOB" required maxlength="10" style="text-transform: uppercase;" />

          <label>Description</label>
          <textarea v-model="projectForm.description" rows="2"></textarea>

          <button type="submit">Create Project</button>
        </form>
      </div>

      <!-- Invite User Panel -->
      <div class="admin-panel">
        <h2>Invite User</h2>
        <form @submit.prevent="createUser" class="admin-form">
          <label>Email Address</label>
          <input type="email" v-model="userForm.email" placeholder="user@bugsbunny.local" required />

          <label>Temporary Password</label>
          <input type="password" v-model="userForm.password" required />

         <div class="role-section">
          <div class="role-header">
            <label>Platform Role</label>
            <button 
              type="button" 
              @click="showAddRoleModal = !showAddRoleModal" 
              class="text-link-btn">
              {{ showAddRoleModal ? 'Close Role Manager' : '⚙️ Manage Roles' }}
            </button>
          </div>
            <!-- Dynamic Roles Dropdown -->
            <select v-model="userForm.role" required>
              <option v-for="role in roles" :key="role.id" :value="role.name">
                {{ role.name }}
              </option>
            </select>

            <!-- Inline Role Manager Drawer -->
            <div v-if="showAddRoleModal" class="role-manager-box">
              <!-- Add Role Input -->
              <div class="inline-role-box">
                <input 
                  v-model="newRoleName" 
                  placeholder="New role name" 
                  class="role-input"
                  @keyup.enter="createRole"
                />
                <button type="button" @click="createRole" class="btn-sm btn-primary">Add</button>
              </div>

              <!-- Existing Roles List with Delete Action -->
              <div class="existing-roles-list">
                <span class="sub-label">Existing Roles:</span>
                <div v-for="role in roles" :key="role.id" class="role-chip">
                  <span>{{ role.name }}</span>
                  <button 
                    v-if="!['admin', 'administrator'].includes(role.name.toLowerCase())"
                    type="button" 
                    @click="deleteRole(role.id)" 
                    class="role-delete-btn"
                    title="Delete role">
                    &times;
                  </button>
                </div>
              </div>
            </div>
          </div>

          <button type="submit" class="submit-btn">Create User</button>
        </form>
      </div>

      <!-- Manage Projects Panel -->
      <div class="admin-panel">
        <h2>Manage Projects</h2>
        <ul class="admin-list">
          <li v-for="project in projects" :key="project.id" class="list-item">
            <span><strong>{{ project.project_key }}</strong>: {{ project.name }} {{ project.is_archived ? '(Archived)' : '' }}</span>
            <button v-if="!project.is_archived" @click="archiveProject(project.id)" class="btn-sm btn-danger">Archive</button>
          </li>
        </ul>
      </div>

      <!-- Quick Actions / Navigation Cards -->
      <div class="admin-panel">
        <h2>Manage Workflows</h2>
        <p class="panel-desc">Define custom stages and transition rules.</p>
        <router-link to="/admin/workflows" custom v-slot="{ navigate }">
          <button @click="navigate" class="action-btn">Workflow Builder</button>
        </router-link>
      </div>

      <div class="admin-panel">
        <h2>Manage Versions</h2>
        <p class="panel-desc">Add or remove Target Releases and Builds for your projects.</p>
        <router-link to="/admin/versions" custom v-slot="{ navigate }">
          <button @click="navigate" class="action-btn">Version Manager</button>
        </router-link>
      </div>

      <!-- Manage Users Panel -->
      <div class="admin-panel">
        <h2>Manage Users</h2>
        <ul class="admin-list">
          <li v-for="user in users" :key="user.id" class="list-item">
            <div class="user-info">
              <span>{{ user.email }}</span>
              <span class="user-role badge">{{ user.role }}</span>
              <span v-if="user.is_active === false" class="deactivated-tag">Deactivated</span>
            </div>
            <div class="actions">
              <button @click="resetPassword(user.id)" class="btn-sm">🔑 Reset</button>
              <button @click="toggleUserStatus(user.id)" class="btn-sm" :class="{ 'btn-danger': user.is_active !== false }">
                {{ user.is_active !== false ? 'Deactivate' : 'Reactivate' }}
              </button>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </main>
</template>
<style scoped>
.admin-container { max-width: 1000px; margin: 40px auto; padding: 0 20px; }
.success-msg { background: #dcfce7; color: #166534; padding: 12px; border-radius: 6px; margin-bottom: 20px; text-align: center; font-weight: bold; }
.admin-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 30px; }
.admin-panel { background: white; padding: 25px; border-radius: 8px; border: 1px solid #e4e4e7; box-shadow: 0 2px 4px rgba(0,0,0,0.05); }
.admin-form { display: flex; flex-direction: column; gap: 10px; }
label { font-size: 0.9em; font-weight: bold; color: #3f3f46; margin-top: 10px; }
input, textarea, select { padding: 10px; border: 1px solid #ddd; border-radius: 6px; font-family: inherit; }
button { margin-top: 15px; background: #18181b; color: white; border: none; padding: 12px; border-radius: 6px; font-weight: bold; cursor: pointer; transition: 0.2s; }
button:hover { background: #3f3f46; }
.admin-list { list-style: none; padding: 0; margin: 0; }
.list-item { display: flex; justify-content: space-between; align-items: center; padding: 12px 0; border-bottom: 1px solid #e4e4e7; }
.list-item:last-child { border-bottom: none; }
.actions { display: flex; gap: 8px; }
.user-info { display: flex; flex-direction: column; gap: 4px; }
.user-role { font-size: 0.75em; align-self: flex-start; }
.role-section {
  display: flex;
  flex-direction: column;
  margin-bottom: 1rem;
}

.role-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.25rem;
}

.text-link-btn {
  background: none;
  border: none;
  color: #2563eb;
  font-size: 0.8rem;
  cursor: pointer;
  padding: 0;
}

.text-link-btn:hover {
  text-decoration: underline;
}

.inline-role-box {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
  background: #f8fafc;
  padding: 0.5rem;
  border-radius: 4px;
  border: 1px solid #e2e8f0;
}

.role-input {
  flex: 1;
  padding: 0.25rem 0.5rem;
  font-size: 0.85rem;
}

.action-btn {
  margin-top: 1rem;
  background: #1e293b;
  color: white;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
}

.panel-desc {
  color: #64748b;
  font-size: 0.9rem;
  margin-top: 0.5rem;
}
.role-manager-box {
  background: #f8fafc;
  padding: 0.75rem;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
  margin-top: 0.5rem;
}

.existing-roles-list {
  margin-top: 0.75rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.sub-label {
  font-size: 0.75rem;
  color: #64748b;
  width: 100%;
}

.role-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  background: #e2e8f0;
  padding: 0.2rem 0.6rem;
  border-radius: 9999px;
  font-size: 0.8rem;
  color: #334155;
}

.role-delete-btn {
  background: none;
  border: none;
  color: #ef4444;
  font-weight: bold;
  font-size: 1rem;
  cursor: pointer;
  line-height: 1;
  padding: 0;
}

.role-delete-btn:hover {
  color: #b91c1c;
}
.deactivated-tag { font-size: 0.75em; color: #dc2626; font-weight: bold; }
.btn-sm { padding: 6px 10px; font-size: 0.85em; background: #e4e4e7; color: #18181b; border: none; border-radius: 4px; cursor: pointer; }
.btn-danger { background: #fee2e2; color: #dc2626; }
.btn-danger:hover { background: #fca5a5; }
</style>
