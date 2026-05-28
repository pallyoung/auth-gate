import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { CertificateForm } from './CertificateForm'
import { renderWithI18n } from '../test/render'

describe('CertificateForm', () => {
  it('offers only supported DNS provider options', async () => {
    await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    const providerSelect = screen.getByLabelText('DNS Provider')

    expect(screen.getByRole('option', { name: 'CloudFlare' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'AWS Route53' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Manual (DIY)' })).not.toBeInTheDocument()
    expect(providerSelect).toHaveValue('cloudflare')
  })

  it('does not render manual DNS guidance that points to an unsupported flow', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    await user.selectOptions(screen.getByLabelText('DNS Provider'), 'route53')

    expect(screen.queryByText('Manual Mode:')).not.toBeInTheDocument()
    expect(screen.queryByText('Manual (DIY)')).not.toBeInTheDocument()
  })

  it('prevents duplicate submissions while certificate provisioning is pending', async () => {
    let resolveSubmit: (() => void) | undefined
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve
        })
    )
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'Billing Certificate')
    await user.type(screen.getByLabelText('Domain'), 'billing.example.com')
    await user.type(screen.getByLabelText('CloudFlare API Token'), 'cf_test_token')

    const submitButton = screen.getByRole('button', { name: 'Provision Certificate' })
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
    const { container } = await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'Billing Certificate')
    await user.type(screen.getByLabelText('Domain'), 'billing.example.com')
    await user.type(screen.getByLabelText('CloudFlare API Token'), 'cf_test_token')

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

  it('rejects whitespace-only CloudFlare tokens before submit', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'Billing Certificate')
    await user.type(screen.getByLabelText('Domain'), 'billing.example.com')
    await user.type(screen.getByLabelText('CloudFlare API Token'), '   ')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(await screen.findByText('CloudFlare API token is required')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('rejects whitespace-only Route53 keys before submit', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.selectOptions(screen.getByLabelText('DNS Provider'), 'route53')
    await user.type(screen.getByLabelText('Certificate Name'), 'Billing Certificate')
    await user.type(screen.getByLabelText('Domain'), 'billing.example.com')
    await user.type(screen.getByLabelText('AWS Access Key ID'), '   ')
    await user.type(screen.getByLabelText('AWS Secret Access Key'), '\t')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(
      await screen.findByText('AWS Access Key ID and Secret Access Key are required for Route53')
    ).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('re-enables submit after a successful synchronous provisioning handler returns', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'Billing Certificate')
    await user.type(screen.getByLabelText('Domain'), 'billing.example.com')
    await user.type(screen.getByLabelText('CloudFlare API Token'), 'cf_test_token')

    const submitButton = screen.getByRole('button', { name: 'Provision Certificate' })
    await user.click(submitButton)

    expect(onSubmit).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('button', { name: 'Provision Certificate' })).not.toBeDisabled()
  })

  it('submits a trimmed domain when the input has surrounding whitespace', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'Billing Certificate')
    await user.type(screen.getByLabelText('Domain'), ' billing.example.com ')
    await user.type(screen.getByLabelText('CloudFlare API Token'), 'cf_test_token')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'Billing Certificate',
      domain: 'billing.example.com',
      dns_provider: 'cloudflare',
      provider_config: {
        api_token: 'cf_test_token',
      },
    })
    expect(screen.queryByText('Invalid domain format. Use something like "example.com" or "*.example.com".')).not.toBeInTheDocument()
  })

  it('retranslates the current validation error when the language changes', async () => {
    const user = userEvent.setup()
    const view = await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'Billing Certificate')
    await user.type(screen.getByLabelText('Domain'), 'not-a-domain')
    await user.type(screen.getByLabelText('CloudFlare API Token'), 'cf_test_token')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(
      await screen.findByText('Invalid domain format. Use something like "example.com" or "*.example.com".')
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(
      screen.getByText('域名格式无效。请使用类似 "example.com" 或 "*.example.com" 的格式。')
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Invalid domain format. Use something like "example.com" or "*.example.com".')
    ).not.toBeInTheDocument()
  })
})
