export const messages = {
  ru: {
    projects: 'Проекты',
    checkApi: 'Проверить API',
    loading: 'Загрузка…',
    error: 'Ошибка',
  },
  en: {
    projects: 'Projects',
    checkApi: 'Check API',
    loading: 'Loading…',
    error: 'Error',
  },
} as const

export type Locale = keyof typeof messages
