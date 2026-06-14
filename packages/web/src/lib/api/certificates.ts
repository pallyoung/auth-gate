import { listResource, request } from './client'
import type { Certificate, CertificateInput } from './types'

interface CAExport {
  cert_pem: string
  name: string
  not_after: string
}

export const certificatesApi = {
  list: () => listResource<Certificate>('/certificates'),
  get: (id: string) => request<Certificate>(`/certificates/${id}`),
  create: (data: CertificateInput) => request<Certificate>('/certificates', { method: 'POST', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/certificates/${id}`, { method: 'DELETE' }),
  resign: (id: string) => request<void>(`/certificates/${id}/resign`, { method: 'POST' }),
  getCA: () => request<CAExport>('/ca'),
}
