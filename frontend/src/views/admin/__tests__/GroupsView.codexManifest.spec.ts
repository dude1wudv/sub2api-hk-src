import { defineComponent, h, type PropType } from "vue";
import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia, type Pinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AdminGroup, CodexModelsManifestConfig } from "@/types";
import GroupsView from "@/views/admin/hk/GroupsView.vue";
import PricingEntryCard from "@/components/admin/channel/PricingEntryCard.vue";
let pinia: Pinia;

const {
  listGroups,
  getModelAllowlistCandidates,
  getUsageSummary,
  getCapacitySummary,
  getLiveCapability,
  createGroup,
  updateGroup,
} = vi.hoisted(() => ({
  listGroups: vi.fn(),
  getModelAllowlistCandidates: vi.fn(),
  getUsageSummary: vi.fn(),
  getCapacitySummary: vi.fn(),
  getLiveCapability: vi.fn(),
  createGroup: vi.fn(),
  updateGroup: vi.fn(),
}));

vi.mock("@/api/admin", () => ({
  adminAPI: {
    groups: {
      list: listGroups,
      getAll: vi.fn().mockResolvedValue([]),
      getModelAllowlistCandidates,
      getUsageSummary,
      getCapacitySummary,
      getLiveCapability,
      create: createGroup,
      update: updateGroup,
      delete: vi.fn(),
      duplicate: vi.fn(),
      updateSortOrder: vi.fn(),
    },
    accounts: {
      list: vi.fn(),
      getById: vi.fn(),
    },
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}));

vi.mock("@/stores/onboarding", () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn(),
  }),
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  };
});

const sourceGroup = {
  id: 42,
  name: "OpenAI",
  description: null,
  platform: "openai",
  rate_multiplier: 1,
  rpm_limit: 0,
  is_exclusive: false,
  status: "active",
  subscription_type: "standard",
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  long_context_pricing_enabled: true,
  force_openai_fast: false,
  free_openai_fast: false,
  force_openai_ultrafast: false,
  free_openai_ultrafast: false,
  model_pricing: [],
  profit_control_enabled: false,
  profit_min_margin: 0,
  profit_safety_buffer: 0,
  allow_image_generation: false,
  allow_batch_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  batch_image_discount_multiplier: 0.5,
  batch_image_hold_multiplier: 0.6,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  video_rate_independent: false,
  video_rate_multiplier: 1,
  video_price_480p: null,
  video_price_720p: null,
  video_price_1080p: null,
  web_search_price_per_call: null,
  search_price_per_1k: null,
  audio_realtime_price_per_min: null,
  audio_tts_price_per_million_chars: null,
  audio_stt_price_per_hour: null,
  peak_rate_enabled: false,
  peak_start: "",
  peak_end: "",
  peak_rate_multiplier: 1,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  allow_messages_dispatch: false,
  allow_live: false,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: "2026-09-05T00:00:00Z",
  updated_at: "2026-09-05T00:00:00Z",
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: true,
  supported_model_scopes: [],
  account_count: 1,
  active_account_count: 1,
  rate_limited_account_count: 0,
  models_list_config: undefined,
  codex_models_manifest_config: {
    enabled: false,
    account_ids: [],
    fallback_to_scheduler: false,
  },
  sort_order: 10,
} satisfies AdminGroup;

const AppLayoutStub = defineComponent({
  template: "<main><slot /></main>",
});

const TablePageLayoutStub = defineComponent({
  template: '<section><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>',
});

const DataTableStub = defineComponent({
  props: {
    data: { type: Array, default: () => [] },
  },
  template: '<div><div v-if="data.length"><slot name="cell-actions" :row="data[0]" /></div></div>',
});

const BaseDialogStub = defineComponent({
  props: {
    show: { type: Boolean, default: false },
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
});

const CodexManifestAccountsFieldStub = defineComponent({
  name: "CodexManifestAccountsField",
  props: {
    modelValue: {
      type: Object as PropType<CodexModelsManifestConfig>,
      required: true,
    },
  },
  emits: ["update:modelValue"],
  setup(props, { emit, expose }) {
    expose({ validate: () => true, resetValidation: () => undefined });
    return () =>
      h("div", { "data-testid": "codex-manifest-field" }, [
        h(
          "output",
          { "data-testid": "codex-manifest-value" },
          JSON.stringify(props.modelValue),
        ),
        h(
          "button",
          {
            type: "button",
            "data-testid": "codex-manifest-enable",
            onClick: () =>
              emit("update:modelValue", {
                ...props.modelValue,
                enabled: true,
              }),
          },
          "enable",
        ),
        h(
          "button",
          {
            type: "button",
            "data-testid": "codex-manifest-select-account",
            onClick: () =>
              emit("update:modelValue", {
                ...props.modelValue,
                account_ids: [17],
              }),
          },
          "select account",
        ),
      ]);
  },
});

const mountView = () =>
  mount(GroupsView, {
    global: {
      plugins: [pinia],
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        EmptyState: true,
        Select: true,
        PlatformIcon: true,
        Icon: true,
        GroupCapacityBadge: true,
        GroupRateMultipliersModal: true,
        GroupRPMOverridesModal: true,
        CodexManifestAccountsField: CodexManifestAccountsFieldStub,
        PricingEntryCard: true,
        VueDraggable: true,
      },
    },
  });

