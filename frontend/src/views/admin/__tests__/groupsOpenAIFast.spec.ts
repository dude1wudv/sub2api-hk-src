import { describe, expect, it } from "vitest";

import {
  normalizeGroupOpenAIFast,
  supportsGroupOpenAIFast,
} from "../groupsOpenAIFast";
import en from "@/i18n/locales/en/admin/overview";
import zh from "@/i18n/locales/zh/admin/overview";

describe("groupsOpenAIFast", () => {
  it("supports OpenAI and composite groups", () => {
    expect(supportsGroupOpenAIFast("openai")).toBe(true);
    expect(supportsGroupOpenAIFast("composite")).toBe(true);
    expect(supportsGroupOpenAIFast("anthropic")).toBe(false);
  });

  it("clears stale enabled state on unsupported platforms", () => {
    expect(normalizeGroupOpenAIFast("openai", true)).toBe(true);
    expect(normalizeGroupOpenAIFast("composite", true)).toBe(true);
    expect(normalizeGroupOpenAIFast("anthropic", true)).toBe(false);
    expect(normalizeGroupOpenAIFast("openai", false)).toBe(false);
  });

  it("provides localized switch labels", () => {
    expect(zh.groups.openaiFast).toMatchObject({
      title: expect.any(String),
      force: expect.any(String),
      hint: expect.stringContaining("service_tier=priority"),
      free: expect.any(String),
      freeHint: expect.stringContaining("Standard"),
    });
    expect(en.groups.openaiFast).toMatchObject({
      title: expect.any(String),
      force: expect.any(String),
      hint: expect.stringContaining("service_tier=priority"),
      free: expect.any(String),
      freeHint: expect.stringContaining("Standard"),
    });
    for (const locale of [zh, en]) {
      expect(locale.groups.openaiFast.hint).toContain('2.5');
      expect(locale.groups.openaiUltraFast).toMatchObject({
        title: expect.any(String), force: expect.any(String), free: expect.any(String),
        hint: expect.stringContaining('service_tier=ultrafast'),
        freeHint: expect.stringContaining('Standard'),
      });
      expect(locale.groups.openaiUltraFast.hint).toContain('8');
      expect(locale.groups.openaiUltraFast.hint).toContain('WebSocket');
    }
    expect(zh.groups.openaiFast.hint).toContain('优先于全局');
    expect(en.groups.openaiFast.hint).toContain('precedence over the global');
  });
});
