import copy
import unittest

from deepseek_suite_pricing import NATIVE, channel_candidate, group_candidate, price_for, rollback_state, token_models


class SuitePricingTests(unittest.TestCase):
    def channel(self):
        return {"model_pricing": [
            {"platform": "openai", "models": ["deepseek-v4.1-flash"], "billing_mode": "token",
             "input_price": 2e-6, "output_price": 8e-6, "time_pricing": {"timezone": "UTC", "periods": []}},
            {"platform": "openai", "models": ["deepseek-v4-pro"], "billing_mode": "token",
             "input_price": 9e-6, "output_price": 27e-6},
            {"platform": "openai", "models": ["other-model"], "billing_mode": "token", "input_price": 5},
        ]}

    def test_both_platforms_are_priced_without_changing_existing_values(self):
        original = self.channel()
        before = copy.deepcopy(original)
        after = channel_candidate(original, ["openai", "deepseek"])
        self.assertEqual(original, before)
        self.assertEqual(after["model_pricing"][2], before["model_pricing"][2])
        for platform in ("openai", "deepseek"):
            for model in (*token_models(platform), *NATIVE):
                self.assertIsNotNone(price_for(after["model_pricing"], platform, model))
            self.assertEqual(price_for(after["model_pricing"], platform, "deepseek-flash")["input_price"], 2e-6)

    def test_group_rates_inherit_and_native_units_are_explicit(self):
        channel = channel_candidate(self.channel(), ["deepseek"])
        group = {"platform": "deepseek", "model_pricing": [
            {"platform": "deepseek", "models": ["other-model"], "input_price": 5}
        ]}
        result = group_candidate(group, channel)["model_pricing"]
        self.assertEqual(result[0], group["model_pricing"][0])
        flash = price_for(result, "deepseek", "deepseek-flash")
        self.assertIsNone(flash, "text prices and schedules must come from the channel")
        for model, (unit, amount) in NATIVE.items():
            price = price_for(result, "deepseek", model)
            self.assertEqual(price["metered_unit"], unit)
            self.assertEqual(price["per_request_price"], amount)
            self.assertTrue(price["rate_independent"])

    def test_remove_previous_rate_only_text_cards_including_channel_aliases(self):
        channel = self.channel()
        aliases = ["deepseek-pro", "deepseek-v4-pro-0813", "deepseek-v4.1-flash-2099"]
        channel["model_pricing"][0]["models"].extend(aliases)
        channel = channel_candidate(channel, ["openai"])
        group = {"platform": "openai", "model_pricing": [
            {"platform": "openai", "models": ["deepseek-flash", *aliases],
             "billing_mode": "token", "rate_independent": True,
             "input_price": None, "output_price": None, "intervals": []}
        ]}
        before = copy.deepcopy(group)
        result = group_candidate(group, channel)["model_pricing"]
        self.assertEqual(group, before)
        for model in ("deepseek-flash", *aliases):
            self.assertIsNone(price_for(result, "openai", model))
        self.assertEqual(len(result), len(NATIVE))
        self.assertEqual(group_candidate({**group, "model_pricing": result}, channel)["model_pricing"],
                         result, "repeated enrollment must not restore independent text rates")

    def test_preserve_explicit_text_prices_including_zero_and_unrelated_cards(self):
        channel = channel_candidate(self.channel(), ["openai"])
        for amount in (0, 3e-6):
            entry = {"platform": "openai", "models": ["deepseek-flash"],
                     "billing_mode": "token", "rate_independent": True, "input_price": amount}
            group = {"platform": "openai", "model_pricing": [
                entry, {"platform": "openai", "models": ["other-model"], "rate_independent": True}
            ]}
            result = group_candidate(group, channel)["model_pricing"]
            expected = {k: v for k, v in entry.items() if k != "rate_independent"}
            self.assertEqual(result[0], expected)
            self.assertEqual(result[1], group["model_pricing"][1])

    def test_no_guessed_token_price(self):
        with self.assertRaisesRegex(RuntimeError, "missing canonical"):
            channel_candidate({"model_pricing": []}, ["deepseek"])

    def test_native_price_conflict_fails_closed(self):
        channel = self.channel()
        channel["model_pricing"].append({"platform": "openai", "models": ["deepseek-asr"],
                                         "billing_mode": "per_request", "per_request_price": 99})
        with self.assertRaisesRegex(RuntimeError, "native price differs"):
            channel_candidate(channel, ["openai"])

    def test_rollback_after_lost_put_response_uses_candidate_with_api_defaults(self):
        candidate = [{"platform": "deepseek", "models": ["deepseek-flash"],
                      "billing_mode": "token", "rate_independent": True, "intervals": []}]
        actual = [{**candidate[0], "id": 33, "input_price": None, "output_price": None,
                   "time_pricing": None, "sort_order": 0, "metered_unit": ""}]
        self.assertEqual(rollback_state(actual, [], candidate), "restore")

    def test_rollback_can_resume_after_prior_items_were_restored(self):
        before = self.channel()["model_pricing"]
        candidate = channel_candidate(self.channel(), ["openai"])["model_pricing"]
        self.assertEqual(rollback_state(before, before, candidate), "already-restored")

    def test_rollback_refuses_new_explicit_zero_price_and_other_drift(self):
        candidate = [{"platform": "deepseek", "models": ["deepseek-flash"],
                      "billing_mode": "token", "rate_independent": True}]
        for price in (0, 99):
            current = [{**candidate[0], "input_price": price}]
            with self.assertRaisesRegex(RuntimeError, "changed"):
                rollback_state(current, [], candidate)


if __name__ == "__main__":
    unittest.main()
