import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { createPinia } from "pinia";
import { createI18n } from "vue-i18n";
import type { UserSubscription } from "@/types";
import SubscriptionPlanCard from "../SubscriptionPlanCard.vue";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackWarn: false,
  missingWarn: false,
  messages: {
    en: {
      payment: {
        days: "days",
        balanceUnit: "balance",
        saleEndsAt: "Sale ends: {time}",
        models: "Models",
        planCard: {
          quota: "Quota",
          totalLimit: "Total Limit",
          rate: "Rate",
          unlimited: "Unlimited",
          onePurchaseOnly: "One purchase only",
          onePurchaseOnlyHint: "This plan can only be purchased once per account.",
        },
        subscribeNow: "Subscribe now",
      },
    },
  },
});

const mountPlanCard = (
  groupPlatform: string,
  plan: Record<string, unknown> = {},
  activeSubscriptions: UserSubscription[] = [],
) =>
  mount(SubscriptionPlanCard, {
    props: {
      plan: {
        id: 1,
        group_id: 10,
        group_platform: groupPlatform,
        name: "Pro",
        price: 10,
        amount: 1000,
        features: [],
        rate_multiplier: 1,
        validity_days: 30,
        validity_unit: "day",
        one_purchase_per_user: false,
        supported_model_scopes: ["claude", "gemini_text", "gemini_image"],
        is_active: true,
        ...plan,
      },
      activeSubscriptions,
    },
    global: { plugins: [i18n, createPinia()] },
  });

describe("SubscriptionPlanCard", () => {
  it("does not show Antigravity model scopes for OpenAI plans", () => {
    const text = mountPlanCard("openai").text();

    expect(text).not.toContain("Claude");
    expect(text).not.toContain("Gemini");
    expect(text).not.toContain("Imagen");
  });

  it("shows model scopes for Antigravity plans", () => {
    const text = mountPlanCard("antigravity").text();

    expect(text).toContain("Claude");
    expect(text).toContain("Gemini");
    expect(text).toContain("Imagen");
  });

  it("labels one-purchase plans, splits escaped newlines, and does not offer renewal", () => {
    const text = mountPlanCard("openai", {
      one_purchase_per_user: true,
      features: ["First feature\\nSecond feature"],
    }, [{ group_id: 10, status: "active" } as UserSubscription]).text();

    expect(text).toContain("payment.planCard.onePurchaseOnly");
    expect(text).toContain("First feature");
    expect(text).toContain("Second feature");
    expect(text).toContain("payment.subscribeNow");
    expect(text).not.toContain("payment.renewNow");
  });

  it("labels balance plans without an external-payment dollar prefix", () => {
    const text = mountPlanCard("openai", {
      purchase_mode: "balance",
      price: 5,
      daily_limit_usd: 200,
      sale_ends_at: "2026-07-13T09:00:00.000Z",
    }).text();

    expect(text).toContain("5");
    expect(text).toContain("payment.balanceUnit");
    expect(text).toContain("payment.planCard.totalLimit");
    expect(text).toContain("payment.saleEndsAt");
  });
});
