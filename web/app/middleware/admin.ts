export default defineNuxtRouteMiddleware(async () => {
  const adminStore = useAdminStore()
  const notificationsStore = useNotificationsStore()
  if (await adminStore.checkAccess(true)) return

  notificationsStore.addNotification({
    type: 'error',
    message: 'You do not have administrator access.',
  })
  return navigateTo('/threads')
})
