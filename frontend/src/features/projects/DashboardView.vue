<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

import { healthApi } from '../../api/client'
import { useAsync } from '../../composables/useAsync'

const { loading, error, run } = useAsync<{ status: string }>()
const status = ref<string | null>(null)

async function checkHealth() {
  try {
    const result = await run(() => healthApi.get())
    status.value = result.status
    ElMessage.success(result.status)
  } catch {
    status.value = null
  }
}
</script>

<template>
  <el-container class="dashboard">
    <el-header>
      <div>
        <strong>Web Studio</strong>
        <span>Projects</span>
      </div>
    </el-header>

    <el-main>
      <el-card>
        <template #header>
          <div class="card-header">
            <span>Projects</span>
            <el-button
              type="primary"
              :loading="loading"
              @click="checkHealth"
            >
              Check API
            </el-button>
          </div>
        </template>

        <el-alert
          v-if="error"
          title="API check failed"
          type="error"
          show-icon
        />
        <el-alert
          v-else-if="status"
          :title="'API status: ' + status"
          type="success"
          show-icon
        />
        <el-empty
          v-else
          description="No projects yet"
        />
      </el-card>
    </el-main>
  </el-container>
</template>

<style scoped>
.dashboard {
  min-height: 100vh;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
