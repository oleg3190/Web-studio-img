<!-- eslint-disable vue/max-attributes-per-line, vue/singleline-html-element-content-newline -->
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { generationsApi, iterationsApi, projectsApi, type Generation, type Iteration, type IterationType, type Project } from '../../api/client'

const route = useRoute()
const project = ref<Project | null>(null)
const iterations = ref<Iteration[]>([])
const error = ref<string | null>(null)
const loading = ref(true)
const creating = ref(false)
const generating = ref(false)
const generations = ref<Generation[]>([])
let generationTimer: ReturnType<typeof setInterval> | undefined

const types: IterationType[] = ['idea', 'sketch', 'generation', 'selection', 'composition', 'prompt', 'manual_edit', 'final']
const form = ref<{ type: IterationType; title: string; description: string }>({
  type: 'idea',
  title: '',
  description: '',
})
const generationForm = ref({ prompt: '', negative_prompt: '', aspect_ratio: '1:1', seed: undefined as number | undefined })

async function loadTimeline() {
  const projectId = String(route.params.projectId)
  const [loadedProject, timeline] = await Promise.all([
    projectsApi.get(projectId),
    iterationsApi.list(projectId),
  ])
  project.value = loadedProject
  iterations.value = timeline.iterations
}

async function createIteration() {
  creating.value = true
  error.value = null
  try {
    const projectId = String(route.params.projectId)
    const created = await iterationsApi.create(projectId, {
      type: form.value.type,
      title: form.value.title.trim() || undefined,
      description: form.value.description.trim() || undefined,
    })
    iterations.value.push(created)
    form.value = { type: 'idea', title: '', description: '' }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not create iteration'
  } finally {
    creating.value = false
  }
}

async function createGeneration() {
  generating.value = true
  error.value = null
  try {
    const projectId = String(route.params.projectId)
    const created = await generationsApi.create(projectId, {
      prompt: generationForm.value.prompt.trim(),
      negative_prompt: generationForm.value.negative_prompt.trim() || undefined,
      aspect_ratio: generationForm.value.aspect_ratio,
      seed: generationForm.value.seed,
    }, crypto.randomUUID())
    generations.value = [created, ...generations.value.filter(item => item.id !== created.id)]
    generationForm.value = { prompt: '', negative_prompt: '', aspect_ratio: '1:1', seed: undefined }
    startGenerationPolling()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not queue generation'
  } finally {
    generating.value = false
  }
}

function startGenerationPolling() {
  if (generationTimer) return
  generationTimer = setInterval(async () => {
    const projectId = String(route.params.projectId)
    const active = generations.value.filter(item => item.status === 'queued' || item.status === 'running')
    if (active.length === 0) {
      clearInterval(generationTimer)
      generationTimer = undefined
      return
    }
    await Promise.all(active.map(async item => {
      try {
        const updated = await generationsApi.get(projectId, item.id)
        const index = generations.value.findIndex(candidate => candidate.id === item.id)
        if (index >= 0) generations.value[index] = updated
      } catch {
        // Keep the last known status; the next poll retries.
      }
    }))
  }, 1000)
}

async function cancelGeneration(item: Generation) {
  error.value = null
  try {
    const updated = await generationsApi.cancel(String(route.params.projectId), item.id)
    const index = generations.value.findIndex(candidate => candidate.id === item.id)
    if (index >= 0) generations.value[index] = updated
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not cancel generation'
  }
}

async function restoreIteration(iteration: Iteration) {
  error.value = null
  try {
    const restored = await iterationsApi.restore(String(route.params.projectId), iteration.id)
    iterations.value.push(restored)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not restore iteration'
  }
}

onMounted(async () => {
  try {
    await loadTimeline()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Could not load studio'
  } finally {
    loading.value = false
  }
})
onUnmounted(() => {
  if (generationTimer) clearInterval(generationTimer)
})
</script>

