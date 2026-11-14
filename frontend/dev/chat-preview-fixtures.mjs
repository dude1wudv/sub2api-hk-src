import { randomUUID } from 'node:crypto'

// In-memory UI fixtures only. No upstream client, credential or database access.
export function createChatPreview(fixtures, { ok, error, unavailable, readJson }) {
  const owners = new Map()
  const capability = { transport: 'chat_completions', image_input: false, document_input: false, native_deepseek_extensions: false, native_files: false, native_search: false, native_read_url: false, reasoning: true, tool_call: false, search: false, read_url: false, voice: false, reasons: { image_input: '合成文本模型不支持图片', document_input: '合成文本模型不支持文档', search: '合成模型未配置搜索' } }
  capability.reasoning_efforts = ['low', 'medium', 'high']
  const model = { id: 'preview-chat-model', pricing: { billing_mode: 'token', input_price: 0.000001, output_price: 0.000002 }, capabilities: capability, available: true, unavailable_reason: '' }
  const groupView = id => { const group = fixtures.groups.find(group => group.id === id); return group && { ...group, user_rate_multiplier: 1 } }
  return async (request, response, user, path, params) => {
    if (!owners.has(user.id)) owners.set(user.id, { presets: [], sessions: new Map(), preferences: {}, revision: 0, runs: new Map() })
    const state = owners.get(user.id), method = request.method
    const fail = (status, code, message) => error(response, status, code, message)
    if (path === '/chat/catalog' && method === 'GET') {
      const group = groupView(Number(params.get('group_id')))
      return group ? ok(response, { group, models: [model].filter(model => model.id.includes(params.get('search') || '')) }) : fail(404, 'CHAT_NOT_FOUND', 'Synthetic group not found')
    }
    if (path === '/chat/presets') {
      if (method === 'GET') return ok(response, state.presets)
      if (method === 'POST') {
        const body = await readJson(request), group = groupView(body.group_id)
        if (!group || body.model !== model.id) return fail(404, 'CHAT_NOT_FOUND', 'Synthetic model not found')
        let preset = state.presets.find(item => item.group_id === body.group_id && item.model === body.model)
        if (!preset) { preset = { id: randomUUID(), group_id: group.id, group, model: model.id, display_name: body.display_name || '', sort_order: state.presets.length, is_default: false, normal_reasoning_effort: 'low', thinking_reasoning_effort: 'high', available: true, unavailable_reason: '', capabilities: capability }; state.presets.push(preset) }
        return ok(response, preset)
      }
    }
    if (path === '/chat/presets/order' && method === 'PUT') {
      const { ids } = await readJson(request)
      if (!Array.isArray(ids) || new Set(ids).size !== state.presets.length || ids.some(id => !state.presets.some(item => item.id === id))) return fail(400, 'CHAT_INVALID_REQUEST', 'Invalid order')
      state.presets.sort((a,b) => ids.indexOf(a.id)-ids.indexOf(b.id)); state.presets.forEach((item,index) => { item.sort_order=index }); return ok(response, {})
    }
    const segments = path.split('/').filter(Boolean), id = segments[2]
    if (segments[1] === 'presets' && segments.length === 3) {
      const preset = state.presets.find(item => item.id === id)
      if (!preset) return fail(404, 'CHAT_NOT_FOUND', 'Synthetic preset not found')
      if (method === 'DELETE') { state.presets = state.presets.filter(item => item.id !== id); return ok(response,{}) }
      if (method === 'PATCH') {
        const body = await readJson(request)
        for (const key of ['normal_reasoning_effort', 'thinking_reasoning_effort']) if (key in body && !capability.reasoning_efforts.includes(body[key])) return fail(400, 'CHAT_INVALID_REQUEST', 'Invalid reasoning effort')
        if (body.is_default) state.presets.forEach(item => { item.is_default = false })
        for (const key of ['display_name', 'is_default', 'normal_reasoning_effort', 'thinking_reasoning_effort']) if (key in body) preset[key] = body[key]
        return ok(response, preset)
      }
    }
    if (path === '/chat/preferences') {
      if (method === 'PUT') state.preferences = await readJson(request)
      if (method === 'GET' || method === 'PUT') return ok(response,{ preferences: state.preferences, history_revision: state.revision })
    }
    if (path === '/chat/sessions') {
      if (method === 'GET') return ok(response,{ items: [...state.sessions.values()].map(({session}) => ({ id:session.id,title:session.title,createdAt:session.createdAt,updatedAt:session.updatedAt })).sort((a,b) => b.updatedAt-a.updatedAt), next_cursor: '' })
      if (method === 'DELETE') { state.sessions.clear(); state.revision++; return ok(response,{}) }
    }
    if (segments[1] !== 'sessions' || !id) return unavailable(response)
    const existing = state.sessions.get(id)
    if (segments.length === 3) {
      if (method === 'GET') return existing ? ok(response,existing) : fail(404,'CHAT_NOT_FOUND','Synthetic session not found')
      if (method === 'DELETE') { state.sessions.delete(id); state.revision++; return ok(response,{}) }
      if (method === 'PUT' || method === 'PATCH') {
        const body = await readJson(request)
        if (body.expected_revision !== (existing?.revision ?? 0)) return fail(409,'CHAT_REVISION_CONFLICT','此会话已在其他设备更新，请重新载入')
        if (method === 'PATCH' && !existing) return fail(404,'CHAT_NOT_FOUND','Synthetic session not found')
        const result = {session:method === 'PUT' ? body.session : {...existing.session,title:body.title},revision:existing ? existing.revision+1 : 0}
        state.sessions.set(id,result); state.revision++; return ok(response,result)
      }
    }
    if (!existing) return fail(404,'CHAT_NOT_FOUND','Synthetic session not found')
    if (segments[3] === 'stop' && method === 'POST') {
      const body = await readJson(request), run = state.runs.get(id)
      if (!run || run.id !== body.run_id) return fail(404, 'CHAT_NOT_FOUND', 'Synthetic run not found')
      run.stop(); return ok(response, {})
    }
    if (segments[3] === 'generate' && method === 'POST') {
      const body = await readJson(request), preset = state.presets.find(item => item.id === body.preset_id)
      const answer = existing.session.messages.find(message => message.id === body.assistant_message_id && message.role === 'assistant')
      if (!preset || !answer) return fail(400,'CHAT_INVALID_REQUEST','Invalid synthetic generation')
      if (body.expected_revision !== existing.revision) return fail(409,'CHAT_REVISION_CONFLICT','此会话已在其他设备更新，请重新载入')
      const question = existing.session.messages.find(message => message.id === answer.parentId)?.content || ''
      const text = 'CHAT_PREVIEW_OK · 合成流式回复，未调用真实模型。\n\n' + '## 渲染检查\n\n**粗体**、`code`、公式 $x^2 + y^2 = z^2$。\n\n```js\nconsole.log(42)\n```\n\n```mermaid\ngraph LR\n A[问题] --> B[回答]\n```\n\n<script>window.__chatXss = true</script>\n\n[恶意链接](javascript:alert(1))'
      const payload = text
      answer.status='streaming'; answer.groupId=preset.group_id; answer.model=preset.model; answer.presetId=preset.id; answer.prefix=true; answer.blocks=[{type:'thinking',thinking:'这是合成思考块。'},{type:'text',text:''}]
      response.writeHead(200,{'Content-Type':'text/event-stream','Cache-Control':'no-store'})
      const send = event => response.write(`data: ${JSON.stringify(event)}

`)
      const runID = randomUUID()
      send({type:'chat_run',run_id:runID}); send({type:'content_block_start',index:0,content_block:answer.blocks[0]}); send({type:'content_block_stop',index:0}); send({type:'content_block_start',index:1,content_block:{type:'text',text:''}})
      let offset=0,finished=false
      const finish = stopped => {
        if (finished) return
        finished = true; clearInterval(timer)
        answer.status = stopped ? 'stopped' : 'done'
        existing.revision++; state.revision++; existing.session.updatedAt = Date.now()
        if (existing.session.title === '新对话' || !existing.session.title) existing.session.title = question.slice(0,40) || '合成预览'
        if (!stopped) {
          send({ type:'content_block_stop', index:1 }); send({ type:'message_delta', delta:{ stop_reason:'end_turn', stop_sequence:null } })
          send({ type:'message_stop' }); send({ type:'chat_saved', revision:existing.revision })
        }
        response.end()
      }
      const timer=setInterval(()=>{ const chunk=payload.slice(offset,offset+8); offset+=chunk.length; answer.blocks[1].text+=chunk; send({type:'content_block_delta',index:1,delta:{type:'text_delta',text:chunk}}); if(offset>=payload.length)finish(false) },question.includes('慢')?150:20)
      state.runs.set(id, { id:runID, stop:()=>finish(true) })
      response.on('close',()=>finish(true)); return
    }
    return unavailable(response)
  }
}
