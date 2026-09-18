<!-- src/views/WorkflowBuilderView.vue -->
<script setup>
import { ref, onMounted, watch } from 'vue'
import { API_URL } from '@/config'

const projects = ref([])
const selectedProject = ref('')
const roles = ref([])
const stages = ref([])
const transitions = ref([])

// Form states
const newStage = ref({ status_name: '', step_order: 1, sla_hours: 0 })
const newTransition = ref({ from_stage_id: '', to_stage_id: '', required_role_id: '', requires_comment: false, requires_build_version: false })

const existingRules = ref([])

const loadRules = async () => {
  if (!selectedProject.value) return
  const token = localStorage.getItem('bunny_token')
  const res = await fetch(`${API_URL}/projects/${selectedProject.value}/workflow-rules`, {
    headers: { 'Authorization': `Bearer ${token}` }
  })
  if (res.ok) {
    existingRules.value = await res.json() || []
  }
}

const fetchProjectsAndRoles = async () => {
  const token = localStorage.getItem('bunny_token')
  try {
    const [projRes, roleRes] = await Promise.all([
      fetch(`${API_URL}/projects`, { headers: { 'Authorization': `Bearer ${token}` } }),
      fetch(`${API_URL}/roles`, { headers: { 'Authorization': `Bearer ${token}` } })
    ])
    projects.value = await projRes.json()
    // Roles might not exist yet if API isn't written, default to empty array
    roles.value = roleRes.ok ? await roleRes.json() : [{id: 1, name: 'Developer'}, {id: 2, name: 'Lead'}] 
    
    if (projects.value.length > 0) {
      selectedProject.value = projects.value[0].id
    }
  } catch (e) {
    console.error("Failed to load initial data", e)
  }
}

const loadWorkflowContext = async () => {
  if (!selectedProject.value) return
  const token = localStorage.getItem('bunny_token')
  try {
    const stageRes = await fetch(`${API_URL}/projects/${selectedProject.value}/stages`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    stages.value = stageRes.ok ? await stageRes.json() : []
    
    // Default next step order
    newStage.value.step_order = stages.value.length + 1
  } catch (e) {
    console.error("Failed to load stages", e)
  }
}

// Watch for project changes and reload the workflow
watch(selectedProject, () => {
  loadWorkflowContext()
  loadRules()
})

const addStage = async () => {
  if (!selectedProject.value) return
  const token = localStorage.getItem('bunny_token')
  
  try {
    const response = await fetch(`${API_URL}/projects/${selectedProject.value}/stages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(newStage.value)
    })
    
    if (!response.ok) {
        const errorText = await response.text(); // Read the exact backend error
        throw new Error(`Failed to save: ${errorText}`);  
    }

    loadRules()
    const savedStage = await response.json()
    stages.value.push(savedStage) // Update UI with real DB record
    
    // Reset form and increment the step order
    newStage.value = { status_name: '', step_order: stages.value.length + 1, sla_hours: 0 }
  } catch (e) {
    console.error("Error adding stage:", e)
  }
}

const addTransition = async () => {
  if (!selectedProject.value) return
  const token = localStorage.getItem('bunny_token')
  
  // Format the payload: Go expects an integer pointer or null for the role
  const payload = { ...newTransition.value }
  payload.from_stage_id = parseInt(payload.from_stage_id)
  payload.to_stage_id = parseInt(payload.to_stage_id)
  payload.required_role_id = payload.required_role_id ? parseInt(payload.required_role_id) : null

  try {
    const response = await fetch(`${API_URL}/projects/${selectedProject.value}/transitions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(payload)
    })
    
    if (!response.ok) throw new Error("Failed to add transition")
    
    const savedTransition = await response.json()
    transitions.value.push(savedTransition) 
    
    // Reset form
    newTransition.value = { from_stage_id: '', to_stage_id: '', required_role_id: '', requires_comment: false, requires_build_version: false }
  } catch (e) {
    console.error("Error adding transition:", e)
  }
}

onMounted(async () => {
  // 1. Wait for projects and roles to load (sets selectedProject.value)
  await fetchProjectsAndRoles()
  
  // 2. Now that a project is selected, fetch its rules!
  loadRules()
})
</script>

