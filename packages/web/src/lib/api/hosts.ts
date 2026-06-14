import { request } from './client'
import type {
  HostEntry,
  HostEntryInput,
  HostEntryReorderInput,
  HostProfile,
  HostProfileInput,
  HostProfileListResponse,
} from './types'

export const hostsApi = {
  list: () => request<HostProfileListResponse>('/host-profiles'),
  get: (id: string) => request<HostProfile>(`/host-profiles/${id}`),
  create: (data: HostProfileInput) =>
    request<HostProfile>('/host-profiles', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: HostProfileInput) =>
    request<HostProfile>(`/host-profiles/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<{ status: string }>(`/host-profiles/${id}`, { method: 'DELETE' }),
  activate: (id: string) =>
    request<HostProfile>(`/host-profiles/${id}/activate`, { method: 'POST' }),
  listEntries: (profileId: string) =>
    request<HostEntry[]>(`/host-profiles/${profileId}/entries`),
  createEntry: (profileId: string, data: HostEntryInput) =>
    request<HostEntry>(`/host-profiles/${profileId}/entries`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateEntry: (profileId: string, entryId: string, data: HostEntryInput) =>
    request<HostEntry>(`/host-profiles/${profileId}/entries/${entryId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  deleteEntry: (profileId: string, entryId: string) =>
    request<{ status: string }>(`/host-profiles/${profileId}/entries/${entryId}`, {
      method: 'DELETE',
    }),
  reorderEntries: (profileId: string, data: HostEntryReorderInput) =>
    request<{ status: string }>(`/host-profiles/${profileId}/entries/reorder`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
}
