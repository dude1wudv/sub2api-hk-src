import type Icon from '@/components/icons/Icon.vue'
import type { RegisteredFeatureFlag } from '@/utils/featureFlags'

export type AdminIcon = InstanceType<typeof Icon>['$props']['name']
export interface AdminNavEntry {
  path: string
  key: string
  icon: AdminIcon
  standardOnly?: boolean
  flag?: RegisteredFeatureFlag | 'ops' | 'adminPayment'
}
export interface AdminNavGroup {
  id: string
  zh: string
  en: string
  items: AdminNavEntry[]
}

// Keep feature/simple-mode predicates identical to the established administration menu.
export const adminNavigation: AdminNavGroup[] = [
  { id: 'overview', zh: '总览', en: 'Overview', items: [
    { path: '/admin/dashboard', key: 'nav.dashboard', icon: 'chart' },
    { path: '/admin/ops', key: 'nav.ops', icon: 'bolt', flag: 'ops' }
  ] },
  { id: 'resources', zh: '资源', en: 'Resources', items: [
    { path: '/admin/accounts', key: 'nav.accounts', icon: 'server' },
    { path: '/admin/groups', key: 'nav.groups', icon: 'grid' },
    { path: '/admin/channels/pricing', key: 'nav.channelPricing', icon: 'dollar', standardOnly: true },
    { path: '/admin/channels/monitor', key: 'nav.channelMonitor', icon: 'bolt', standardOnly: true, flag: 'channelMonitor' },
    { path: '/admin/proxies', key: 'nav.proxies', icon: 'globe' }
  ] },
  { id: 'billing', zh: '用户与计费', en: 'Users & billing', items: [
    { path: '/admin/users', key: 'nav.users', icon: 'users', standardOnly: true },
    { path: '/admin/subscriptions', key: 'nav.subscriptions', icon: 'creditCard', standardOnly: true, flag: 'subscription' },
    { path: '/admin/redeem', key: 'nav.redeemCodes', icon: 'gift', standardOnly: true },
    { path: '/admin/promo-codes', key: 'nav.promoCodes', icon: 'gift', standardOnly: true },
    { path: '/admin/orders/dashboard', key: 'nav.paymentDashboard', icon: 'chart', standardOnly: true, flag: 'adminPayment' },
    { path: '/admin/orders', key: 'nav.orderManagement', icon: 'document', standardOnly: true, flag: 'adminPayment' },
    { path: '/admin/orders/plans', key: 'nav.paymentPlans', icon: 'cube', standardOnly: true, flag: 'adminPayment' },
    { path: '/admin/affiliates/invites', key: 'nav.affiliateInviteRecords', icon: 'userPlus', standardOnly: true, flag: 'affiliate' },
    { path: '/admin/affiliates/rebates', key: 'nav.affiliateRebateRecords', icon: 'dollar', standardOnly: true, flag: 'affiliate' },
    { path: '/admin/affiliates/transfers', key: 'nav.affiliateTransferRecords', icon: 'swap', standardOnly: true, flag: 'affiliate' }
  ] },
  { id: 'security', zh: '安全与审计', en: 'Security & audit', items: [
    { path: '/admin/risk-control', key: 'nav.contentModeration', icon: 'shield', flag: 'riskControl' },
    { path: '/admin/prompt-audit', key: 'nav.promptAudit', icon: 'document', flag: 'riskControl' },
    { path: '/admin/usage', key: 'nav.usage', icon: 'chartBar' },
    { path: '/admin/audit-logs', key: 'nav.auditLogs', icon: 'clipboard', standardOnly: true }
  ] },
  { id: 'system', zh: '系统', en: 'System', items: [
    { path: '/admin/announcements', key: 'nav.announcements', icon: 'bell' },
    { path: '/admin/plugins', key: 'nav.plugins', icon: 'cube', flag: 'pluginManagement' },
    { path: '/admin/settings', key: 'nav.settings', icon: 'cog' }
  ] }
]

export function visibleAdminNavigation(simpleMode: boolean, enabled: (flag: NonNullable<AdminNavEntry['flag']>) => boolean) {
  return adminNavigation.map(group => ({ ...group, items: group.items.filter(item =>
    (!simpleMode || !item.standardOnly) && (!item.flag || enabled(item.flag))
  ) })).filter(group => group.items.length > 0)
}
