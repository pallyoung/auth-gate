import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { CertificateForm } from './CertificateForm'
import { renderWithI18n } from '../test/render'

describe('CertificateForm', () => {
  it('renders in local CA mode by default with name and domain fields', async () => {
    await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    expect(screen.getByText('Local CA Generate')).toBeInTheDocument()
    expect(screen.getByText('Import PEM')).toBeInTheDocument()
    expect(screen.getByLabelText('Certificate Name')).toBeInTheDocument()
    expect(screen.getByLabelText('Domain')).toBeInTheDocument()
    expect(screen.queryByLabelText('Certificate PEM')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Private Key PEM')).not.toBeInTheDocument()
  })

  it('shows PEM fields when import mode is selected', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    await user.click(screen.getByText('Import PEM'))

    expect(screen.getByLabelText('Certificate PEM')).toBeInTheDocument()
    expect(screen.getByLabelText('Private Key PEM')).toBeInTheDocument()
  })

  it('hides PEM fields when switching back to local CA mode', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    await user.click(screen.getByText('Import PEM'))
    expect(screen.getByLabelText('Certificate PEM')).toBeInTheDocument()

    await user.click(screen.getByText('Local CA Generate'))
    expect(screen.queryByLabelText('Certificate PEM')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Private Key PEM')).not.toBeInTheDocument()
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

  it('submits local CA mode without source field', async () => {
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
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'Billing Certificate',
      domain: 'billing.example.com',
    })
  })

  it('submits import mode with source, cert_pem, and key_pem', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.click(screen.getByText('Import PEM'))
    await user.type(screen.getByLabelText('Certificate Name'), 'Imported Cert')
    await user.type(screen.getByLabelText('Domain'), 'example.com')
    await user.type(screen.getByLabelText('Certificate PEM'), '-----BEGIN CERTIFICATE-----\nfakecert\n-----END CERTIFICATE-----')
    await user.type(screen.getByLabelText('Private Key PEM'), '-----BEGIN RSA PRIVATE KEY-----\nfakekey\n-----END RSA PRIVATE KEY-----')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'Imported Cert',
      domain: 'example.com',
      source: 'imported',
      cert_pem: '-----BEGIN CERTIFICATE-----\nfakecert\n-----END CERTIFICATE-----',
      key_pem: '-----BEGIN RSA PRIVATE KEY-----\nfakekey\n-----END RSA PRIVATE KEY-----',
    })
  })

  it('rejects empty cert PEM in import mode before submit', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.click(screen.getByText('Import PEM'))
    await user.type(screen.getByLabelText('Certificate Name'), 'Test')
    await user.type(screen.getByLabelText('Domain'), 'example.com')
    await user.type(screen.getByLabelText('Private Key PEM'), 'some-key')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(await screen.findByText('Certificate PEM content is required')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('rejects empty key PEM in import mode before submit', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.click(screen.getByText('Import PEM'))
    await user.type(screen.getByLabelText('Certificate Name'), 'Test')
    await user.type(screen.getByLabelText('Domain'), 'example.com')
    await user.type(screen.getByLabelText('Certificate PEM'), 'some-cert')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(await screen.findByText('Private key PEM content is required')).toBeInTheDocument()
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
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'Billing Certificate',
      domain: 'billing.example.com',
    })
    expect(screen.queryByText('Invalid domain format')).not.toBeInTheDocument()
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

  it('shows subject fields in local CA mode', async () => {
    await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    expect(screen.getByLabelText('Organization (O)')).toBeInTheDocument()
    expect(screen.getByLabelText('Organizational Unit (OU)')).toBeInTheDocument()
    expect(screen.getByLabelText('Country (C)')).toBeInTheDocument()
    expect(screen.getByLabelText('State / Province (ST)')).toBeInTheDocument()
    expect(screen.getByLabelText('City / Locality (L)')).toBeInTheDocument()
  })

  it('hides subject fields in import mode', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    expect(screen.getByLabelText('Organization (O)')).toBeInTheDocument()

    await user.click(screen.getByText('Import PEM'))

    expect(screen.queryByLabelText('Organization (O)')).not.toBeInTheDocument()
  })

  it('submits subject fields when filled in local CA mode', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'My Cert')
    await user.type(screen.getByLabelText('Domain'), 'example.com')
    await user.type(screen.getByLabelText('Organization (O)'), 'Acme Corp')
    await user.type(screen.getByLabelText('Organizational Unit (OU)'), 'Engineering')
    await user.type(screen.getByLabelText('Country (C)'), 'US')
    await user.type(screen.getByLabelText('State / Province (ST)'), 'California')
    await user.type(screen.getByLabelText('City / Locality (L)'), 'San Francisco')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'My Cert',
      domain: 'example.com',
      organization: 'Acme Corp',
      organizational_unit: 'Engineering',
      country: 'US',
      province: 'California',
      locality: 'San Francisco',
    })
  })

  it('omits empty subject fields from submission', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <CertificateForm
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />
    )

    await user.type(screen.getByLabelText('Certificate Name'), 'My Cert')
    await user.type(screen.getByLabelText('Domain'), 'example.com')
    await user.type(screen.getByLabelText('Organization (O)'), 'Acme Corp')
    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    expect(onSubmit).toHaveBeenCalledWith({
      name: 'My Cert',
      domain: 'example.com',
      organization: 'Acme Corp',
    })
  })
})
