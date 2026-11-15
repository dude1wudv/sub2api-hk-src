"""HK-only, additive suite pricing enrollment. Private snapshots stay on HK.

Usage: python3 deepseek_suite_pricing.py RELEASE_DIR EXPECTED_SHA [rollback]
Run after the application supports group rate_independent price cards.
Only groups 81/179/223 and their existing channels 1/27 are touched.
"""
import copy
import datetime as dt
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request

SCOPES = {81: (1, "openai"), 179: (27, "deepseek"), 223: (27, "openai")}
NATIVE = {
    "deepseek-search": ("call", .005),
    "deepseek-fetch": ("call", .005),
    "deepseek-ocr": ("page", .0015),
    "deepseek-files": ("page", .0015),
    "deepseek-asr": ("hour", .78),
    "deepseek-tts": ("10k_characters", 3),
    "deepseek-tts_ws": ("10k_characters", 3),
}
TOKEN_MODELS = (
    "deepseek-flash", "deepseek-v4.1-flash", "deepseek-v4.1-flash-1017",
    "deepseek/deepseek-v4.1-flash", "deepseek/deepseek-v4.1-flash-fast",
    "deepseek-v4-pro",
)


def save_journal(path, value):
    with tempfile.NamedTemporaryFile(mode="w", dir=path.parent, delete=False) as handle:
        json.dump(value, handle)
        handle.flush()
        os.fsync(handle.fileno())
    os.replace(handle.name, path)


def token_models(platform):
    return tuple(m for m in TOKEN_MODELS if platform == "deepseek" or not m.startswith("deepseek/"))


def semantic(value):
    if isinstance(value, dict):
        return {k: semantic(v) for k, v in value.items()
                if k not in {"id", "channel_id", "created_at", "updated_at"}}
    if isinstance(value, list):
        return [semantic(v) for v in value]
    return value


def normalized_prices(value):
    """Compare API defaults without hiding an explicitly configured zero price."""
    if isinstance(value, dict):
        result = {}
        for key, item in semantic(value).items():
            if key == "sort_order":
                continue  # Array order already preserves price/tier precedence.
            item = normalized_prices(item)
            if item is None or item is False or item == "" or item == [] or item == {}:
                continue
            result[key] = sorted(item) if key == "models" else item
        return result
    if isinstance(value, list):
        return [normalized_prices(item) for item in value]
    return value


def rollback_state(current, before, expected):
    if normalized_prices(current) == normalized_prices(before):
        return "already-restored"
    if normalized_prices(current) == normalized_prices(expected):
        return "restore"
    raise RuntimeError("pricing changed since enrollment; do not overwrite")


def price_for(prices, platform, model):
    return next((p for p in prices if p.get("platform") == platform
                 and model in p["models"]), None)


def channel_candidate(original, platforms):
    prices = copy.deepcopy(original["model_pricing"])
    for platform in platforms:
        for model in token_models(platform):
            if price_for(prices, platform, model):
                continue
            canonical = "deepseek-v4-pro" if model == "deepseek-v4-pro" else "deepseek-v4.1-flash"
            same_platform = price_for(prices, platform, canonical)
            if same_platform:
                same_platform["models"].append(model)
                continue
            source = next((p for p in prices if canonical in p["models"]), None)
            if source is None:
                raise RuntimeError("missing canonical token price: " + model)
            entry = semantic(copy.deepcopy(source))
            entry["platform"], entry["models"] = platform, [model]
            prices.append(entry)
        for model, (_, amount) in NATIVE.items():
            entry = price_for(prices, platform, model)
            if entry:
                if entry.get("billing_mode") != "per_request" or entry.get("per_request_price") != amount:
                    raise RuntimeError("native price differs from confirmed canonical price: " + model)
                continue
            prices.append({"platform": platform, "models": [model],
                           "billing_mode": "per_request", "per_request_price": amount,
                           "intervals": []})
    return {"model_pricing": prices}


def group_candidate(original, channel):
    platform = original["platform"]
    prices = copy.deepcopy(original.get("model_pricing") or [])
    tokens = set(token_models(platform))
    for p in channel["model_pricing"]:
        if p.get("platform") == platform and p.get("billing_mode") in ("token", "", None):
            tokens.update(m for m in p["models"] if m.lower().startswith(("deepseek-", "deepseek/")))
    remaining = set(tokens)
    for entry in prices:
        owned = set(entry["models"]) & (tokens | NATIVE.keys())
        if not owned:
            continue
        if owned != set(entry["models"]):
            raise RuntimeError("mixed owned/unowned group price card; split explicitly before enrollment")
        entry["rate_independent"] = True
        remaining.difference_update(owned)
    if remaining:
        # Empty price values intentionally inherit the current channel rates,
        # including its own time schedule and future operator price changes.
        prices.append({"platform": platform, "models": sorted(remaining),
                       "billing_mode": "token", "rate_independent": True, "intervals": []})
    for model, (unit, amount) in NATIVE.items():
        entry = price_for(prices, platform, model)
        if entry is None:
            entry = {"platform": platform, "models": [model], "intervals": []}
            prices.append(entry)
        entry.update(billing_mode="per_request", per_request_price=amount,
                     rate_independent=True, metered_unit=unit)
    return {"model_pricing": prices}


