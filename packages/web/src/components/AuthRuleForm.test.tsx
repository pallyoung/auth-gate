import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'
import { AuthRuleForm } from './AuthRuleForm'
import { renderWithI18n } from '../test/render'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'

const routes: Route[] = [
  {
    id: 'route-1',
    name: 'Protected API',
    host: '',
    path_prefix: '/api',
    backend: 'http://127.0.0.1:9000',
    strip_prefix: true,
    enabled: true,
    priority: 100,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'route-2',
    name: 'Reports API',
    host: '',
    path_prefix: '/reports',
    backend: 'http://127.0.0.1:9100',
    strip_prefix: true,
    enabled: true,
    priority: 75,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
]

function renderForm(rule: AuthRule | null, onSubmit?: (data: AuthRuleInput) => Promise<void> | void) {
  const submitHandler = onSubmit ?? (vi.fn() as (data: AuthRuleInput) => Promise<void> | void)

  return renderWithI18n(
    <AuthRuleForm rule={rule} routes={routes} onSubmit={submitHandler} onCancel={vi.fn()} />,
    { locale: 'en' }
  )
}

describe('AuthRuleForm', () => {
  it('requires a JWT secret when creating a bearer rule', async () => {
    const user = userEvent.setup()

    await renderForm(null)
    await user.selectOptions(screen.getByRole('combobox', { name: 'Type' }), 'bearer')

    expect(screen.getByLabelText('JWT Secret')).toBeInvalid()
  })

  it('submits only bearer config fields for bearer rules', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm(null, onSubmit)

    await user.selectOptions(screen.getByRole('combobox', { name: 'Type' }), 'bearer')
    await user.type(screen.getByLabelText('JWT Secret'), 'shared-secret')
    await user.click(screen.getByRole('button', { name: 'Create Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'bearer',
        config: {
          secret: 'shared-secret',
        },
      })
    })
  })

  it('submits runtime policy fields when provided', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm(null, onSubmit)

    await user.selectOptions(screen.getByRole('combobox', { name: 'Type' }), 'bearer')
    await user.type(screen.getByLabelText('JWT Secret'), 'shared-secret')
    await user.type(screen.getByLabelText('Whitelist'), '127.0.0.1/32, 10.0.0.0/8')
    await user.type(screen.getByLabelText('Rate Limit'), '15')
    await user.type(screen.getByLabelText('Burst'), '30')
    await user.type(screen.getByLabelText('Allowed Origins'), ' https://app.example.com , .example.com ')
    await user.type(screen.getByLabelText('Allowed Methods'), ' GET, POST, OPTIONS ')
    await user.type(screen.getByLabelText('Allowed Headers'), ' Authorization, Content-Type ')
    await user.click(screen.getByLabelText('Allow Credentials'))
    await user.type(screen.getByLabelText('Max Age'), '7200')
    await user.click(screen.getByRole('button', { name: 'Create Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'bearer',
        config: {
          secret: 'shared-secret',
        },
        whitelist: ['127.0.0.1/32', '10.0.0.0/8'],
        rate_limit: 15,
        burst: 30,
        cors_allowed_origins: 'https://app.example.com,.example.com',
        cors_allowed_methods: 'GET,POST,OPTIONS',
        cors_allowed_headers: 'Authorization,Content-Type',
        cors_allow_credentials: true,
        cors_max_age: 7200,
      })
    })
  })

  it('does not block bearer rule edits when the current secret is redacted', async () => {
    await renderForm({
      id: 'rule-1',
      route_id: 'route-1',
      type: 'bearer',
      config: {},
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    expect(screen.getByLabelText('JWT Secret')).toBeValid()
  })

  it('omits an unchanged bearer secret when editing an existing bearer rule', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm({
      id: 'rule-1',
      route_id: 'route-1',
      type: 'bearer',
      config: {},
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }, onSubmit)

    await user.click(screen.getByRole('button', { name: 'Update Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'bearer',
        config: {},
      })
    })
  })

  it('omits a whitespace-only bearer secret when editing an existing bearer rule', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm({
      id: 'rule-1',
      route_id: 'route-1',
      type: 'bearer',
      config: {},
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }, onSubmit)

    await user.type(screen.getByLabelText('JWT Secret'), '   ')
    await user.click(screen.getByRole('button', { name: 'Update Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'bearer',
        config: {},
      })
    })
  })

  it('does not block API key edits when the current secret is redacted', async () => {
    await renderForm({
      id: 'rule-3',
      route_id: 'route-1',
      type: 'apikey',
      config: { header_name: 'X-API-Key' },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    expect(screen.getByLabelText('Secret')).toBeValid()
  })

  it('omits an unchanged API key secret when editing an existing API key rule', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm({
      id: 'rule-3',
      route_id: 'route-1',
      type: 'apikey',
      config: { header_name: 'X-API-Key' },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }, onSubmit)

    await user.click(screen.getByRole('button', { name: 'Update Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'apikey',
        config: {
          header_name: 'X-API-Key',
        },
      })
    })
  })

  it('omits a whitespace-only API key secret when editing an existing API key rule', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm({
      id: 'rule-3',
      route_id: 'route-1',
      type: 'apikey',
      config: { header_name: 'X-API-Key' },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }, onSubmit)

    await user.type(screen.getByLabelText('Secret'), '   ')
    await user.click(screen.getByRole('button', { name: 'Update Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'apikey',
        config: {
          header_name: 'X-API-Key',
        },
      })
    })
  })

  it('does not block basic auth edits when the current password is redacted', async () => {
    await renderForm({
      id: 'rule-2',
      route_id: 'route-1',
      type: 'basic',
      config: { username: 'service-user' },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    expect(screen.getByLabelText('Password')).toBeValid()
  })

  it('omits an unchanged basic auth password when editing an existing basic rule', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm({
      id: 'rule-2',
      route_id: 'route-1',
      type: 'basic',
      config: { username: 'service-user' },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }, onSubmit)

    await user.click(screen.getByRole('button', { name: 'Update Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'basic',
        config: {
          username: 'service-user',
        },
      })
    })
  })

  it('omits a whitespace-only basic auth password when editing an existing basic rule', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderForm({
      id: 'rule-2',
      route_id: 'route-1',
      type: 'basic',
      config: { username: 'service-user' },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }, onSubmit)

    await user.type(screen.getByLabelText('Password'), '   ')
    await user.click(screen.getByRole('button', { name: 'Update Rule' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith({
        route_id: 'route-1',
        type: 'basic',
        config: {
          username: 'service-user',
        },
      })
    })
  })

  it('prevents duplicate submissions while an auth rule save is pending', async () => {
    let resolveSubmit: (() => void) | undefined
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve
        })
    )
    const user = userEvent.setup()

    await renderForm(null, onSubmit)

    await user.selectOptions(screen.getByRole('combobox', { name: 'Type' }), 'bearer')
    await user.type(screen.getByLabelText('JWT Secret'), 'shared-secret')

    const submitButton = screen.getByRole('button', { name: 'Create Rule' })
    await user.click(submitButton)
    await user.click(submitButton)

    expect(onSubmit).toHaveBeenCalledTimes(1)

    resolveSubmit?.()
  })

  it('prevents back-to-back native submissions before the submitting state re-renders', async () => {
    let resolveSubmit: (() => void) | undefined
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve
        })
    )
    const user = userEvent.setup()
    const { container } = await renderForm(null, onSubmit)

    await user.selectOptions(screen.getByRole('combobox', { name: 'Type' }), 'bearer')
    await user.type(screen.getByLabelText('JWT Secret'), 'shared-secret')

    const form = container.querySelector('form')
    expect(form).not.toBeNull()

    await act(async () => {
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(onSubmit).toHaveBeenCalledTimes(1)

    resolveSubmit?.()
  })

  it('refreshes form fields when the edited auth rule changes while the form stays mounted', async () => {
    const view = await renderForm({
      id: 'rule-1',
      route_id: 'route-1',
      type: 'apikey',
      config: {
        header_name: 'X-Billing-Key',
      },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    expect(screen.getByRole('combobox', { name: 'Route' })).toHaveValue('route-1')
    expect(screen.getByRole('combobox', { name: 'Type' })).toHaveValue('apikey')
    expect(screen.getByLabelText('Header Name')).toHaveValue('X-Billing-Key')

    view.rerender(
      <I18nextProvider i18n={view.i18n}>
        <AuthRuleForm
          rule={{
            id: 'rule-2',
            route_id: 'route-2',
            type: 'basic',
            config: {
              username: 'service-user',
            },
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          }}
          routes={routes}
          onSubmit={vi.fn()}
          onCancel={vi.fn()}
        />
      </I18nextProvider>
    )

    expect(screen.getByRole('combobox', { name: 'Route' })).toHaveValue('route-2')
    expect(screen.getByRole('combobox', { name: 'Type' })).toHaveValue('basic')
    expect(screen.getByLabelText('Username')).toHaveValue('service-user')
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
    expect(screen.queryByLabelText('Header Name')).not.toBeInTheDocument()
  })

  it('refreshes form fields when the same auth rule receives updated values while the form stays mounted', async () => {
    const view = await renderForm({
      id: 'rule-1',
      route_id: 'route-1',
      type: 'apikey',
      config: {
        header_name: 'X-Billing-Key',
      },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    view.rerender(
      <I18nextProvider i18n={view.i18n}>
        <AuthRuleForm
          rule={{
            id: 'rule-1',
            route_id: 'route-2',
            type: 'basic',
            config: {
              username: 'service-user-updated',
            },
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-02T00:00:00Z',
          }}
          routes={routes}
          onSubmit={vi.fn()}
          onCancel={vi.fn()}
        />
      </I18nextProvider>
    )

    expect(screen.getByRole('combobox', { name: 'Route' })).toHaveValue('route-2')
    expect(screen.getByRole('combobox', { name: 'Type' })).toHaveValue('basic')
    expect(screen.getByLabelText('Username')).toHaveValue('service-user-updated')
    expect(screen.queryByLabelText('Header Name')).not.toBeInTheDocument()
  })
})
