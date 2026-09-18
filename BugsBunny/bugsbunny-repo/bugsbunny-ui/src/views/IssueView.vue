<!-- src/views/IssueView.vue -->
<script setup>
import { ref, onMounted, defineAsyncComponent, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { API_URL } from '@/config'

const route = useRoute()
const router = useRouter()
const issue = ref(null)
const loading = ref(true)

// Edit Mode State
const isEditing = ref(false)
const editForm = ref({ title: '', body: '', priority: '', target_release_id: 0, fixed_in_build_id: 0 })
const isSaving = ref(false)

const comments = ref([])
const newComment = ref('')
const isSubmittingComment = ref(false)

const attachments = ref([])
const isUploading = ref(false)
const stagedFiles = ref([])

const users = ref([])

// --- WORKFLOW STATE ---
const showTransitionModal = ref(false)
const pendingTransition = ref(null)
const transitionForm = ref({ comment: '', fixed_in_build_id: 0 })
const availableTransitions = ref([])
const projectVersions = ref([])

// NEW: Computed properties for clean dropdowns
const releases = computed(() => projectVersions.value.filter(v => v.version_type === 'release'))
const builds = computed(() => projectVersions.value.filter(v => v.version_type === 'build'))

// --- API FETCHERS ---
const fetchVersions = async () => {
  if (!issue.value?.project_id) return
  const token = localStorage.getItem('bunny_token')
  try {
    const res = await fetch(`${API_URL}/projects/${issue.value.project_id}/versions`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) projectVersions.value = await res.json() || []
  } catch (e) {
    console.error("Failed to load versions:", e)
  }
}

const fetchTransitions = async () => {
  if (!issue.value?.issue_key) return
  const token = localStorage.getItem('bunny_token')
  try {
    const response = await fetch(`${API_URL}/issues/${issue.value.issue_key}/transitions`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (response.ok) {
      availableTransitions.value = await response.json() || []
    }
  } catch (error) {
    console.error("Failed to load transitions:", error)
  }
}

const fetchUsers = async () => {
  const token = localStorage.getItem('bunny_token')
  try {
    const res = await fetch(`${API_URL}/users`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (res.ok) users.value = await res.json() || []
  } catch (error) {
    console.error("Failed to load users:", error)
  }
}

const fetchComments = async (id) => {
  const token = localStorage.getItem('bunny_token')
  const res = await fetch(`${API_URL}/issues/${id}/comments`, {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  if (res.ok) comments.value = await res.json() || []
}

const fetchAttachments = async (id) => {
  const token = localStorage.getItem('bunny_token')
  const res = await fetch(`${API_URL}/issues/${id}/attachments`, {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  if (res.ok) attachments.value = await res.json() || []
}

const fetchIssue = async (id) => {
  loading.value = true
  try {
    const token = localStorage.getItem('bunny_token')
    const response = await fetch(`${API_URL}/issues/${id}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })

    if (!response.ok) {
      const errText = await response.text()
      throw new Error(`Server returned ${response.status}: ${errText}`)
    }

    issue.value = await response.json()  
    
    fetchVersions()
    fetchTransitions()
    await fetchAttachments(id) 
    
  } catch (error) {
    console.error("Failed to fetch issue:", error)
  } finally {
    loading.value = false
  }
}

// --- WORKFLOW TRANSITIONS ---
const attemptStatusChange = (transitionRule) => {
  if (transitionRule.requires_comment || transitionRule.requires_build_version) {
    pendingTransition.value = transitionRule
    transitionForm.value = { comment: '', fixed_in_build_id: 0 } // Reset to 0
    showTransitionModal.value = true
  } else {
    executeStatusChange(transitionRule.to_stage_id, '', 0)
  }
}

const submitTransition = () => {
  executeStatusChange(
    pendingTransition.value.to_stage_id, 
    transitionForm.value.comment, 
    transitionForm.value.fixed_in_build_id
  )
  showTransitionModal.value = false
}

const executeStatusChange = async (newStageId, comment, buildVersionId) => {
  const token = localStorage.getItem('bunny_token')
  try {
    const response = await fetch(`${API_URL}/issues/${route.params.id}/transition`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        new_stage_id: newStageId,
        comment: comment,
        fixed_in_build_id: buildVersionId // Send the ID, not the string!
      })
    })

    if (!response.ok) {
      const err = await response.text()
      alert(`Transition failed: ${err}`) 
      return
    }

    fetchIssue(route.params.id)
    fetchComments(route.params.id)
  } catch (error) {
    console.error("Failed to transition issue:", error)
  }
}

// --- FILE & COMMENT HANDLING ---
const handleFileSelect = (event) => {
  stagedFiles.value.push(...Array.from(event.target.files))
  event.target.value = ''
}

const removeFile = (index) => stagedFiles.value.splice(index, 1)

const handleFileUpload = async (event) => {
  const file = event.target.files[0]
  if (!file) return
  
  isUploading.value = true
  const formData = new FormData()
  formData.append('file', file)

  const token = localStorage.getItem('bunny_token')
  await fetch(`${API_URL}/issues/${route.params.id}/attachments`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` }, 
    body: formData
  })

  await fetchAttachments(route.params.id)
  isUploading.value = false
}

const postComment = async () => {
  if (!newComment.value.trim() && stagedFiles.value.length === 0) return
  isSubmittingComment.value = true
  const token = localStorage.getItem('bunny_token')

  if (newComment.value.trim()) {
    await fetch(`${API_URL}/issues/${route.params.id}/comments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
      body: JSON.stringify({ body: newComment.value })
    })
  }

  if (stagedFiles.value.length > 0) {
    const uploadPromises = stagedFiles.value.map(file => {
      const formData = new FormData()
      formData.append('file', file)
      return fetch(`${API_URL}/issues/${route.params.id}/attachments`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
        body: formData
      })
    })
    await Promise.all(uploadPromises)
  }
  
  newComment.value = ''
  stagedFiles.value = [] 
  await fetchComments(route.params.id) 
  await fetchAttachments(route.params.id)
  isSubmittingComment.value = false
}

// --- EDITING & DELETING ---
const startEditing = () => {
  editForm.value = {
    title: issue.value.title || '',
    body: issue.value.body || '',
    assignee_id: issue.value.assignee_id || 0,
    priority: issue.value.custom_data?.priority || 'none',
    component: issue.value.custom_data?.component || '',
    module: issue.value.custom_data?.module || '',
    target_release_id: issue.value.target_release_id || 0,
    fixed_in_build_id: issue.value.fixed_in_build_id || 0
  }
  isEditing.value = true
}

const submitEdit = async () => {
  isSaving.value = true
  try {
    const token = localStorage.getItem('bunny_token')
    const response = await fetch(`${API_URL}/issues/${route.params.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(editForm.value)
    })

    if (!response.ok) {
      const errorText = await response.text()
      if (response.status === 403) {
        alert(`🛑 Access Denied:\n${errorText}`)
        isEditing.value = false
        await fetchIssue(route.params.id)
        return
      }
      throw new Error(errorText || 'Failed to update issue')
    }

    isEditing.value = false
    fetchIssue(route.params.id) 
    
  } catch (error) {
    console.error("Edit failed:", error)
    alert(`⚠️ Failed to save changes:\n\n${error.message}`)
  } finally {
    isSaving.value = false
  }
}

const deleteIssue = async () => {
  if (!window.confirm(`Are you sure you want to delete ${issue.value.issue_key}? This cannot be undone.`)) return
  
  try {
    const token = localStorage.getItem('bunny_token')
    const response = await fetch(`${API_URL}/issues/${route.params.id}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${token}` }
    })

    if (response.ok) {
      router.push('/queue')
    }
  } catch (error) {
    console.error("Failed to delete issue:", error)
  }
}

const SlaPluginWidget = defineAsyncComponent(() => import('../ui-plugins/SlaWidget.vue'))

onMounted(() => { 
  fetchIssue(route.params.id)
  fetchComments(route.params.id)
  fetchUsers() 
})
watch(() => route.params.id, (newId) => { 
  if (newId) {
    fetchIssue(newId)
    fetchComments(newId)
  }
})
</script>
<template>
  <main class="page-container">
    <router-link to="/queue" class="back-link">← Back to Queue</router-link>
    
    <!-- LOADING STATE -->
    <div v-if="loading" class="loading">Loading issue...</div>
    
    <!-- ERROR STATE (If ID doesn't exist) -->
    <div v-else-if="!issue" class="error-state">Issue not found.</div>
    
    <!-- MAIN CONTENT -->
    <div v-else class="issue-card">
      
      <!-- ==================== VIEW MODE ==================== -->
      <div v-if="!isEditing">
        
        <div class="card-header">
          <span class="badge">{{ issue.issue_key }}</span>
          <div class="header-actions">
            <button @click="startEditing" class="edit-btn">✏️ Edit</button>
            <button @click="deleteIssue" class="delete-btn">🗑️ Delete</button>
          </div>
        </div>
        
        <h2>{{ issue.title }}</h2>
        
        <!-- Cleaned up Meta Data Section -->
        <div class="issue-meta">
          <span class="meta-tag">
            <strong>Current Stage:</strong> 
            <span class="badge-status">{{ issue.status }}</span>
          </span>

          <span class="meta-tag">
            <strong>Priority:</strong>
            <span :class="['badge-priority', issue.custom_data?.priority || 'none']">
              {{ (issue.custom_data?.priority || 'none').toUpperCase() }}
            </span>
          </span>

          <span class="meta-tag"><strong>Assignee:</strong> {{ issue.assignee_email || 'Unassigned' }}</span>
          <span class="meta-tag"><strong>Component:</strong> {{ issue.custom_data?.component || 'N/A' }}</span>
          <span class="meta-tag"><strong>Module:</strong> {{ issue.custom_data?.module || 'N/A' }}</span>
          
          <!-- Updated to use new DB columns -->
          <span class="meta-tag"><strong>Target Release:</strong> {{ issue.target_release || 'N/A' }}</span>
          <span class="meta-tag" v-if="issue.fixed_in_build"><strong>Fixed in Build:</strong> {{ issue.fixed_in_build }}</span>
        </div>

        <!-- NEW: DYNAMIC WORKFLOW TRANSITIONS -->
        <div class="transition-section" v-if="availableTransitions && availableTransitions.length > 0">
          <p class="transition-label"><strong>Move Issue To:</strong></p>
          <div class="transition-actions">
            <button 
              v-for="t in availableTransitions" 
              :key="t.id" 
              @click="attemptStatusChange(t)"
              class="btn-transition"
            >
              {{ t.to_stage_name }}
            </button>
          </div>
        </div>
        
        <p class="body-text">{{ issue.body }}</p>
        
        <div class="plugin-slot">
          <p class="slot-label">-- Plugin Slot: Issue Bottom --</p>
          <component :is="SlaPluginWidget" :issueData="issue" />
        </div>

        <!-- UNIFIED ATTACHMENTS GALLERY -->
        <div v-if="attachments && attachments.length > 0" class="unified-gallery">
          <div v-for="att in attachments" :key="att.id" class="attachment-card-wrapper">
            <a :href="att.file_url" target="_blank" download class="attachment-card">
              📎 {{ att.filename }} (Download)
            </a>
            <span class="attachment-meta">Uploaded by: {{ att.user_email?.split('@')[0] || 'System' }}</span>
          </div>
        </div>

        <!-- COMMENTS SECTION -->
        <div class="comments-section">
          <h3>Discussion</h3>
          
          <div v-if="comments.length === 0" class="no-comments">
            No comments yet. Start the conversation!
          </div>
          
          <div class="comment-list">
            <div v-for="c in comments" :key="c.id" class="comment-bubble">
              <div class="comment-header">
                <strong>👤 {{ c.user_email }}</strong>
                <span class="comment-date">{{ new Date(c.created_at).toLocaleString() }}</span>
              </div>
              <div class="comment-body">{{ c.body }}</div>
            </div>
          </div>

          <form @submit.prevent="postComment" class="comment-form">
            <textarea v-model="newComment" rows="3" placeholder="Add a comment or attach files..."></textarea>
            
            <!-- Staged Files List -->
            <div v-if="stagedFiles.length > 0" class="staged-list">
              <div v-for="(file, index) in stagedFiles" :key="index" class="staged-item">
                📎 {{ file.name }}
                <button type="button" @click="removeFile(index)" class="btn-remove">❌</button>
              </div>
            </div>

            <div class="comment-actions">
              <input type="file" @change="handleFileSelect" multiple class="file-input-sm" id="commentFile" />
              <label for="commentFile" class="btn-attach">📎 Attach</label>
              
              <button type="submit" :disabled="isSubmittingComment">
                {{ isSubmittingComment ? 'Posting...' : 'Post' }}
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- ==================== EDIT MODE ==================== -->
      <form v-if="isEditing" @submit.prevent="submitEdit" class="edit-form">
        <div class="form-group">
          <label>Title:</label>
          <input v-model="editForm.title" type="text" required />
        </div>

        <div class="form-group">
          <label>Description:</label>
          <textarea v-model="editForm.body" rows="6" required></textarea>
        </div>

        <div class="form-group">
          <label>Priority:</label>
          <select v-model="editForm.priority">
            <option value="none">None</option>
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
            <option value="critical">Critical</option>
          </select>
        </div>

        <div class="form-group">
          <label>Assign To:</label>
          <select v-model="editForm.assignee_id">
            <option :value="0">Unassigned</option>
            <option v-for="user in users" :key="user.id" :value="user.id">
              {{ user.email }}
            </option>
          </select>
        </div>

        <div class="custom-fields-grid">
          <div class="form-group">
            <label>Component:</label>
            <input v-model="editForm.component" type="text" />
          </div>
          <div class="form-group">
            <label>Module:</label>
            <input v-model="editForm.module" type="text" />
          </div>
          <div class="custom-fields-grid">
            <div class="form-group">
              <label>Component:</label>
              <input v-model="editForm.component" type="text" />
            </div>
            <div class="form-group">
              <label>Module:</label>
              <input v-model="editForm.module" type="text" />
            </div>
            
            <div class="form-group">
              <label>Target Release:</label>
              <select v-model="editForm.target_release_id">
                <option :value="0">Unassigned</option>
                <option v-for="release in releases" :key="release.id" :value="release.id">
                  {{ release.name }}
                </option>
              </select>
            </div>
            
            <div class="form-group">
              <label>Fixed in Build:</label>
              <select v-model="editForm.fixed_in_build_id">
                <option :value="0">Unassigned</option>
                <option v-for="build in builds" :key="build.id" :value="build.id">
                  {{ build.name }}
                </option>
              </select>
            </div>
          </div>
        </div>

        <div class="actions">
          <button type="button" class="btn-cancel" @click="isEditing = false">Cancel</button>
          <button type="submit" class="btn-primary">Save Changes</button>
        </div>
      </form>
      
    </div>

    <!-- TRANSITION MODAL -->
    <div v-if="showTransitionModal" class="modal-overlay">
      <div class="modal-content">
        <h2>Update Issue Status</h2>
        <p>Please provide the required information to transition this issue.</p>
        
        <form @submit.prevent="submitTransition">
          <div v-if="pendingTransition?.requires_build_version" class="form-group">
            <label>Fixed In Build Version <span class="required">*</span></label>
            <select v-model="transitionForm.fixed_in_build_id" required>
              <option :value="0" disabled>Select a build...</option>
              <option v-for="build in builds" :key="build.id" :value="build.id">
                {{ build.name }}
              </option>
            </select>
          </div>

          <div v-if="pendingTransition?.requires_comment" class="form-group">
            <label>Resolution Comment <span class="required">*</span></label>
            <textarea v-model="transitionForm.comment" rows="4" placeholder="Explain the fix or reason..." required></textarea>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn-cancel" @click="showTransitionModal = false">Cancel</button>
            <button type="submit" class="btn-confirm">Confirm Transition</button>
          </div>
        </form>
      </div>
    </div>

  </main>
</template>
<style scoped>
.page-container { max-width: 800px; margin: 40px auto; padding: 0 20px; }
.back-link { text-decoration: none; color: #a1a1aa; font-weight: 600; display: inline-block; margin-bottom: 20px;}
.back-link:hover { color: #18181b; }
.issue-meta { display: flex; gap: 15px; background: #f4f4f5; padding: 10px 15px; border-radius: 6px; margin-bottom: 20px; align-items: center; flex-wrap: wrap;}
.meta-tag { font-size: 0.85em; color: #3f3f46; }
.issue-card { background: white; border: 1px solid #e4e4e7; padding: 30px; border-radius: 8px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); }
.card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
.badge { background: #18181b; color: white; padding: 4px 10px; border-radius: 4px; font-size: 0.8em; font-weight: bold; }
.body-text { white-space: pre-wrap; color: #3f3f46; line-height: 1.5; }

.edit-btn { background: none; border: 1px solid #e4e4e7; padding: 6px 12px; border-radius: 4px; cursor: pointer; transition: background 0.2s; }
.edit-btn:hover { background: #f4f4f5; }

.plugin-slot { margin-top: 40px; padding: 20px; border: 2px dashed #e4e4e7; border-radius: 8px; background: #fafafa;}
.slot-label { font-size: 0.75em; color: #a1a1aa; text-transform: uppercase; margin-top: 0; font-weight: bold; }

.modal-overlay {
  position: fixed; top: 0; left: 0; width: 100vw; height: 100vh;
  background: rgba(0, 0, 0, 0.6);
  display: flex; justify-content: center; align-items: center;
  z-index: 1000;
}
.modal-content {
  background: white; padding: 30px; border-radius: 8px;
  width: 90%; max-width: 500px; box-shadow: 0 10px 25px rgba(0,0,0,0.2);
}
.required { color: #dc2626; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
.btn-cancel { background: #e4e4e7; color: #333; }
.btn-confirm { background: #18181b; color: white; }

/* Edit Form Styles */
.edit-form { display: flex; flex-direction: column; gap: 15px; }
.form-group { display: flex; flex-direction: column; gap: 5px; }
.form-group label { font-size: 0.9em; font-weight: bold; color: #3f3f46; }
input, textarea, select { padding: 10px; border: 1px solid #ccc; border-radius: 4px; font-family: inherit; }
.action-buttons { display: flex; gap: 10px; justify-content: flex-end; margin-top: 10px; }
.save-btn { background: #18181b; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; font-weight: bold; }
.cancel-btn { background: white; color: #3f3f46; border: 1px solid #ccc; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
.header-actions { display: flex; gap: 10px; }
.delete-btn { background: none; border: 1px solid #fee2e2; color: #991b1b; padding: 6px 12px; border-radius: 4px; cursor: pointer; transition: background 0.2s; }
.delete-btn:hover { background: #fee2e2; }
.comments-section { margin-top: 40px; border-top: 1px solid #e4e4e7; padding-top: 20px; }
.no-comments { color: #a1a1aa; font-style: italic; margin-bottom: 20px; }
.comment-list { display: flex; flex-direction: column; gap: 15px; margin-bottom: 20px; }
.comment-bubble { background: #fafafa; border: 1px solid #e4e4e7; padding: 15px; border-radius: 8px; }
.comment-header { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 0.9em; }
.comment-date { color: #a1a1aa; }
.comment-body { white-space: pre-wrap; color: #3f3f46; line-height: 1.4; }
.comment-form { display: flex; flex-direction: column; gap: 10px; align-items: flex-end; }
.comment-form textarea { width: 100%; box-sizing: border-box; }
.comment-form button { background: #646cff; color: white; border: none; padding: 8px 16px; border-radius: 4px; font-weight: bold; cursor: pointer; }
.attachments-section { margin-top: 30px; border-top: 1px solid #e4e4e7; padding-top: 20px; }
.upload-controls { margin-bottom: 15px; }
.attachment-gallery { display: flex; gap: 10px; flex-wrap: wrap; }
.attachment-card { background: #e0e7ff; color: #4338ca; padding: 10px 15px; border-radius: 6px; text-decoration: none; font-weight: 500; font-size: 0.9em; transition: 0.2s; }
.attachment-card:hover { background: #c7d2fe; }
.staged-list { margin: 10px 0; display: flex; flex-direction: column; gap: 5px; }
.staged-item { 
  display: flex; 
  justify-content: space-between; 
  align-items: center; /* Centers the X vertically with the text */
  background: #fffbeb; 
  border: 1px solid #fcd34d; 
  padding: 8px 12px; 
  border-radius: 4px; 
  font-size: 0.85em; 
  gap: 15px; /* Adds breathing room between the filename and the button */
}

.btn-remove { 
  background: transparent !important; /* Strips the purple */
  padding: 0 !important; /* Strips the bulky padding */
  border: none; 
  cursor: pointer; 
  color: #dc2626; 
  font-size: 1.1em;
  line-height: 1;
  min-width: auto;
}

.btn-remove:hover {
  transform: scale(1.1); /* Smooth hover pop instead of a color change */
}

.transition-section {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  padding: 15px;
  border-radius: 6px;
  margin: 20px 0;
}
.transition-label {
  margin-top: 0;
  margin-bottom: 10px;
  color: #475569;
}
.transition-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.btn-transition {
  background: #3b82f6;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
  transition: background 0.2s;
}
.btn-transition:hover {
  background: #2563eb;
}
.unified-gallery { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 20px; background: #fafafa; padding: 15px; border-radius: 6px; border: 1px dashed #d4d4d8; }
.attachment-card-wrapper { display: flex; flex-direction: column; gap: 4px; }
.attachment-card { background: #e0e7ff; color: #4338ca; padding: 8px 12px; border-radius: 4px; text-decoration: none; font-size: 0.85em; font-weight: bold; }
.attachment-card:hover { background: #c7d2fe; }
.attachment-meta { font-size: 0.7em; color: #a1a1aa; padding-left: 2px; }

.comment-actions { display: flex; justify-content: space-between; width: 100%; align-items: center; margin-top: 10px; }
.file-input-sm { display: none; /* Hide default input */ }
.btn-attach { cursor: pointer; background: #e4e4e7; padding: 8px 12px; border-radius: 4px; font-size: 0.85em; font-weight: bold; color: #3f3f46; }
.btn-attach:hover { background: #d4d4d8; }
</style>

