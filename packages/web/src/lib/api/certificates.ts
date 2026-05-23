import { listResource, request } from './client'
import type { Certificate, CertificateInput } from './types'

export const certificatesApi = {
  list: () => listResource<Certificate>('/certificates'),
  get: (id: string) => request<Certificate>(`/certificates/${id}`),
  create: (data: CertificateInput) => request<Certificate>('/certificates', { method: 'POST', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/certificates/${id}`, { method: 'DELETE' }),
  renew: (id: string) => request<void>(`/certificates/${id}/renew`, { method: 'POST' }),
}