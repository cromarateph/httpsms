<script setup lang="ts">
import {
  mdiAccountEdit,
  mdiAccountPlus,
  mdiArrowLeft,
  mdiDeleteOutline,
  mdiDownloadOutline,
  mdiKeyVariant,
  mdiMagnify,
  mdiRefresh,
  mdiShieldAccount,
} from '@mdi/js'
import { Line } from 'vue-chartjs'
import type { ChartData, ChartOptions } from 'chart.js'
import type {
  AdminAccountPayload,
  AdminDailyUsage,
  AdminReportSummary,
  AdminUser,
} from '~/stores/admin'
import { toApiError } from '~/utils/api-error'

definePageMeta({
  middleware: ['auth', 'admin'],
})

useHead({
  title: 'Admin Portal - EvilMachine SMS',
})

const adminStore = useAdminStore()
const notificationsStore = useNotificationsStore()
const tab = ref('overview')
const loading = ref(true)
const saving = ref(false)
const search = ref('')
const page = ref(1)
const itemsPerPage = ref(25)
const accountDialog = ref(false)
const editingUserId = ref<string | null>(null)
const formErrors = ref(new Map<string, string[]>())

const today = new Date()
const thirtyDaysAgo = new Date(today)
thirtyDaysAgo.setUTCDate(thirtyDaysAgo.getUTCDate() - 29)
const reportFrom = ref(thirtyDaysAgo.toISOString().slice(0, 10))
const reportTo = ref(today.toISOString().slice(0, 10))

const accountForm = reactive<AdminAccountPayload>({
  email: '',
  password: '',
  timezone: 'Asia/Manila',
  subscription_name: 'free',
  notification_message_status_enabled: true,
  notification_webhook_enabled: true,
  notification_heartbeat_enabled: true,
  notification_newsletter_enabled: true,
})

const planOptions = [
  { title: 'Internal Pro · 5,000 messages', value: 'free' },
  { title: 'Ultra · 10,000 messages', value: 'ultra-monthly' },
  { title: '20K · 20,000 messages', value: '20k-monthly' },
  { title: '50K · 50,000 messages', value: '50k-monthly' },
  { title: '100K · 100,000 messages', value: '100k-monthly' },
  { title: '200K · 200,000 messages', value: '200k-monthly' },
]

const accountHeaders = [
  { title: 'Account', key: 'email', sortable: false },
  { title: 'Plan', key: 'subscription_name', sortable: false },
  { title: 'Current usage', key: 'current_messages', sortable: false },
  { title: 'Phones', key: 'phone_count', sortable: false },
  { title: 'Last activity', key: 'last_message_at', sortable: false },
  { title: 'Joined', key: 'created_at', sortable: false },
  { title: '', key: 'actions', sortable: false, align: 'end' as const },
]

const auditHeaders = [
  { title: 'Timestamp', key: 'created_at' },
  { title: 'Administrator', key: 'admin_email' },
  { title: 'Action', key: 'action' },
  { title: 'Target user', key: 'target_user_id' },
  { title: 'Details', key: 'details' },
]

const numberFormatter = new Intl.NumberFormat()
const dateFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
})

function formatNumber(value: number | undefined) {
  return numberFormatter.format(value ?? 0)
}

function formatDate(value: string | null | undefined) {
  return value ? dateFormatter.format(new Date(value)) : 'No activity'
}

function formatRate(value: number | undefined) {
  return `${(value ?? 0).toFixed(1)}%`
}

function planLabel(value: string) {
  return (
    planOptions.find((plan) => plan.value === value)?.title.split(' · ')[0] ??
    value
  )
}

function statusColor(status: string) {
  if (['delivered', 'sent', 'received'].includes(status)) return 'success'
  if (['failed', 'expired', 'deleted'].includes(status)) return 'error'
  if (['pending', 'scheduled', 'sending'].includes(status)) return 'warning'
  return 'default'
}

