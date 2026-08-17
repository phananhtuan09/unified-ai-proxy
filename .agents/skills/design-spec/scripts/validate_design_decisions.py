#!/usr/bin/env python3
"""Validate a design decision manifest against the design plan it was approved from."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from datetime import datetime
from pathlib import Path
from typing import Any


SHA256 = re.compile(r"^[0-9a-f]{64}$")
KEBAB_CASE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
PREVIEW_KINDS = {"ui", "api", "full-stack", "workflow", "data", "generic"}
RULE_SOURCES = {"agent-proposed-not-objected"}
CONSTRAINT_SOURCES = {"agent-proposed-not-objected", "human-request", "approved-upstream"}


def fail(message: str) -> None:
    raise ValueError(message)


def require_string(value: Any, field: str) -> str:
    if not isinstance(value, str) or not value.strip():
        fail(f"{field} must be a non-empty string")
    return value.strip()


def require_string_list(value: Any, field: str) -> list[str]:
    if not isinstance(value, list):
        fail(f"{field} must be an array")
    return [require_string(item, f"{field}[{index}]") for index, item in enumerate(value)]


def require_object_list(value: Any, field: str) -> list[dict[str, Any]]:
    if not isinstance(value, list):
        fail(f"{field} must be an array")
    for index, item in enumerate(value):
        if not isinstance(item, dict):
            fail(f"{field}[{index}] must be an object")
    return value


def parse_timestamp(value: Any, field: str) -> str:
    timestamp = require_string(value, field)
    normalized = timestamp[:-1] + "+00:00" if timestamp.endswith("Z") else timestamp
    try:
        datetime.fromisoformat(normalized)
    except ValueError as error:
        fail(f"{field} must be an ISO-8601 timestamp: {error}")
    return timestamp


def load_json(path: Path, label: str) -> dict[str, Any]:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        fail(f"{label} does not exist: {path}")
    except json.JSONDecodeError as error:
        fail(f"{label} is not valid JSON: {error}")
    if not isinstance(data, dict):
        fail(f"{label} root must be an object")
    return data


def resolve_repo_path(repo_root: Path, value: str, field: str) -> Path:
    candidate = Path(value)
    if candidate.is_absolute():
        fail(f"{field} must be repository-relative")
    resolved_root = repo_root.resolve()
    resolved = (resolved_root / candidate).resolve()
    if resolved != resolved_root and resolved_root not in resolved.parents:
        fail(f"{field} escapes the repository root")
    return resolved


def compare_identified(
    manifest_items: list[dict[str, Any]],
    plan_items: list[dict[str, Any]],
    field: str,
    text_key: str,
) -> None:
    plan_map = {item["id"]: item[text_key] for item in plan_items}
    manifest_map: dict[str, str] = {}
    for index, item in enumerate(manifest_items):
        item_id = require_string(item.get("id"), f"{field}[{index}].id")
        if item_id in manifest_map:
            fail(f"duplicate id in {field}: {item_id}")
        manifest_map[item_id] = require_string(item.get(text_key), f"{field}[{index}].{text_key}")
    if manifest_map.keys() != plan_map.keys():
        missing = sorted(plan_map.keys() - manifest_map.keys())
        extra = sorted(manifest_map.keys() - plan_map.keys())
        fail(f"{field} ids do not match the design plan: missing {missing}, unknown {extra}")
    for item_id, text in manifest_map.items():
        if text != plan_map[item_id]:
            fail(f"{field} {item_id} {text_key} does not match the design plan")


def validate_manifest(data: dict[str, Any], repo_root: Path, plan_override: Path | None) -> dict[str, Any]:
    if data.get("schema_version") != 3:
        fail("schema_version must equal 3")

    feature_slug = require_string(data.get("feature_slug"), "feature_slug")
    if not KEBAB_CASE.fullmatch(feature_slug):
        fail("feature_slug must be kebab-case")
    design_revision = require_string(str(data.get("design_revision") or ""), "design_revision")

    design_path_value = require_string(data.get("design_path"), "design_path")
    if not design_path_value.endswith(".json"):
        fail("design_path must point to the design plan JSON")
    design_path = plan_override.resolve() if plan_override else resolve_repo_path(repo_root, design_path_value, "design_path")
    if not design_path.is_file():
        fail(f"design plan does not exist: {design_path}")

    plan = load_json(design_path, "design plan")
    if plan.get("feature_slug") != feature_slug:
        fail(f"feature_slug mismatch: manifest {feature_slug}, plan {plan.get('feature_slug')}")
    if str(plan.get("design_revision") or "") != design_revision:
        fail(f"design_revision mismatch: manifest {design_revision}, plan {plan.get('design_revision')}")

    expected_sha = require_string(data.get("design_sha256"), "design_sha256")
    if not SHA256.fullmatch(expected_sha):
        fail("design_sha256 must contain 64 lowercase hexadecimal characters")
    actual_sha = hashlib.sha256(design_path.read_bytes()).hexdigest()
    if actual_sha != expected_sha:
        fail(f"design_sha256 mismatch: expected {expected_sha}, actual {actual_sha}")

    parse_timestamp(data.get("approved_at"), "approved_at")
    if data.get("approval_source") != "local-runner":
        fail("approval_source must equal 'local-runner'")
    if data.get("approval_meaning") != "direction-approved":
        fail("approval_meaning must equal 'direction-approved'")

    if require_string(data.get("goal"), "goal") != require_string(plan.get("goal"), "plan goal"):
        fail("goal does not match the design plan")

    preview = data.get("output_preview")
    if not isinstance(preview, dict):
        fail("output_preview must be an object")
    if preview.get("kind") not in PREVIEW_KINDS:
        fail(f"output_preview.kind must be one of {sorted(PREVIEW_KINDS)}")
    require_string(preview.get("summary"), "output_preview.summary")

    scope = data.get("scope")
    if not isinstance(scope, dict):
        fail("scope must be an object")
    require_string_list(scope.get("in"), "scope.in")
    require_string_list(scope.get("out"), "scope.out")

    decisions = require_object_list(data.get("decisions"), "decisions")
    if not decisions:
        fail("decisions must be a non-empty array")
    plan_decisions = plan.get("decisions") or []
    compare_identified(decisions, plan_decisions, "decisions", "question")
    options = {
        decision["id"]: {option["answer"] for option in decision.get("options") or []}
        for decision in plan_decisions
    }
    for index, decision in enumerate(decisions):
        decision_id = decision["id"]
        answer = require_string(decision.get("answer"), f"decisions[{index}].answer")
        if answer not in options[decision_id]:
            fail(f"decisions[{index}].answer is not an option offered by the design plan for {decision_id}")
        if decision.get("source") != "human":
            fail(f"decisions[{index}].source must equal 'human'")

    rules = require_object_list(data.get("business_rules") or [], "business_rules")
    compare_identified(rules, plan.get("business_rules") or [], "business_rules", "statement")
    for index, rule in enumerate(rules):
        if rule.get("source") not in RULE_SOURCES:
            fail(f"business_rules[{index}].source must be one of {sorted(RULE_SOURCES)}")

    constraints = require_object_list(data.get("implementation_constraints") or [], "implementation_constraints")
    compare_identified(
        constraints, plan.get("implementation_constraints") or [], "implementation_constraints", "statement"
    )
    for index, constraint in enumerate(constraints):
        if constraint.get("source") not in CONSTRAINT_SOURCES:
            fail(f"implementation_constraints[{index}].source must be one of {sorted(CONSTRAINT_SOURCES)}")

    require_string_list(data.get("ai_discretion") or [], "ai_discretion")
    for index, assumption in enumerate(require_object_list(data.get("assumptions") or [], "assumptions")):
        require_string(assumption.get("id"), f"assumptions[{index}].id")
        require_string(assumption.get("statement"), f"assumptions[{index}].statement")
    for index, risk in enumerate(require_object_list(data.get("risks") or [], "risks")):
        require_string(risk.get("id"), f"risks[{index}].id")
        require_string(risk.get("statement"), f"risks[{index}].statement")
    for index, item in enumerate(require_object_list(data.get("evidence") or [], "evidence")):
        require_string(item.get("path"), f"evidence[{index}].path")
        require_string(item.get("observation"), f"evidence[{index}].observation")

    require_string_list(data.get("constraints"), "constraints")
    require_string_list(data.get("unresolved_non_blocking"), "unresolved_non_blocking")

    return {
        "feature_slug": feature_slug,
        "design_revision": design_revision,
        "design_path": design_path_value,
        "decision_count": len(decisions),
        "business_rule_count": len(rules),
        "constraint_count": len(constraints),
        "design_sha256": actual_sha,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path, help="Path to the decision manifest JSON")
    parser.add_argument("--repo-root", type=Path, default=Path.cwd(), help="Repository root used for relative paths")
    parser.add_argument("--plan", type=Path, help="Optional explicit design plan path for checksum validation")
    args = parser.parse_args()

    try:
        result = validate_manifest(load_json(args.manifest, "manifest"), args.repo_root, args.plan)
    except ValueError as error:
        print(f"invalid: {error}", file=sys.stderr)
        return 1

    print(json.dumps({"valid": True, **result}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
