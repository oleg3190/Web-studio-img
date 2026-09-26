import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { useAsync } from './useAsync'

describe('useAsync', () => {
  it('tracks loading and resolves data', async () => {
    const asyncState = useAsync<string>()
    const promise = asyncState.run(() => new Promise(resolve => setTimeout(() => resolve('ok'), 5)))

    expect(asyncState.loading.value).toBe(true)
    await promise
    await nextTick()

    expect(asyncState.loading.value).toBe(false)
    expect(asyncState.data.value).toBe('ok')
    expect(asyncState.error.value).toBeNull()
  })

  it('exposes errors and resets loading', async () => {
    const asyncState = useAsync<string>()
    await expect(asyncState.run(async () => { throw new Error('boom') })).rejects.toThrow('boom')

    expect(asyncState.loading.value).toBe(false)
    expect(asyncState.error.value).toBe('boom')
  })
})