function chartData(values: AdminDailyUsage[]): ChartData<'line'> {
  return {
    labels: values.map((value) => value.date),
    datasets: [
      {
        label: 'Sent',
        data: values.map((value) => value.sent),
        borderColor: '#42a5f5',
        backgroundColor: 'rgba(66, 165, 245, 0.16)',
        fill: true,
        tension: 0.25,
      },
      {
        label: 'Received',
        data: values.map((value) => value.received),
        borderColor: '#ff9800',
        backgroundColor: 'rgba(255, 152, 0, 0.12)',
        fill: true,
        tension: 0.25,
      },
    ],
  }
}

const chartOptions: ChartOptions<'line'> = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: {
    legend: {
      position: 'bottom',
      labels: { color: '#d8d8d8', usePointStyle: true },
    },
  },
  scales: {
    x: {
      ticks: { color: '#a9a9a9', maxTicksLimit: 8 },
      grid: { color: 'rgba(255, 255, 255, 0.06)' },
    },
    y: {
      beginAtZero: true,
      ticks: { color: '#a9a9a9', precision: 0 },
      grid: { color: 'rgba(255, 255, 255, 0.06)' },
    },
  },
}

const overviewChartData = computed(() =>
  chartData(adminStore.overview?.daily_usage ?? []),
)
const reportChartData = computed(() =>
  chartData(adminStore.report?.daily_usage ?? []),
)

async function loadAccounts() {
  await adminStore.loadUsers(
    search.value.trim(),
    itemsPerPage.value,
    (page.value - 1) * itemsPerPage.value,
  )
}

async function refreshAll() {
  loading.value = true
  try {
    await Promise.all([
      adminStore.loadOverview(),
      loadAccounts(),
      adminStore.loadAuditLogs(),
      adminStore.loadReport(reportFrom.value, reportTo.value),
    ])
  } finally {
    loading.value = false
  }
}

async function searchAccounts() {
  page.value = 1
  await loadAccounts()
}

async function selectAccount(user: { id: string }) {
  loading.value = true
  try {
    await adminStore.loadUser(user.id)
  } finally {
    loading.value = false
  }
}

async function openAccount(user: { id: string }) {
  tab.value = 'accounts'
  await selectAccount(user)
}

function resetAccountForm() {
  Object.assign(accountForm, {
    email: '',
    password: '',
    timezone: 'Asia/Manila',
    subscription_name: 'free',
    notification_message_status_enabled: true,
    notification_webhook_enabled: true,
    notification_heartbeat_enabled: true,
    notification_newsletter_enabled: true,
  })
  formErrors.value = new Map()
}

function openCreateAccount() {
  editingUserId.value = null
  resetAccountForm()
  accountDialog.value = true
}

function openEditAccount(user: AdminUser) {
  editingUserId.value = user.id
  Object.assign(accountForm, {
    email: user.email,
    password: '',
    timezone: user.timezone || 'Asia/Manila',
    subscription_name: user.subscription_name,
    notification_message_status_enabled:
      user.notification_message_status_enabled,
    notification_webhook_enabled: user.notification_webhook_enabled,
    notification_heartbeat_enabled: user.notification_heartbeat_enabled,
    notification_newsletter_enabled: user.notification_newsletter_enabled,
  })
  formErrors.value = new Map()
  accountDialog.value = true
}

async function saveAccount() {
  saving.value = true
  formErrors.value = new Map()
  try {
    if (editingUserId.value) {
      await adminStore.updateUser(editingUserId.value, accountForm)
      notificationsStore.addNotification({
        type: 'success',
        message: 'Account updated successfully.',
      })
    } else {
      await adminStore.createUser(accountForm)
      notificationsStore.addNotification({
        type: 'success',
        message: 'Account created successfully.',
      })
    }
    accountDialog.value = false
    await Promise.all([
      loadAccounts(),
      adminStore.loadOverview(),
      adminStore.loadAuditLogs(),
    ])
  } catch (error) {
    const apiError = toApiError(error)
    formErrors.value = new Map(
      Object.entries(apiError.data?.data ?? {}).map(([key, values]) => [
        key,
        values,
      ]),
    )
    notificationsStore.addNotification({
      type: 'error',
      message: apiError.data?.message ?? 'The account could not be saved.',
    })
  } finally {
    saving.value = false
  }
}

