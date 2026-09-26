<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { healthApi } from '../../api/client'
import { useAsync } from '../../composables/useAsync'

const { loading, error, run } = useAsync<{ status: string }>()
const apiStatus = ref('unknown')

async function checkApi() {
  try {
    const result = await run(() => healthApi.get())
    apiStatus.value = result.status
    ElMessage.success('API доступен')
  } catch {
    ElMessage.error(error.value ?? 'API недоступен')
  }
}
</script>

<template>
  <el-container class="page">
    <el-header class="header">
      <div>
        <strong>Web Studio IMG</strong>
        <span class="tag">Create. Refine. Prove.</span>
      </div>
      <el-button :loading="loading" @click="checkApi">Проверить API</el-button>
    </el-header>

    <el-main>
      <el-card shadow="never">
        <template #header>Projects</template>
        <el-empty description="Проекты появятся здесь" />
        <el-alert v-if="error" :title="error" type="error" show-icon />
        <p>API status: <strong>{{ apiStatus }}</strong></p>
      </el-card>
    </el-main>
  </el-container>
</template>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.header { display: flex; align-items: center; justify-content: space-between; background: #fff; border-bottom: 1px solid #ebeef5; }
.tag { margin-left: 12px; color: #909399; font-size: 13px; }
</style>