<template>
  <el-container class="studio">
    <el-header class="header">
      <RouterLink to="/projects"><el-button link>← Projects</el-button></RouterLink>
      <strong>{{ project?.name ?? 'Studio' }}</strong>
      <el-tag v-if="project">{{ project.status }}</el-tag>
    </el-header>

    <el-main>
      <el-alert v-if="error" :title="error" type="error" show-icon closable @close="error = null" />
      <el-skeleton v-if="loading" :rows="8" animated />

      <template v-else>
        <el-card class="create-card">
          <template #header>Create iteration</template>
          <el-form label-position="top" @submit.prevent="createIteration">
            <el-form-item label="Type">
              <el-select v-model="form.type">
                <el-option v-for="type in types" :key="type" :label="type" :value="type" />
              </el-select>
            </el-form-item>
            <el-form-item label="Title">
              <el-input v-model="form.title" maxlength="200" show-word-limit />
            </el-form-item>
            <el-form-item label="Description">
              <el-input v-model="form.description" type="textarea" maxlength="5000" show-word-limit />
            </el-form-item>
            <el-button type="primary" :loading="creating" @click="createIteration">Create</el-button>
          </el-form>
        </el-card>


        <el-card class="create-card">
          <template #header>Generate image</template>
          <el-form label-position="top" @submit.prevent="createGeneration">
            <el-form-item label="Prompt">
              <el-input v-model="generationForm.prompt" type="textarea" maxlength="20000" show-word-limit />
            </el-form-item>
            <el-form-item label="Negative prompt">
              <el-input v-model="generationForm.negative_prompt" type="textarea" maxlength="10000" />
            </el-form-item>
            <el-form-item label="Aspect ratio">
              <el-select v-model="generationForm.aspect_ratio">
                <el-option label="1:1" value="1:1" />
                <el-option label="16:9" value="16:9" />
                <el-option label="9:16" value="9:16" />
                <el-option label="4:3" value="4:3" />
              </el-select>
            </el-form-item>
            <el-form-item label="Seed (optional)">
              <el-input-number v-model="generationForm.seed" :min="0" />
            </el-form-item>
            <el-button type="primary" :loading="generating" :disabled="!generationForm.prompt.trim()" @click="createGeneration">Generate</el-button>
          </el-form>
        </el-card>

        <el-card v-if="generations.length" class="timeline-card">
          <template #header>Generation Queue</template>
          <el-timeline>
            <el-timeline-item v-for="item in generations" :key="item.id" :timestamp="new Date(item.created_at).toLocaleString()" placement="top">
              <div class="generation">
                <div class="iteration-head">
                  <el-tag :type="item.status === 'succeeded' ? 'success' : item.status === 'failed' ? 'danger' : 'warning'">{{ item.status }}</el-tag>
                  <strong>{{ item.prompt }}</strong>
                  <el-button v-if="item.status === 'queued' || item.status === 'running'" size="small" @click="cancelGeneration(item)">Cancel</el-button>
                </div>
                <small v-if="item.model_version">Model version: {{ item.model_version }}</small>
                <el-alert v-if="item.error_message" :title="item.error_message" type="error" :closable="false" />
              </div>
            </el-timeline-item>
          </el-timeline>
        </el-card>

        <el-card class="timeline-card">
          <template #header>Creative Timeline</template>
          <el-empty v-if="iterations.length === 0" description="No iterations yet." />
          <el-timeline v-else>
            <el-timeline-item
              v-for="item in iterations"
              :key="item.id"
              :timestamp="new Date(item.created_at).toLocaleString()"
              placement="top"
            >
              <div class="iteration">
                <div class="iteration-head">
                  <el-tag>{{ item.type }}</el-tag>
                  <strong>{{ item.title || 'Untitled iteration' }}</strong>
                  <el-button size="small" @click="restoreIteration(item)">Restore as new iteration</el-button>
                </div>
                <p v-if="item.description">{{ item.description }}</p>
                <small v-if="item.parent_iteration_id">Parent: {{ item.parent_iteration_id }}</small>
              </div>
            </el-timeline-item>
          </el-timeline>
        </el-card>
      </template>
    </el-main>
  </el-container>
</template>

<style scoped>
.studio { min-height: 100vh; }
.header { display: flex; align-items: center; justify-content: space-between; }
.create-card, .timeline-card { margin-bottom: 16px; }
.iteration { display: grid; gap: 8px; }
.iteration-head { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.iteration p { margin: 0; }
.iteration small, .generation small { color: var(--el-text-color-secondary); }
.generation { display: grid; gap: 8px; }
</style>