async function deleteAccount(user: AdminUser) {
  if (
    !window.confirm(
      `Delete ${user.email}? This permanently removes the account and schedules cleanup of its messages, phones, and integrations.`,
    )
  ) {
    return
  }

  loading.value = true
  try {
    await adminStore.deleteUser(user.id)
    notificationsStore.addNotification({
      type: 'success',
      message: 'Account deleted successfully.',
    })
    await Promise.all([
      loadAccounts(),
      adminStore.loadOverview(),
      adminStore.loadAuditLogs(),
    ])
  } catch (error) {
    notificationsStore.addNotification({
      type: 'error',
      message:
        toApiError(error).data?.message ?? 'The account could not be deleted.',
    })
  } finally {
    loading.value = false
  }
}

async function rotateAPIKey(user: AdminUser) {
  if (
    !window.confirm(
      `Rotate the API key for ${user.email}? Existing integrations using the current key will stop working.`,
    )
  ) {
    return
  }

  loading.value = true
  try {
    await adminStore.rotateAPIKey(user.id)
    notificationsStore.addNotification({
      type: 'success',
      message: 'API key rotated successfully.',
    })
    await adminStore.loadAuditLogs()
  } catch {
    notificationsStore.addNotification({
      type: 'error',
      message: 'The API key could not be rotated.',
    })
  } finally {
    loading.value = false
  }
}

async function generateReport() {
  loading.value = true
  try {
    await adminStore.loadReport(reportFrom.value, reportTo.value)
  } finally {
    loading.value = false
  }
}

function csvCell(value: string | number) {
  let text = String(value)
  if (/^[=+\-@]/.test(text)) text = `'${text}`
  return `"${text.replaceAll('"', '""')}"`
}

