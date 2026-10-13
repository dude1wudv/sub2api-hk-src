// Synthetic charts for manual/CUA layout inspection. No backend or browser automation.
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'
import vue from '@vitejs/plugin-vue'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const previewPath = '/distribution-layout-preview.html'
const fixture = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Distribution cards · synthetic preview</title></head><body>
<div id="app"></div>
<script type="module">
import { createApp, h, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import Model from '/src/components/charts/ModelDistributionChart.vue'
import Group from '/src/components/charts/GroupDistributionChart.vue'
import Endpoint from '/src/components/charts/EndpointDistributionChart.vue'
import messages from '/src/i18n/locales/en/index.ts'
// Match main.ts: the design system loads through hk/bootstrap before Tailwind.
import '/src/styles/design-system.css'
import '/src/style.css'
const widths = [300, 360, 400, 620, 840]
const requestedWidth = Number(new URLSearchParams(location.search).get('width'))
const cardWidth = ref(widths.includes(requestedWidth) ? requestedWidth : 360)
const metrics = { requests: 1234567890123, total_tokens: 9876543210, actual_cost: 1234567.89, account_cost: 2345678.91, cost: 3456789.12 }
const name = 'very-long-model-or-group-name-without-spaces-and-with-many-identifiers-'.repeat(3)
const cards = [
 ['model', '模型分布', Model, { modelStats: [{ ...metrics, model: name }], showSourceToggle: true, showMetricToggle: true, enableRankingView: true, enableBreakdown: false }],
 ['group', '分组分布', Group, { groupStats: [{ ...metrics, group_id: 1, group_name: name }], showMetricToggle: true, enableBreakdown: false }],
 ['endpoint', '端点分布', Endpoint, { endpointStats: [{ ...metrics, endpoint: '/v1/' + name }], showSourceToggle: true, showMetricToggle: true, enableBreakdown: false }],
 ['ranking', '消费排名：点击卡片中的 User Spending Ranking 切换', Model, { modelStats: [], enableRankingView: true, enableBreakdown: false, rankingItems: [{ user_id: 1, email: name + '@example.test', actual_cost: metrics.actual_cost, requests: metrics.requests, tokens: metrics.total_tokens }] }],
]
createApp({ render: () => h('main', { style: 'padding:12px;max-width:100%;' }, [
 h('header', { style: 'display:flex;flex-wrap:wrap;align-items:center;gap:12px;margin-bottom:20px;' }, [
  h('h1', { style: 'font-size:18px;font-weight:600;' }, '分布卡片 · 本地合成预览'),
  h('label', { style: 'display:flex;align-items:center;gap:8px;' }, [
   '卡片宽度',
   h('select', { id: 'preview-card-width', 'aria-label': '卡片宽度', value: cardWidth.value, onChange: event => { cardWidth.value = Number(event.target.value) }, style: 'border:1px solid #ccc;border-radius:6px;padding:6px;' }, widths.map(width => h('option', { value: width }, width + 'px'))),
  ]),
  h('p', { style: 'font-size:12px;' }, '数据均为合成；未连接后端。可调整卡片宽度与浏览器视口，手动验证横向滚动。'),
 ]),
 h('div', { style: 'display:flex;flex-wrap:wrap;align-items:flex-start;gap:24px;max-width:100%;' }, cards.map(([key, label, component, props]) => h('section', { 'data-case': key, style: 'width:' + cardWidth.value + 'px;max-width:100%;min-width:0;' }, [
  h('h2', { style: 'font-size:12px;margin-bottom:8px;overflow-wrap:anywhere;' }, label),
  h(component, props),
 ]))),
]) }).use(createI18n({ legacy: false, locale: 'en', messages: { en: messages } })).mount('#app')
</script></body></html>`

const server = await createServer({
  root: frontendRoot,
  configFile: false,
  envFile: false,
  plugins: [vue(), {
    name: 'distribution-layout-preview',
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        if (/^\/(api|v1|setup|health)(\/|\?|$)/.test(request.url || '')) {
          response.statusCode = 501
          response.end('Synthetic preview: backend operations are unavailable.')
          return
        }
        next()
      })
      server.middlewares.use(previewPath, async (_request, response) => {
        response.setHeader('Content-Type', 'text/html; charset=utf-8')
        response.end(await server.transformIndexHtml(previewPath, fixture))
      })
    },
  }],
  resolve: { alias: { '@': resolve(frontendRoot, 'src'), 'vue-i18n': 'vue-i18n/dist/vue-i18n.runtime.esm-bundler.js' } },
  define: { __VUE_I18N_FULL_INSTALL__: true, __VUE_I18N_LEGACY_API__: false, __INTLIFY_PROD_DEVTOOLS__: false, __INTLIFY_JIT_COMPILATION__: true, __INTLIFY_DROP_MESSAGE_COMPILER__: false },
  server: { host: '127.0.0.1', port: 3430, strictPort: true },
  logLevel: 'error',
})
await server.listen()
console.log('Distribution preview: http://127.0.0.1:3430' + previewPath)
console.log('Synthetic data only. Ctrl+C stops the local Vite server.')
let closing = false
const close = async () => {
  if (closing) return
  closing = true
  await server.close()
}
process.on('SIGINT', close)
process.on('SIGTERM', close)
