import copy
import unittest

from deepseek_suite_pricing import NATIVE, channel_candidate, group_candidate, price_for, token_models


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
        self.assertTrue(flash["rate_independent"])
        self.assertNotIn("input_price", flash)
        for model, (unit, amount) in NATIVE.items():
            price = price_for(result, "deepseek", model)
            self.assertEqual(price["metered_unit"], unit)
            self.assertEqual(price["per_request_price"], amount)
            self.assertTrue(price["rate_independent"])

    def test_no_guessed_token_price(self):
        with self.assertRaisesRegex(RuntimeError, "missing canonical"):
            channel_candidate({"model_pricing": []}, ["deepseek"])

    def test_native_price_conflict_fails_closed(self):
        channel = self.channel()
        channel["model_pricing"].append({"platform": "openai", "models": ["deepseek-asr"],
                                         "billing_mode": "per_request", "per_request_price": 99})
        with self.assertRaisesRegex(RuntimeError, "native price differs"):
            channel_candidate(channel, ["openai"])


if __name__ == "__main__":
    unittest.main()
