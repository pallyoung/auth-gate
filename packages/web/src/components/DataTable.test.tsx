import { vi } from 'vitest'

vi.hoisted(() => {
  process.env.NODE_ENV = 'test'
})

import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { DataTable } from './DataTable'
import { renderWithI18n } from '../test/render'

describe('DataTable i18n', () => {
  it('renders shared table labels in zh-CN', async () => {
    await renderWithI18n(
      <DataTable
        columns={[{ key: 'name', header: 'Name' }]}
        data={[]}
        onEdit={vi.fn()}
      />,
      { locale: 'zh-CN' }
    )

    expect(screen.getByText('操作')).toBeInTheDocument()
    expect(screen.getAllByText('暂无数据')).toHaveLength(2)
  })
})