def main():
    os.umask(0o077)
    base, expected = Path(sys.argv[1]).resolve(), sys.argv[2]
    assert base.parent == Path("/opt/sub2api-deploy")
    app = json.loads(subprocess.check_output(["docker", "inspect", "sub2api"]))[0]
    assert app["Config"]["Labels"]["org.opencontainers.image.revision"] == expected
    assert app["State"]["Health"]["Status"] == "healthy"
    out = base / "audit/suite-pricing"
    rollback = len(sys.argv) > 3 and sys.argv[3] == "rollback"
    if not rollback:
        out.mkdir(mode=0o700, exist_ok=False)
    root = "http://127.0.0.1:" + app["NetworkSettings"]["Ports"]["8080/tcp"][0]["HostPort"]
    admin = Path("/opt/sub2api-external-quality/admin.key").read_text().strip()

    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, *args):
            return None

    http = urllib.request.build_opener(NoRedirect)

    def call(kind, ident, payload=None):
        data = None if payload is None else json.dumps(payload).encode()
        req = urllib.request.Request(root + "/api/v1/admin/" + kind + "/" + str(ident),
                                     data=data, method="GET" if payload is None else "PUT",
                                     headers={"x-api-key": admin, "Content-Type": "application/json"})
        try:
            with http.open(req, timeout=40) as response:
                result = json.load(response)
        except urllib.error.HTTPError as error:
            # Never include response bodies or credential-bearing requests in logs.
            raise RuntimeError(f"{kind}/{ident}: HTTP {error.code}") from None
        return result.get("data", result)

    if rollback:
        before = json.loads((out / "before.json").read_text())
        plan = json.loads((out / "plan.json").read_text())
        journal = out / "attempted.json"
        if not journal.exists():
            journal = out / "applied.json"
        attempted = json.loads(journal.read_text()) if journal.exists() else []
        assert not (out / "rollback.json").exists()
        restored = []
        for item in reversed(attempted):
            kind, ident = item["kind"], str(item["id"])
            after_file = out / f"{kind}-{ident}.after.json"
            expected_prices = (json.loads(after_file.read_text()) if after_file.exists()
                               else plan[kind][ident])["model_pricing"]
            current = call(kind, ident)
            state = rollback_state(current["model_pricing"], before[kind][ident]["model_pricing"],
                                   expected_prices)
            if state == "restore":
                call(kind, ident, {"model_pricing": before[kind][ident]["model_pricing"]})
            actual = call(kind, ident)
            assert normalized_prices(actual["model_pricing"]) == normalized_prices(before[kind][ident]["model_pricing"])
            restored.append(item)
            (out / "rollback-progress.json").write_text(json.dumps(restored))
        (out / "rollback.json").write_text(json.dumps({"restored": restored}))
        print(json.dumps({"restored": restored, "application_unchanged": True}))
        return

    before = {"channels": {str(i): call("channels", i) for i in (1, 27)},
              "groups": {str(i): call("groups", i) for i in SCOPES}}
    (out / "before.json").write_text(json.dumps(before, indent=2))
    candidates = {"channels": {}, "groups": {}}
    for cid, platforms in ((1, ["openai"]), (27, ["openai", "deepseek"])):
        original = before["channels"][str(cid)]
        required = {gid for gid, (channel_id, _) in SCOPES.items() if channel_id == cid}
        assert required.issubset(set(original["group_ids"]))
        candidates["channels"][str(cid)] = channel_candidate(original, platforms)
    for gid, (cid, platform) in SCOPES.items():
        original = before["groups"][str(gid)]
        assert original["platform"] == platform and original["status"] == "active"
        candidates["groups"][str(gid)] = group_candidate(original, candidates["channels"][str(cid)])
    (out / "plan.json").write_text(json.dumps(candidates, indent=2))
    applied, attempted = [], []
    for kind in ("channels", "groups"):
        for ident, payload in candidates[kind].items():
            current = call(kind, ident)
            assert current == before[kind][ident], "configuration changed concurrently; do not overwrite"
            attempted.append({"kind": kind, "id": int(ident)})
            save_journal(out / "attempted.json", attempted)
            call(kind, ident, payload)
            applied.append({"kind": kind, "id": int(ident)})
            (out / "applied.json").write_text(json.dumps(applied))
            actual = call(kind, ident)
            save_journal(out / f"{kind}-{ident}.after.json", actual)
            # API normalizes omitted nullable/default fields. Compare the owned
            # semantic price fields, and keep full before/after private for audit.
            for proposed in payload["model_pricing"]:
                got = next((p for p in actual["model_pricing"]
                            if p.get("platform") == proposed.get("platform")
                            and set(p["models"]) == set(proposed["models"])), None)
                assert got is not None, "missing saved price card"
                for field, value in semantic(proposed).items():
                    if field == "models":
                        continue
                    assert semantic(got.get(field)) == value, f"price field mismatch: {field}"
            for field in (("group_ids", "model_mapping", "restrict_models", "billing_model_source")
                          if kind == "channels" else ("rate_multiplier", "peak_rate_enabled", "model_allowlist")):
                assert actual.get(field) == before[kind][ident].get(field), "unowned field changed: " + field
    result = {"sha": expected, "finished_at": dt.datetime.now(dt.timezone.utc).isoformat(),
              "groups": {}, "native_prices": NATIVE, "applied": applied,
              "token_prices": "inherited from existing channel cards",
              "customer_multiplier": 1}
    for gid, (cid, platform) in SCOPES.items():
        channel = call("channels", cid)
        group = call("groups", gid)
        models = sorted(set(token_models(platform)) | set(NATIVE))
        for model in models:
            assert price_for(channel["model_pricing"], platform, model) is not None
            assert price_for(group["model_pricing"], platform, model)["rate_independent"] is True
        result["groups"][str(gid)] = {"platform": platform, "channel_id": cid,
                                     "checked_models": models, "missing_pricing": [],
                                     "group_rate_unchanged": group["rate_multiplier"]}
    (out / "summary.json").write_text(json.dumps(result, indent=2))
    print(json.dumps(result))


if __name__ == "__main__":
    main()
