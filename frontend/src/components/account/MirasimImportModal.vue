<template>
  <BaseDialog :show="show" title="完成 Mirasim 授权导入" @close="close">
    <form id="mirasim-import-form" class="space-y-4" @submit.prevent="submit">
      <p class="input-hint">授权已返回。选择账号配置后导入；授权凭据仅暂存在此页面，刷新或关闭后需要重新授权。</p>
      <div>
        <label class="input-label">账号名称（可选）</label>
        <input v-model="name" class="input" maxlength="100" placeholder="留空使用授权账号名称" />
      </div>
      <div>
        <label class="input-label">代理</label>
        <ProxySelector v-model="proxyID" :proxies="proxies" :disabled="submitting" />
        <p class="input-hint">授权验证和后续模型请求均使用此代理。留空表示直连。</p>
      </div>
      <GroupSelector v-model="groupIDs" :groups="activeGroups" platform="mirasim" />
      <p class="input-hint">不选择时使用 mirasim-default；默认模型为 DeepSeek V4.1 Flash、GLM 5.3 Flash、Kimi K3。发布 API Key 前请核对分组价格。</p>
      <div>
        <label class="input-label">并发数</label>
        <input v-model.number="concurrency" class="input" type="number" min="1" max="100" required />
      </div>
    </form>
    <template #footer>
      <button class="btn btn-secondary" :disabled="submitting" @click="close">取消</button>
      <button class="btn btn-primary" type="submit" form="mirasim-import-form" :disabled="submitting">{{ submitting ? '导入中…' : '导入账号' }}</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ProxySelector from '@/components/common/ProxySelector.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import type { AdminGroup, Proxy } from '@/types'
import type { MirasimImportOptions } from '@/api/admin/accounts'

const props = defineProps<{ show: boolean; submitting: boolean; proxies: Proxy[]; groups: AdminGroup[] }>()
const emit = defineEmits<{ close: []; submit: [options: MirasimImportOptions] }>()
const name = ref('')
const proxyID = ref<number | null>(null)
const groupIDs = ref<number[]>([])
const concurrency = ref(5)
const activeGroups = computed(() => props.groups.filter(group => group.platform === 'mirasim' && group.status === 'active'))
watch(() => props.show, show => {
  if (show) {
    name.value = ''
    proxyID.value = null
    groupIDs.value = []
    concurrency.value = 5
  }
})
const close = () => { if (!props.submitting) emit('close') }
const submit = () => {
  if (props.submitting || !Number.isInteger(concurrency.value) || concurrency.value < 1 || concurrency.value > 100) return
  emit('submit', { name: name.value.trim(), proxy_id: proxyID.value, group_ids: groupIDs.value, concurrency: concurrency.value })
}
</script>