describe("GroupsView Codex manifest and OpenAI tier forms", () => {
  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    localStorage.clear();
    listGroups.mockReset();
    getModelAllowlistCandidates.mockReset();
    getUsageSummary.mockReset();
    getCapacitySummary.mockReset();
    getLiveCapability.mockReset();
    createGroup.mockReset().mockResolvedValue(sourceGroup);
    updateGroup.mockReset().mockResolvedValue(sourceGroup);

    listGroups.mockResolvedValue({
      items: [sourceGroup],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    });
    getModelAllowlistCandidates.mockResolvedValue([]);
    getUsageSummary.mockResolvedValue([]);
    getCapacitySummary.mockResolvedValue([]);
    getLiveCapability.mockResolvedValue({ supported: false });
  });

  it("preserves consecutive child updates on the reactive edit config", async () => {
    const wrapper = mountView();
    await flushPromises();

    const editButton = wrapper
      .findAll("button")
      .find((button) => button.text().includes("common.edit"));
    expect(editButton).toBeTruthy();
    await editButton!.trigger("click");
    await flushPromises();

    expect(wrapper.get('[data-testid="codex-manifest-value"]').text()).toBe(
      JSON.stringify(sourceGroup.codex_models_manifest_config),
    );

    await wrapper.get('[data-testid="codex-manifest-enable"]').trigger("click");
    await flushPromises();
    expect(wrapper.get('[data-testid="codex-manifest-value"]').text()).toContain(
      '"enabled":true',
    );

    await wrapper
      .get('[data-testid="codex-manifest-select-account"]')
      .trigger("click");
    await flushPromises();
    expect(wrapper.get('[data-testid="codex-manifest-value"]').text()).toBe(
      JSON.stringify({
        enabled: true,
        account_ids: [17],
        fallback_to_scheduler: false,
      }),
    );

    wrapper.unmount();
  });

  async function openCreate(wrapper: ReturnType<typeof mountView>, platform: string) {
    await flushPromises();
    await wrapper.findAll('button').find(button => button.text() === 'admin.groups.createGroup')!.trigger('click');
    await flushPromises();
    const select = wrapper.getComponent('[data-tour="group-form-platform"]');
    select.vm.$emit('update:modelValue', platform);
    await flushPromises();
    expect(select.props('modelValue')).toBe(platform);
    await wrapper.get('#create-group-form input[type="text"]').setValue('Speed group');
    await wrapper.get('#create-group-form-advanced-tab').trigger('click');
    return select;
  }

  it.each(['openai', 'composite'])('creates %s groups with independent Fast and Ultra Fast settings', async platform => {
    const wrapper = mountView();
    await openCreate(wrapper, platform);
    for (const setting of ['force-openai-fast', 'free-openai-fast', 'force-openai-ultrafast', 'free-openai-ultrafast']) {
      expect(wrapper.get(`[data-testid="create-${setting}"]`).attributes('aria-checked')).toBe('false');
    }
    await wrapper.get('[data-testid="create-force-openai-fast"]').trigger('click');
    await wrapper.get('[data-testid="create-force-openai-ultrafast"]').trigger('click');
    await wrapper.get('[data-testid="create-free-openai-ultrafast"]').trigger('click');
    expect(wrapper.get('[data-testid="create-free-openai-fast"]').attributes('aria-checked')).toBe('false');
    await wrapper.get('#create-group-form').trigger('submit');
    await flushPromises();
    expect(createGroup).toHaveBeenCalledWith(expect.objectContaining({
      platform, force_openai_fast: true, free_openai_fast: false,
      force_openai_ultrafast: true, free_openai_ultrafast: true,
    }));
    await openCreate(wrapper, platform);
    expect(wrapper.get('[data-testid="create-force-openai-ultrafast"]').attributes('aria-checked')).toBe('false');
    expect(wrapper.get('[data-testid="create-free-openai-ultrafast"]').attributes('aria-checked')).toBe('false');
    wrapper.unmount();
  });

  it('hides and clears stale Ultra Fast settings when changing to an unsupported platform', async () => {
    const wrapper = mountView();
    const select = await openCreate(wrapper, 'openai');
    await wrapper.get('[data-testid="create-force-openai-ultrafast"]').trigger('click');
    await wrapper.get('[data-testid="create-free-openai-ultrafast"]').trigger('click');
    await wrapper.get('#create-group-form-basic-tab').trigger('click');
    select.vm.$emit('update:modelValue', 'anthropic');
    await flushPromises();
    expect(wrapper.find('[data-testid="create-force-openai-ultrafast"]').exists()).toBe(false);
    await wrapper.get('#create-group-form').trigger('submit');
    await flushPromises();
    expect(createGroup).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'anthropic', force_openai_fast: false, free_openai_fast: false,
      force_openai_ultrafast: false, free_openai_ultrafast: false,
    }));
    wrapper.unmount();
  });

  async function openPricedGroup() {
    listGroups.mockResolvedValue({ items: [{ ...sourceGroup,
      force_openai_ultrafast: true, free_openai_ultrafast: true,
      model_pricing: [{ platform: 'openai', models: ['gpt-5.5'], billing_mode: 'token',
        input_price: 3e-6, output_price: 15e-6, cache_write_price: null, cache_read_price: null,
        fast_multiplier: 3, ultrafast_multiplier: 9, flex_multiplier: 0.5,
        image_input_price: null, image_output_price: null, per_request_price: null,
        intervals: [], time_pricing: null }],
    }], total: 1, page: 1, page_size: 20, pages: 1 });
    const wrapper = mountView();
    await flushPromises();
    await wrapper.findAll('button').find(button => button.text().includes('common.edit'))!.trigger('click');
    await flushPromises();
    await wrapper.get('#edit-group-form-advanced-tab').trigger('click');
    return wrapper;
  }

  it('loads group tier overrides and saves edited Ultra Fast pricing without dropping Fast or Flex', async () => {
    const wrapper = await openPricedGroup();
    expect(wrapper.get('[data-testid="edit-force-openai-ultrafast"]').attributes('aria-checked')).toBe('true');
    expect(wrapper.get('[data-testid="edit-free-openai-ultrafast"]').attributes('aria-checked')).toBe('true');
    expect(wrapper.get('[data-testid="edit-free-openai-fast"]').attributes('aria-checked')).toBe('false');
    await wrapper.get('#edit-group-form-billing-tab').trigger('click');
    const card = wrapper.getComponent(PricingEntryCard);
    expect(card.props('enableTierMultipliers')).toBe(true);
    expect(card.props('entry')).toMatchObject({ input_price: 3, fast_multiplier: 3, ultrafast_multiplier: 9, flex_multiplier: 0.5 });
    card.vm.$emit('update', { ...card.props('entry'), ultrafast_multiplier: '10' });
    await wrapper.get('#edit-group-form-advanced-tab').trigger('click');
    await wrapper.get('[data-testid="edit-free-openai-ultrafast"]').trigger('click');
    await wrapper.get('#edit-group-form').trigger('submit');
    await flushPromises();
    expect(updateGroup).toHaveBeenCalledWith(42, expect.objectContaining({
      force_openai_ultrafast: true, free_openai_ultrafast: false, free_openai_fast: false,
      model_pricing: [expect.objectContaining({ input_price: 3e-6, fast_multiplier: 3, ultrafast_multiplier: 10, flex_multiplier: 0.5 })],
    }));
    wrapper.unmount();
  });

  it('rejects an invalid group Ultra Fast multiplier and clears an override to the default', async () => {
    const wrapper = await openPricedGroup();
    await wrapper.get('#edit-group-form-billing-tab').trigger('click');
    const card = wrapper.getComponent(PricingEntryCard);
    card.vm.$emit('update', { ...card.props('entry'), ultrafast_multiplier: '0' });
    await flushPromises();
    await wrapper.get('#edit-group-form').trigger('submit');
    expect(updateGroup).not.toHaveBeenCalled();
    card.vm.$emit('update', { ...card.props('entry'), ultrafast_multiplier: '' });
    await flushPromises();
    await wrapper.get('#edit-group-form').trigger('submit');
    await flushPromises();
    expect(updateGroup).toHaveBeenCalledWith(42, expect.objectContaining({
      model_pricing: [expect.objectContaining({ fast_multiplier: 3, ultrafast_multiplier: null, flex_multiplier: 0.5 })],
    }));
    wrapper.unmount();
  });
});
