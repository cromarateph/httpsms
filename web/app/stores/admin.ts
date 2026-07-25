import { defineStore } from 'pinia'

export interface AdminUser {
  id: string
  email: string
  timezone: string
  active_phone_id: string | null
  subscription_name: string
  subscription_limit: number
  notification_message_status_enabled: boolean
  notification_webhook_enabled: boolean
  notification_heartbeat_enabled: boolean
  notification_newsletter_enabled: boolean
  sent_messages: number
  received_messages: number
  current_messages: number
  phone_count: number
  thread_count: number
  webhook_count: number
  last_message_at: string | null
  created_at: string
  updated_at: string
}

export interface AdminCount {
  name: string
  count: number
}

export interface AdminDailyUsage {
  date: string
  sent: number
  received: number
  total: number
}

export interface AdminTopUser {
  id: string
  email: string
  sent: number
  received: number
  total: number
}

export interface AdminReportSummary {
  total_messages: number
  sent_messages: number
  received_messages: number
  delivered_messages: number
  failed_messages: number
  expired_messages: number
  active_users: number
  new_users: number
  delivery_rate: number
}

export interface AdminReport {
  from: string
  to: string
  summary: AdminReportSummary
  daily_usage: AdminDailyUsage[]
  status_breakdown: AdminCount[]
  top_users: AdminTopUser[]
}

export interface AdminOverview {
  total_users: number
  new_users_30_days: number
  active_users_30_days: number
  connected_phones: number
  total_sent: number
  total_received: number
  total_messages: number
  messages_30_days: number
  failed_30_days: number
  delivery_rate_30_days: number
  daily_usage: AdminDailyUsage[]
  status_breakdown: AdminCount[]
  top_users: AdminTopUser[]
}

export interface AdminUserDetail {
  user: AdminUser
  usage_history: Array<{
    id: string
    sent_messages: number
    received_messages: number
    start_timestamp: string
    end_timestamp: string
  }>
  message_status: AdminCount[]
}

export interface AdminAuditLog {
  id: string
  admin_user_id: string
  admin_email: string
  action: string
  target_user_id: string
  details: string
  created_at: string
}

export interface AdminAccountPayload {
  email: string
  password: string
  timezone: string
  subscription_name: string
  notification_message_status_enabled?: boolean
  notification_webhook_enabled?: boolean
  notification_heartbeat_enabled?: boolean
  notification_newsletter_enabled?: boolean
}

export const useAdminStore = defineStore('admin', () => {
  const { apiFetch } = useApi()
  const hasAccess = ref<boolean | null>(null)
  const overview = ref<AdminOverview | null>(null)
  const users = ref<AdminUser[]>([])
  const totalUsers = ref(0)
  const selectedUser = ref<AdminUserDetail | null>(null)
  const report = ref<AdminReport | null>(null)
  const auditLogs = ref<AdminAuditLog[]>([])

  async function checkAccess(force = false): Promise<boolean> {
    if (!force && hasAccess.value !== null) return hasAccess.value
    try {
      await apiFetch('/v1/admin/access')
      hasAccess.value = true
    } catch {
      hasAccess.value = false
    }
    return hasAccess.value
  }

  async function loadOverview() {
    const response = await apiFetch<{ data: AdminOverview }>(
      '/v1/admin/overview',
    )
    overview.value = response.data
  }

  async function loadUsers(search = '', limit = 25, skip = 0) {
    const response = await apiFetch<{
      data: { users: AdminUser[]; total: number }
    }>('/v1/admin/users', {
      query: { search, limit, skip },
    })
    users.value = response.data.users
    totalUsers.value = response.data.total
  }

  async function loadUser(userId: string) {
    const response = await apiFetch<{ data: AdminUserDetail }>(
      `/v1/admin/users/${userId}`,
    )
    selectedUser.value = response.data
    return response.data
  }

  async function createUser(payload: AdminAccountPayload) {
    const response = await apiFetch<{ data: AdminUserDetail }>(
      '/v1/admin/users',
      { method: 'POST', body: payload },
    )
    selectedUser.value = response.data
    return response.data
  }

  async function updateUser(userId: string, payload: AdminAccountPayload) {
    const response = await apiFetch<{ data: AdminUserDetail }>(
      `/v1/admin/users/${userId}`,
      { method: 'PUT', body: payload },
    )
    selectedUser.value = response.data
    return response.data
  }

  async function deleteUser(userId: string) {
    await apiFetch(`/v1/admin/users/${userId}`, { method: 'DELETE' })
    if (selectedUser.value?.user.id === userId) selectedUser.value = null
  }

  async function rotateAPIKey(userId: string) {
    const response = await apiFetch<{ data: AdminUserDetail }>(
      `/v1/admin/users/${userId}/rotate-api-key`,
      { method: 'POST' },
    )
    selectedUser.value = response.data
    return response.data
  }

  async function loadReport(from: string, to: string) {
    const response = await apiFetch<{ data: AdminReport }>(
      '/v1/admin/reports',
      { query: { from, to } },
    )
    report.value = response.data
  }

  async function loadAuditLogs(limit = 100) {
    const response = await apiFetch<{ data: AdminAuditLog[] }>(
      '/v1/admin/audit-logs',
      { query: { limit } },
    )
    auditLogs.value = response.data
  }

  function resetState() {
    hasAccess.value = null
    overview.value = null
    users.value = []
    totalUsers.value = 0
    selectedUser.value = null
    report.value = null
    auditLogs.value = []
  }

  return {
    hasAccess,
    overview,
    users,
    totalUsers,
    selectedUser,
    report,
    auditLogs,
    checkAccess,
    loadOverview,
    loadUsers,
    loadUser,
    createUser,
    updateUser,
    deleteUser,
    rotateAPIKey,
    loadReport,
    loadAuditLogs,
    resetState,
  }
})
