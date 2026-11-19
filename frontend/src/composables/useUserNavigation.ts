import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { useAuthStore } from '@/stores/auth'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

type UserNavigationIcon =
  | 'home'
  | 'key'
  | 'chart'
  | 'play'
  | 'chat'
  | 'server'
  | 'bolt'
  | 'sparkles'
  | 'cube'
  | 'creditCard'
  | 'document'
  | 'gift'
  | 'users'
  | 'user'
  | 'link'
  | 'mcp'

type UserNavigationGroup = 'primary' | 'models' | 'billing' | 'tools' | 'custom'

export interface UserNavigationItem {
  path: string
  label: string
  icon: UserNavigationIcon
  group: UserNavigationGroup
  activePaths?: string[]
  href?: string
  newTab?: boolean
  query?: Record<string, string>
  action?: 'helios'
  iconSvg?: string
  hideInSimpleMode?: boolean
  enabled?: boolean
}

/** Personal routes organized into 4 focused M3 domains while preserving all legacy URLs. */
export function useUserNavigation() {
  const { t, locale } = useI18n()
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
  const zh = computed(() => locale.value.startsWith('zh'))
  const settings = computed(() => appStore.cachedPublicSettings)
  const purchaseLabel = computed(() => (zh.value ? '充值与兑换' : 'Top up & redeem'))
  const subscriptions = useSubscriptionStore()

  watch(
    () => authStore.isAuthenticated && resolveFeatureFlag(settings.value, FeatureFlags.subscription),
    (enabled) => {
      if (enabled) void subscriptions.fetchAvailability().catch(() => undefined)
    },
    { immediate: true }
  )

  const subscriptionVisible = computed(() => {
    const state = subscriptions.availability
    return (
      !state ||
      subscriptions.availabilityError ||
      state.has_sale_plans ||
      state.has_active_subscriptions ||
      state.has_subscription_history
    )
  })

  const subscriptionLabel = computed(() =>
    subscriptions.availability && !subscriptions.availability.has_sale_plans
      ? zh.value
        ? '我的订阅'
        : 'My subscriptions'
      : zh.value
        ? '订阅套餐'
        : 'Subscriptions'
  )

  const items = computed<UserNavigationItem[]>(() => {
    if (!authStore.isAuthenticated || (appStore.backendModeEnabled && !authStore.isAdmin)) return []

    const enabled = (flag: keyof typeof FeatureFlags) => resolveFeatureFlag(settings.value, FeatureFlags[flag])
    const entries: UserNavigationItem[] = [
      // Domain 1: Core Workspace & API
      { path: '/dashboard', label: zh.value ? '工作台' : 'Overview', icon: 'home', group: 'primary' },
      { path: '/keys', label: zh.value ? 'API 密钥' : 'API keys', icon: 'key', group: 'primary' },
      {
        path: '/usage',
        activePaths: ['/usage', '/traffic'],
        label: zh.value ? '用量与流量' : 'Usage & traffic',
        icon: 'chart',
        group: 'primary',
        hideInSimpleMode: true
      },
      {
        path: '/playground',
        label: zh.value ? 'API 调试台' : 'Playground',
        icon: 'play',
        group: 'primary'
      },

      // Domain 2: Models & Observability
      {
        path: '/model-plaza',
        activePaths: ['/model-plaza', '/available-channels'],
        query: { embedded: '1' },
        label: t('nav.modelPlaza'),
        icon: 'cube',
        group: 'models',
        enabled: enabled('modelPlaza')
      },
      {
        path: '/monitor',
        label: t('nav.channelStatus'),
        icon: 'bolt',
        group: 'models',
        enabled: enabled('channelMonitor')
      },

      // Domain 3: Billing & Account
      // Card redemption, not a payment gateway. Self-serve checkout is retired.
      {
        path: '/purchase',
        activePaths: ['/purchase', '/redeem', '/orders'],
        label: purchaseLabel.value,
        icon: 'creditCard',
        group: 'billing',
        hideInSimpleMode: true
      },
      {
        path: '/subscriptions',
        label: subscriptionLabel.value,
        icon: 'document',
        group: 'billing',
        hideInSimpleMode: true,
        enabled: enabled('subscription') && subscriptionVisible.value
      },
      {
        path: '/affiliate',
        label: t('nav.affiliate'),
        icon: 'users',
        group: 'billing',
        hideInSimpleMode: true,
        enabled: enabled('affiliate')
      },
      { path: '/profile', label: zh.value ? '账户与安全' : 'Account', icon: 'user', group: 'billing' },

      // Domain 4: AI Apps & Integrations
      { path: '/chat', href: '/chat', newTab: true, label: zh.value ? '站内对话' : 'Chat', icon: 'chat', group: 'tools' },
      { path: '/chat-models', label: zh.value ? '聊天模型' : 'Chat models', icon: 'cube', group: 'tools' },
      { path: '/image2', href: '/image2/', label: t('nav.image2'), icon: 'sparkles', group: 'tools', hideInSimpleMode: true },
      { path: '/helios', action: 'helios', label: t('nav.infiniteCanvas'), icon: 'cube', group: 'tools', hideInSimpleMode: true },
      {
        path: '/batch-image',
        label: t('nav.batchImage'),
        icon: 'sparkles',
        group: 'tools',
        hideInSimpleMode: true,
        enabled: canUseBatchImage.value
      },
      { path: '/settings/mcp', label: t('mcp.title'), icon: 'mcp', group: 'tools', enabled: enabled('mcp') },

      // Custom user menu items (preserve card-shop deduplication filter)
      ...(settings.value?.custom_menu_items ?? [])
        .filter((item) => item.visibility === 'user' && !/云猫|卡网|catfk/i.test(item.label))
        .sort((a, b) => a.sort_order - b.sort_order)
        .map(
          (item): UserNavigationItem => ({
            path: '/custom/' + item.id,
            label: item.label,
            icon: 'link',
            iconSvg: item.icon_svg,
            group: 'custom'
          })
        )
    ]
    return entries.filter((item) => item.enabled !== false && !(authStore.isSimpleMode && item.hideInSimpleMode))
  })

  const primaryItems = computed(() => items.value.filter((item) => item.group === 'primary'))
  const secondarySections = computed(() => {
    const groups: { id: UserNavigationGroup; label: string }[] = [
      { id: 'models', label: zh.value ? '模型与观测' : 'Models & health' },
      { id: 'billing', label: zh.value ? '账单与账户' : 'Billing & account' },
      { id: 'tools', label: zh.value ? 'AI 应用与工具' : 'AI apps & tools' },
      { id: 'custom', label: zh.value ? '资源' : 'Resources' }
    ]
    return groups
      .map((group) => ({ ...group, items: items.value.filter((item) => item.group === group.id) }))
      .filter((group) => group.items.length > 0)
  })

  return { items, primaryItems, secondarySections, purchaseLabel, refreshBatchImageAccess }
}
