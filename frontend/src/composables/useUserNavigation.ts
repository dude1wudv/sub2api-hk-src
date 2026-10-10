import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'

type UserNavigationIcon = 'key' | 'chart' | 'server' | 'bolt' | 'sparkles' | 'cube' | 'creditCard' | 'document' | 'gift' | 'users' | 'user' | 'link'
type UserNavigationGroup = 'primary' | 'services' | 'tools' | 'billing' | 'custom'

export interface UserNavigationItem {
  path: string
  label: string
  icon: UserNavigationIcon
  group: UserNavigationGroup
  activePaths?: string[]
  href?: string
  query?: Record<string, string>
  action?: 'helios'
  iconSvg?: string
  hideInSimpleMode?: boolean
  enabled?: boolean
}

/** Personal routes keep their original availability, independently of account role. */
export function useUserNavigation() {
  const { t, locale } = useI18n()
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
  const zh = computed(() => locale.value.startsWith('zh'))
  const settings = computed(() => appStore.cachedPublicSettings)
  const purchaseLabel = computed(() => zh.value ? '充值' : 'Recharge')

  const items = computed<UserNavigationItem[]>(() => {
    if (!authStore.isAuthenticated || (appStore.backendModeEnabled && !authStore.isAdmin)) return []

    const enabled = (flag: keyof typeof FeatureFlags) => resolveFeatureFlag(settings.value, FeatureFlags[flag])
    const entries: UserNavigationItem[] = [
      { path: '/keys', activePaths: ['/keys', '/dashboard'], label: 'API', icon: 'key', group: 'primary' },
      { path: '/usage', label: zh.value ? '用量信息' : 'Usage', icon: 'chart', group: 'primary', hideInSimpleMode: true },
      { path: '/purchase', activePaths: ['/purchase', '/redeem', '/orders'], label: purchaseLabel.value, icon: 'creditCard', group: 'primary', hideInSimpleMode: true },
      { path: '/subscriptions', label: zh.value ? '订阅' : 'Subscriptions', icon: 'document', group: 'primary', hideInSimpleMode: true, enabled: enabled('subscription') },
      { path: '/profile', label: zh.value ? '账户' : 'Account', icon: 'user', group: 'primary' },
      { path: '/model-plaza', query: { embedded: '1' }, label: t('nav.modelPlaza'), icon: 'cube', group: 'services', enabled: enabled('modelPlaza') },
      { path: '/available-channels', label: t('nav.availableChannels'), icon: 'server', group: 'services', hideInSimpleMode: true, enabled: enabled('availableChannels') },
      { path: '/monitor', label: t('nav.channelStatus'), icon: 'bolt', group: 'services', enabled: enabled('channelMonitor') },
      { path: '/image2', href: '/image2/', label: t('nav.image2'), icon: 'sparkles', group: 'tools', hideInSimpleMode: true },
      { path: '/helios', action: 'helios', label: t('nav.infiniteCanvas'), icon: 'cube', group: 'tools', hideInSimpleMode: true },
      { path: '/batch-image', label: t('nav.batchImage'), icon: 'sparkles', group: 'tools', hideInSimpleMode: true, enabled: canUseBatchImage.value },
      { path: '/affiliate', label: t('nav.affiliate'), icon: 'users', group: 'billing', hideInSimpleMode: true, enabled: enabled('affiliate') },
      ...(settings.value?.custom_menu_items ?? [])
        .filter(item => item.visibility === 'user')
        .sort((a, b) => a.sort_order - b.sort_order)
        .map((item): UserNavigationItem => ({
          path: '/custom/' + item.id,
          label: item.label,
          icon: 'link',
          iconSvg: item.icon_svg,
          group: 'custom',
        })),
    ]
    return entries.filter(item => item.enabled !== false && !(authStore.isSimpleMode && item.hideInSimpleMode))
  })

  const primaryItems = computed(() => items.value.filter(item => item.group === 'primary'))
  const moreSections = computed(() => {
    const groups: { id: UserNavigationGroup; label: string }[] = [
      { id: 'services', label: zh.value ? '服务' : 'Services' },
      { id: 'tools', label: zh.value ? '工具' : 'Tools' },
      { id: 'billing', label: zh.value ? '账单与权益' : 'Billing and benefits' },
      { id: 'custom', label: zh.value ? '更多资源' : 'Resources' },
    ]
    return groups.map(group => ({ ...group, items: items.value.filter(item => item.group === group.id) }))
      .filter(group => group.items.length > 0)
  })

  return { items, primaryItems, moreSections, purchaseLabel, refreshBatchImageAccess }
}
