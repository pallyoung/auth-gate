import { vi } from 'vitest'

vi.hoisted(() => {
  process.env.NODE_ENV = 'test'
})

import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LoginPage } from './LoginPage'
import { renderWithI18n } from '../test/render'

describe('LoginPage i18n', () => {
  it('renders translated login copy in zh-CN', async () => {
    await renderWithI18n(<LoginPage onLogin={vi.fn()} />, { locale: 'zh-CN' })

    expect(screen.getByRole('heading', { name: '登录控制台' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument()
    expect(screen.getByLabelText('用户名')).toBeInTheDocument()
  })
})