<template>
  <main class="workflow-container">
    <div class="header">
      <h1>Workflow Builder</h1>
      <select v-model="selectedProject" class="project-selector">
        <option v-for="p in projects" :key="p.id" :value="p.id">
          {{ p.name }} ({{ p.project_key }})
        </option>
      </select>
    </div>

    <div class="builder-grid">
      <!-- 1. Stages Panel -->
      <section class="panel">
        <h2>1. Define Stages & SLAs</h2>
        <p class="help-text">Create the statuses an issue can hold in this project.</p>
        
        <div class="stage-list">
          <div v-for="stage in stages" :key="stage.id" class="stage-card">
            <span class="step-badge">{{ stage.step_order }}</span>
            <div class="stage-info">
              <strong>{{ stage.status_name }}</strong>
              <small v-if="stage.sla_hours > 0">SLA: {{ stage.sla_hours }}h</small>
              <small v-else>No SLA</small>
            </div>
          </div>
        </div>

        <form @submit.prevent="addStage" class="inline-form">
          <input v-model="newStage.status_name" placeholder="Status Name (e.g. QA)" required />
          <input v-model.number="newStage.sla_hours" type="number" placeholder="SLA (Hours)" min="0" />
          <button type="submit">Add Stage</button>
        </form>
      </section>

      <!-- 2. Transitions Panel -->
      <section class="panel">
        <h2>2. Transition Rules</h2>
        <p class="help-text">Define how issues move between stages and who can move them.</p>

        <form @submit.prevent="addTransition" class="transition-form">
          <div class="row">
            <select v-model="newTransition.from_stage_id" required>
              <option value="" disabled>From Stage...</option>
              <option v-for="s in stages" :key="s.id" :value="s.id">{{ s.status_name }}</option>
            </select>
            <span>➔</span>
            <select v-model="newTransition.to_stage_id" required>
              <option value="" disabled>To Stage...</option>
              <option v-for="s in stages" :key="s.id" :value="s.id">{{ s.status_name }}</option>
            </select>
          </div>

          <select v-model="newTransition.required_role_id">
            <option value="">Any Role (No Restriction)</option>
            <option v-for="r in roles" :key="r.id" :value="r.id">Requires: {{ r.name }}</option>
          </select>

          <div class="checkboxes">
            <label>
              <input type="checkbox" v-model="newTransition.requires_comment" /> Require Comment
            </label>
            <label>
              <input type="checkbox" v-model="newTransition.requires_build_version" /> Require Build Version
            </label>
          </div>

          <button type="submit">Create Rule</button>
        </form>

        <!-- Active Rules Table -->
        <div class="existing-rules-panel" v-if="existingRules.length > 0">
          <h3 style="margin-top: 30px; border-top: 1px solid #eee; padding-top: 15px;">Active Rules</h3>
          <table class="rule-table" style="width: 100%; text-align: left; margin-top: 10px;">
            <thead>
              <tr>
                <th>From</th>
                <th>→ To</th>
                <th>Role</th>
                <th>Mandatory</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rule in existingRules" :key="rule.id">
                <td><strong>{{ rule.from_stage }}</strong></td>
                <td>{{ rule.to_stage }}</td>
                <td>{{ rule.role_name }}</td>
                <td>
                  <span v-if="rule.requires_comment">Comment</span>
                  <span v-if="rule.requires_build">Build</span>
                  <span v-if="!rule.requires_comment && !rule.requires_build" style="color: #aaa;">None</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

      </section>
    </div>
  </main>
</template>

<style scoped>
.workflow-container { max-width: 1000px; margin: 40px auto; padding: 0 20px; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; }
.project-selector { padding: 10px; border-radius: 6px; font-size: 1.1em; border: 1px solid #ddd; }
.builder-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 30px; }
.panel { background: white; padding: 25px; border-radius: 8px; border: 1px solid #e4e4e7; box-shadow: 0 2px 10px rgba(0,0,0,0.05); }
.help-text { color: #666; font-size: 0.9em; margin-bottom: 20px; }
.stage-list { display: flex; flex-direction: column; gap: 10px; margin-bottom: 20px; }
.stage-card { display: flex; align-items: center; padding: 12px; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; }
.step-badge { background: #18181b; color: white; width: 24px; height: 24px; border-radius: 12px; display: flex; align-items: center; justify-content: center; font-size: 0.8em; font-weight: bold; margin-right: 15px; }
.stage-info { display: flex; flex-direction: column; }
.inline-form { display: flex; gap: 10px; }
.inline-form input { flex: 1; padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
.transition-form { display: flex; flex-direction: column; gap: 15px; background: #f8fafc; padding: 15px; border: 1px dashed #cbd5e1; border-radius: 6px; }
.row { display: flex; align-items: center; gap: 10px; }
.row select { flex: 1; padding: 10px; border: 1px solid #ddd; border-radius: 4px; }
.checkboxes { display: flex; gap: 15px; font-size: 0.9em; }
button { background: #18181b; color: white; border: none; padding: 10px 15px; border-radius: 4px; cursor: pointer; font-weight: bold; }
button:hover { background: #3f3f46; }
</style>