function exportReport() {
  const report = adminStore.report
  if (!report) return

  const rows: Array<Array<string | number>> = [
    ['EvilMachine SMS report', `${report.from} to ${report.to}`],
    [],
    ['Metric', 'Value'],
    ['Total messages', report.summary.total_messages],
    ['Sent messages', report.summary.sent_messages],
    ['Received messages', report.summary.received_messages],
    ['Delivered messages', report.summary.delivered_messages],
    ['Failed messages', report.summary.failed_messages],
    ['Expired messages', report.summary.expired_messages],
    ['Active users', report.summary.active_users],
    ['New users', report.summary.new_users],
    ['Delivery rate', `${report.summary.delivery_rate.toFixed(1)}%`],
    [],
    ['Date', 'Sent', 'Received', 'Total'],
    ...report.daily_usage.map((day) => [
      day.date,
      day.sent,
      day.received,
      day.total,
    ]),
    [],
    ['Top account', 'Sent', 'Received', 'Total'],
    ...report.top_users.map((user) => [
      user.email,
      user.sent,
      user.received,
      user.total,
    ]),
  ]
  const csv = rows.map((row) => row.map(csvCell).join(',')).join('\n')
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }))
  const link = document.createElement('a')
  link.href = url
  link.download = `sms-report-${report.from}-${report.to}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

function summaryMetrics(summary: AdminReportSummary | undefined) {
  return [
    ['Total messages', formatNumber(summary?.total_messages)],
    ['Sent', formatNumber(summary?.sent_messages)],
    ['Received', formatNumber(summary?.received_messages)],
    ['Active users', formatNumber(summary?.active_users)],
    ['New users', formatNumber(summary?.new_users)],
    ['Delivery rate', formatRate(summary?.delivery_rate)],
  ]
}

watch([page, itemsPerPage], loadAccounts)

onMounted(refreshAll)
</script>

<template>
  <VContainer fluid class="admin-shell px-0 pt-0">
    <VAppBar>
      <VBtn icon to="/threads" aria-label="Return to messages">
        <VIcon :icon="mdiArrowLeft" />
      </VBtn>
      <VToolbarTitle class="d-flex align-center">
        Admin Portal
        <VChip class="ml-3" size="small" color="primary" variant="tonal">
          Internal
        </VChip>
      </VToolbarTitle>
      <VBtn
        :icon="mdiRefresh"
        :loading="loading"
        aria-label="Refresh admin data"
        @click="refreshAll"
      />
      <VProgressLinear
        :active="loading"
        :indeterminate="loading"
        color="primary"
        absolute
        location="bottom"
      />
    </VAppBar>

    <VContainer class="admin-content">
      <div class="d-flex flex-wrap align-end justify-space-between ga-4 mb-4">
        <div>
          <h1 class="text-headline-large mb-1">Operations</h1>
          <p class="text-medium-emphasis mb-0">
            Accounts, platform usage, delivery health, and administrator
            activity.
          </p>
        </div>
        <VBtn
          color="primary"
          :prepend-icon="mdiAccountPlus"
          @click="openCreateAccount"
        >
          Create account
        </VBtn>
      </div>

      <VTabs v-model="tab" color="primary" class="admin-tabs mb-5">
        <VTab value="overview">Overview</VTab>
        <VTab value="accounts">Accounts</VTab>
        <VTab value="reports">Reports</VTab>
        <VTab value="activity">Activity</VTab>
      </VTabs>

      <VWindow v-model="tab">
        <VWindowItem value="overview">
          <VSheet class="metric-strip mb-6" border>
            <div class="metric-cell">
              <span>Total accounts</span>
              <strong>{{
                formatNumber(adminStore.overview?.total_users)
              }}</strong>
            </div>
            <div class="metric-cell">
              <span>New · 30 days</span>
              <strong>{{
                formatNumber(adminStore.overview?.new_users_30_days)
              }}</strong>
            </div>
            <div class="metric-cell">
              <span>Active · 30 days</span>
              <strong>{{
                formatNumber(adminStore.overview?.active_users_30_days)
              }}</strong>
            </div>
            <div class="metric-cell">
              <span>Connected phones</span>
              <strong>{{
                formatNumber(adminStore.overview?.connected_phones)
              }}</strong>
            </div>
            <div class="metric-cell">
              <span>All-time usage</span>
              <strong>{{
                formatNumber(adminStore.overview?.total_messages)
              }}</strong>
            </div>
            <div class="metric-cell">
              <span>Delivery · 30 days</span>
              <strong>{{
                formatRate(adminStore.overview?.delivery_rate_30_days)
              }}</strong>
            </div>
          </VSheet>

          <VRow>
            <VCol cols="12" lg="8">
              <VSheet class="section-sheet h-100" border>
                <div class="section-heading">
                  <div>
                    <h2 class="text-headline-small mb-1">Message activity</h2>
                    <p class="text-medium-emphasis mb-0">Last 30 UTC days</p>
                  </div>
                  <span class="text-body-large font-weight-medium">
                    {{ formatNumber(adminStore.overview?.messages_30_days) }}
                    total
                  </span>
                </div>
                <div class="admin-chart">
                  <ClientOnly>
                    <Line :data="overviewChartData" :options="chartOptions" />
                  </ClientOnly>
                </div>
              </VSheet>
            </VCol>
            <VCol cols="12" lg="4">
              <VSheet class="section-sheet h-100" border>
                <div class="section-heading">
                  <div>
                    <h2 class="text-headline-small mb-1">Delivery status</h2>
                    <p class="text-medium-emphasis mb-0">Last 30 UTC days</p>
                  </div>
                </div>
                <VList lines="one" bg-color="transparent">
                  <VListItem
                    v-for="status in adminStore.overview?.status_breakdown"
                    :key="status.name"
                  >
                    <template #prepend>
                      <VChip
                        :color="statusColor(status.name)"
                        size="small"
                        variant="tonal"
                        class="mr-3 text-capitalize"
                      >
                        {{ status.name }}
                      </VChip>
                    </template>
                    <template #append>
                      <strong>{{ formatNumber(status.count) }}</strong>
                    </template>
                  </VListItem>
                </VList>
                <VAlert
                  v-if="!adminStore.overview?.status_breakdown.length"
                  type="info"
                  variant="tonal"
                  text="No message activity in this period."
                />
              </VSheet>
            </VCol>
          </VRow>

          <VSheet class="section-sheet mt-6" border>
            <div class="section-heading">
              <div>
                <h2 class="text-headline-small mb-1">Most active accounts</h2>
                <p class="text-medium-emphasis mb-0">
                  Message records from the last 30 days
                </p>
              </div>
            </div>
            <VTable density="comfortable">
              <thead>
                <tr>
                  <th>Account</th>
                  <th class="text-right">Sent</th>
                  <th class="text-right">Received</th>
                  <th class="text-right">Total</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="user in adminStore.overview?.top_users"
                  :key="user.id"
                >
                  <td>
                    <button class="account-link" @click="openAccount(user)">
                      {{ user.email }}
                    </button>
                  </td>
                  <td class="text-right">{{ formatNumber(user.sent) }}</td>
                  <td class="text-right">{{ formatNumber(user.received) }}</td>
                  <td class="text-right font-weight-bold">
                    {{ formatNumber(user.total) }}
                  </td>
                </tr>
              </tbody>
            </VTable>
          </VSheet>
        </VWindowItem>

        <VWindowItem value="accounts">
          <div class="d-flex flex-wrap ga-3 align-center mb-4">
            <VTextField
              v-model="search"
              :prepend-inner-icon="mdiMagnify"
              label="Search by email or user ID"
              variant="outlined"
              density="compact"
              hide-details
              class="account-search"
              clearable
              @keyup.enter="searchAccounts"
            />
            <VBtn variant="tonal" @click="searchAccounts">Search</VBtn>
          </div>

          <VSheet
            v-if="adminStore.selectedUser"
            class="account-detail mb-5"
            border
          >
            <div class="d-flex flex-wrap justify-space-between ga-4">
              <div>
                <div class="d-flex align-center ga-2 mb-1">
                  <VIcon color="primary" :icon="mdiShieldAccount" />
                  <h2 class="text-headline-small">
                    {{ adminStore.selectedUser.user.email }}
                  </h2>
                </div>
                <p class="text-medium-emphasis mb-0 text-break">
                  {{ adminStore.selectedUser.user.id }}
                </p>
              </div>
              <div class="d-flex flex-wrap ga-2 align-start">
                <VBtn
                  variant="tonal"
                  :prepend-icon="mdiAccountEdit"
                  @click="openEditAccount(adminStore.selectedUser.user)"
                >
                  Edit
                </VBtn>
                <VBtn
                  variant="tonal"
                  :prepend-icon="mdiKeyVariant"
                  @click="rotateAPIKey(adminStore.selectedUser.user)"
                >
                  Rotate API key
                </VBtn>
                <VBtn
                  color="error"
                  variant="tonal"
                  :prepend-icon="mdiDeleteOutline"
                  @click="deleteAccount(adminStore.selectedUser.user)"
                >
                  Delete
                </VBtn>
              </div>
            </div>

            <div class="detail-grid mt-5">
              <div>
                <span>Plan</span>
                <strong>{{
                  planLabel(adminStore.selectedUser.user.subscription_name)
                }}</strong>
              </div>
              <div>
                <span>Current usage</span>
                <strong>
                  {{
                    formatNumber(adminStore.selectedUser.user.current_messages)
                  }}
                  /
                  {{
                    formatNumber(
                      adminStore.selectedUser.user.subscription_limit,
                    )
                  }}
                </strong>
              </div>
              <div>
                <span>All-time usage</span>
                <strong>{{
                  formatNumber(
                    adminStore.selectedUser.user.sent_messages +
                      adminStore.selectedUser.user.received_messages,
                  )
                }}</strong>
              </div>
              <div>
                <span>Phones</span>
                <strong>{{ adminStore.selectedUser.user.phone_count }}</strong>
              </div>
              <div>
                <span>Threads</span>
                <strong>{{ adminStore.selectedUser.user.thread_count }}</strong>
              </div>
              <div>
                <span>Webhooks</span>
                <strong>{{
                  adminStore.selectedUser.user.webhook_count
                }}</strong>
              </div>
              <div>
                <span>Timezone</span>
                <strong>{{ adminStore.selectedUser.user.timezone }}</strong>
              </div>
              <div>
                <span>Last activity</span>
                <strong>{{
                  formatDate(adminStore.selectedUser.user.last_message_at)
                }}</strong>
              </div>
            </div>

            <div class="mt-5">
              <span class="text-medium-emphasis mr-3">Notifications</span>
              <VChip
                v-if="
                  adminStore.selectedUser.user
                    .notification_message_status_enabled
                "
                size="small"
                class="mr-2 mb-2"
              >
                Message status
              </VChip>
              <VChip
                v-if="adminStore.selectedUser.user.notification_webhook_enabled"
                size="small"
                class="mr-2 mb-2"
              >
                Webhook
              </VChip>
              <VChip
                v-if="
                  adminStore.selectedUser.user.notification_heartbeat_enabled
                "
                size="small"
                class="mr-2 mb-2"
              >
                Heartbeat
              </VChip>
              <VChip
                v-if="
                  adminStore.selectedUser.user.notification_newsletter_enabled
                "
                size="small"
                class="mr-2 mb-2"
              >
                Newsletter
              </VChip>
            </div>
          </VSheet>

          <VSheet class="section-sheet pa-0" border>
            <VDataTableServer
              v-model:page="page"
              v-model:items-per-page="itemsPerPage"
              :headers="accountHeaders"
              :items="adminStore.users"
              :items-length="adminStore.totalUsers"
              :loading="loading"
              item-value="id"
              hover
            >
              <template #[`item.email`]="{ item }">
                <button
                  class="account-link text-left"
                  @click="selectAccount(item)"
                >
                  <strong>{{ item.email }}</strong>
                  <small class="d-block text-medium-emphasis">{{
                    item.id
                  }}</small>
                </button>
              </template>
              <template #[`item.subscription_name`]="{ item }">
                <VChip size="small" color="primary" variant="tonal">
                  {{ planLabel(item.subscription_name) }}
                </VChip>
              </template>
              <template #[`item.current_messages`]="{ item }">
                {{ formatNumber(item.current_messages) }} /
                {{ formatNumber(item.subscription_limit) }}
              </template>
              <template #[`item.last_message_at`]="{ item }">
                {{ formatDate(item.last_message_at) }}
              </template>
              <template #[`item.created_at`]="{ item }">
                {{ formatDate(item.created_at) }}
              </template>
              <template #[`item.actions`]="{ item }">
                <div class="d-flex justify-end">
                  <VBtn
                    icon
                    variant="text"
                    size="small"
                    :aria-label="`Edit ${item.email}`"
                    @click="openEditAccount(item)"
                  >
                    <VIcon :icon="mdiAccountEdit" />
                  </VBtn>
                  <VBtn
                    icon
                    variant="text"
                    size="small"
                    color="error"
                    :aria-label="`Delete ${item.email}`"
                    @click="deleteAccount(item)"
                  >
                    <VIcon :icon="mdiDeleteOutline" />
                  </VBtn>
                </div>
              </template>
            </VDataTableServer>
          </VSheet>
        </VWindowItem>

        <VWindowItem value="reports">
          <VSheet class="section-sheet mb-6" border>
            <div class="d-flex flex-wrap align-end ga-3">
              <VTextField
                v-model="reportFrom"
                type="date"
                label="Start date"
                variant="outlined"
                density="compact"
                hide-details
                class="date-field"
              />
              <VTextField
                v-model="reportTo"
                type="date"
                label="End date"
                variant="outlined"
                density="compact"
                hide-details
                class="date-field"
              />
              <VBtn color="primary" @click="generateReport">
                Generate report
              </VBtn>
              <VBtn
                variant="tonal"
                :prepend-icon="mdiDownloadOutline"
                :disabled="!adminStore.report"
                @click="exportReport"
              >
                Export CSV
              </VBtn>
            </div>
          </VSheet>

          <VSheet class="metric-strip mb-6" border>
            <div
              v-for="[label, value] in summaryMetrics(
                adminStore.report?.summary,
              )"
              :key="label"
              class="metric-cell"
            >
              <span>{{ label }}</span>
              <strong>{{ value }}</strong>
            </div>
          </VSheet>

          <VRow>
            <VCol cols="12" lg="8">
              <VSheet class="section-sheet h-100" border>
                <div class="section-heading">
                  <div>
                    <h2 class="text-headline-small mb-1">Usage over time</h2>
                    <p class="text-medium-emphasis mb-0">
                      {{ adminStore.report?.from }} to
                      {{ adminStore.report?.to }}
                    </p>
                  </div>
                </div>
                <div class="admin-chart">
                  <ClientOnly>
                    <Line :data="reportChartData" :options="chartOptions" />
                  </ClientOnly>
                </div>
              </VSheet>
            </VCol>
            <VCol cols="12" lg="4">
              <VSheet class="section-sheet h-100" border>
                <div class="section-heading">
                  <h2 class="text-headline-small">Status breakdown</h2>
                </div>
                <VList bg-color="transparent">
                  <VListItem
                    v-for="status in adminStore.report?.status_breakdown"
                    :key="status.name"
                  >
                    <template #prepend>
                      <VChip
                        :color="statusColor(status.name)"
                        size="small"
                        variant="tonal"
                        class="mr-3 text-capitalize"
                      >
                        {{ status.name }}
                      </VChip>
                    </template>
                    <template #append>
                      <strong>{{ formatNumber(status.count) }}</strong>
                    </template>
                  </VListItem>
                </VList>
              </VSheet>
            </VCol>
          </VRow>

          <VSheet class="section-sheet mt-6" border>
            <div class="section-heading">
              <h2 class="text-headline-small">Top accounts in this report</h2>
            </div>
            <VTable>
              <thead>
                <tr>
                  <th>Account</th>
                  <th class="text-right">Sent</th>
                  <th class="text-right">Received</th>
                  <th class="text-right">Total</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="user in adminStore.report?.top_users" :key="user.id">
                  <td>{{ user.email }}</td>
                  <td class="text-right">{{ formatNumber(user.sent) }}</td>
                  <td class="text-right">{{ formatNumber(user.received) }}</td>
                  <td class="text-right font-weight-bold">
                    {{ formatNumber(user.total) }}
                  </td>
                </tr>
              </tbody>
            </VTable>
          </VSheet>
        </VWindowItem>

        <VWindowItem value="activity">
          <VSheet class="section-sheet pa-0" border>
            <div class="section-heading">
              <div>
                <h2 class="text-headline-small mb-1">Administrator activity</h2>
                <p class="text-medium-emphasis mb-0">
                  Account changes performed through this portal
                </p>
              </div>
            </div>
            <VDataTable
              :headers="auditHeaders"
              :items="adminStore.auditLogs"
              :items-per-page="25"
              density="comfortable"
            >
              <template #[`item.created_at`]="{ item }">
                {{ formatDate(item.created_at) }}
              </template>
              <template #[`item.action`]="{ item }">
                <VChip size="small" variant="tonal">
                  {{ item.action.replaceAll('.', ' ') }}
                </VChip>
              </template>
              <template #[`item.target_user_id`]="{ item }">
                <code>{{ item.target_user_id }}</code>
              </template>
            </VDataTable>
          </VSheet>
        </VWindowItem>
      </VWindow>
    </VContainer>

    <VDialog v-model="accountDialog" max-width="680">
      <VCard>
        <VCardTitle class="text-headline-small pa-6 pb-2">
          {{ editingUserId ? 'Edit account' : 'Create account' }}
        </VCardTitle>
        <VCardSubtitle class="px-6">
          {{
            editingUserId
              ? 'Changes apply immediately to this account.'
              : 'Creates both Firebase login and application records.'
          }}
        </VCardSubtitle>
        <VCardText class="pa-6">
          <VTextField
            v-model="accountForm.email"
            type="email"
            label="Email"
            variant="outlined"
            :disabled="saving"
            :error-messages="formErrors.get('email')"
          />
          <VTextField
            v-model="accountForm.password"
            type="password"
            :label="
              editingUserId
                ? 'New password (leave blank to keep current)'
                : 'Temporary password'
            "
            variant="outlined"
            :disabled="saving"
            :error-messages="formErrors.get('password')"
          />
          <VRow>
            <VCol cols="12" md="6">
              <VTextField
                v-model="accountForm.timezone"
                label="IANA timezone"
                placeholder="Asia/Manila"
                variant="outlined"
                :disabled="saving"
                :error-messages="formErrors.get('timezone')"
              />
            </VCol>
            <VCol cols="12" md="6">
              <VSelect
                v-model="accountForm.subscription_name"
                label="Account limit"
                :items="planOptions"
                variant="outlined"
                :disabled="saving"
                :error-messages="formErrors.get('subscription_name')"
              />
            </VCol>
          </VRow>

          <template v-if="editingUserId">
            <p class="text-subtitle-1 font-weight-medium mb-2">
              Email notifications
            </p>
            <VCheckbox
              v-model="accountForm.notification_message_status_enabled"
              label="Message status"
              density="compact"
              hide-details
            />
            <VCheckbox
              v-model="accountForm.notification_webhook_enabled"
              label="Webhook failures"
              density="compact"
              hide-details
            />
            <VCheckbox
              v-model="accountForm.notification_heartbeat_enabled"
              label="Phone heartbeat"
              density="compact"
              hide-details
            />
            <VCheckbox
              v-model="accountForm.notification_newsletter_enabled"
              label="Newsletter"
              density="compact"
              hide-details
            />
          </template>
        </VCardText>
        <VCardActions class="px-6 pb-6">
          <VBtn
            variant="text"
            :disabled="saving"
            @click="accountDialog = false"
          >
            Cancel
          </VBtn>
          <VSpacer />
          <VBtn color="primary" :loading="saving" @click="saveAccount">
            {{ editingUserId ? 'Save changes' : 'Create account' }}
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </VContainer>
</template>

<style scoped>
.admin-content {
  max-width: 1500px;
}

.admin-tabs {
  border-bottom: 1px solid rgb(var(--v-border-color), var(--v-border-opacity));
}

.metric-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  border-radius: 12px;
  overflow: hidden;
}

.metric-cell {
  min-height: 104px;
  padding: 20px;
  border-right: 1px solid rgb(var(--v-border-color), var(--v-border-opacity));
}

.metric-cell:last-child {
  border-right: 0;
}

.metric-cell span,
.detail-grid span {
  display: block;
  color: rgb(var(--v-theme-on-surface), 0.72);
  font-size: 0.875rem;
}

.metric-cell strong {
  display: block;
  margin-top: 8px;
  font-size: 1.65rem;
  line-height: 1.15;
}

.section-sheet,
.account-detail {
  border-radius: 12px;
  padding: 24px;
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.admin-chart {
  height: 320px;
}

.account-search {
  max-width: 520px;
}

.account-link {
  color: rgb(var(--v-theme-primary));
  cursor: pointer;
  font: inherit;
  background: transparent;
  border: 0;
}

.account-link:hover,
.account-link:focus-visible {
  text-decoration: underline;
}

.account-link:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 3px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 20px;
}

.detail-grid strong {
  display: block;
  margin-top: 5px;
}

.date-field {
  max-width: 220px;
}

@media (width <= 700px) {
  .admin-content {
    padding-inline: 12px;
  }

  .metric-cell {
    min-height: 88px;
    padding: 16px;
    border-bottom: 1px solid rgb(var(--v-border-color), var(--v-border-opacity));
  }

  .section-sheet,
  .account-detail {
    padding: 16px;
  }

  .admin-chart {
    height: 260px;
  }

  .date-field,
  .account-search {
    max-width: none;
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
