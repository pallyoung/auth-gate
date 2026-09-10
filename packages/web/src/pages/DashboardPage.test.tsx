import { act, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DashboardPage } from './DashboardPage'
import { systemApi } from '../lib/api'
import { renderWithI18n } from '../test/render'

vi.mock('../lib/api', () => ({ systemApi: { stats: vi.fn() } }))

describe('DashboardPage locale', () => {
  beforeEach(() => { vi.mocked(systemApi.stats).mockReset() })

  it('localizes core counts and numbers when the language changes', async () => {
    vi.mocked(systemApi.stats).mockResolvedValue({
      cpu_cores: 8, cpu_usage_percent: 5.4, mem_usage_percent: 10, disk_usage_percent: 12,
      mem_used_bytes: 1024, mem_total_bytes: 2048, disk_used_bytes: 4096, disk_total_bytes: 8192,
      uptime_seconds: 1000, goroutines: 1500, heap_alloc_bytes: 1024, heap_inuse_bytes: 2048,
      gc_count: 1234, gc_pause_total_ms: 12.5, active_routes: 2, total_routes: 3,
      hostname: 'gateway.example.com', platform: 'linux', os: 'linux', arch: 'amd64', kernel_version: '6.1', go_version: 'go1.25',
    } as any)
    const { i18n } = await renderWithI18n(<DashboardPage />, { locale: 'zh-CN' })
    expect(await screen.findByText('8 核')).toBeInTheDocument()
    expect(screen.getAllByText('1,500').length).toBeGreaterThan(0)
    await act(async () => { await i18n.changeLanguage('en') })
    expect(screen.getByText('8 cores')).toBeInTheDocument()
    expect(screen.getByText('gateway.example.com')).toBeInTheDocument()
    expect(systemApi.stats).toHaveBeenCalledTimes(1)
  })

  it('translates a displayed request failure without retrying or showing a raw transport error', async () => {
    vi.mocked(systemApi.stats).mockRejectedValue(new Error('raw transport failure'))
    const { i18n } = await renderWithI18n(<DashboardPage />, { locale: 'en' })
    expect(await screen.findByText('Could not load system statistics. Check your connection and try again.')).toBeInTheDocument()
    await act(async () => { await i18n.changeLanguage('zh-CN') })
    expect(screen.getByText('无法加载系统统计。请检查网络连接后重试。')).toBeInTheDocument()
    expect(screen.queryByText('raw transport failure')).not.toBeInTheDocument()
  })
})
