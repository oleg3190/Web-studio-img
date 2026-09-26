import { ref } from 'vue'

export function useAsync<T>() {
  const loading = ref(false)
  const error = ref<string | null>(null)
  const data = ref<T | null>(null)

  async function run(task: () => Promise<T>) {
    loading.value = true
    error.value = null
    try {
      data.value = await task()
      return data.value
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : 'Unexpected error'
      throw cause
    } finally {
      loading.value = false
    }
  }

  return { loading, error, data, run }
}
