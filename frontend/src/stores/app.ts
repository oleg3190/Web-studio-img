import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const locale = ref<'ru' | 'en'>('ru')
  const isBusy = ref(false)
  const error = ref<string | null>(null)

  const hasError = computed(() => error.value !== null)

  function setBusy(value: boolean) { isBusy.value = value }
  function setError(message: string | null) { error.value = message }
  function setLocale(value: 'ru' | 'en') { locale.value = value }

  return { locale, isBusy, error, hasError, setBusy, setError, setLocale }
})
