import { nextTick, ref, watch } from 'vue'
export interface AdminFormTab { id: string; zh: string; en: string }
export const accountFormTabs: AdminFormTab[] = [
  { id: 'basic', zh: '基础信息', en: 'General' },
  { id: 'auth', zh: '认证与模型', en: 'Authentication & models' },
  { id: 'scheduling', zh: '调度与分组', en: 'Scheduling & groups' },
  { id: 'billing', zh: '计费与额度', en: 'Billing & quotas' },
  { id: 'advanced', zh: '高级设置', en: 'Advanced' }
]
export function useAdminFormTabs(open: () => boolean, initial = 'basic') {
  const formTab = ref(initial)
  let pending: HTMLElement | null = null
  let reporting = false
  watch(open, visible => { if (visible) formTab.value = initial })
  function revealInvalid(event: Event) {
    const input = event.target as HTMLInputElement
    const panel = input.closest<HTMLElement>('[data-admin-tab]')
    if (!panel || reporting) return
    event.preventDefault()
    if (pending) return
    pending = input
    formTab.value = panel.dataset.adminTab!
    void nextTick(() => {
      input.focus()
      reporting = true
      input.reportValidity()
      reporting = false
      pending = null
    })
  }
  return { formTab, revealInvalid }
}
