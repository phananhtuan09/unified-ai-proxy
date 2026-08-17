#!/usr/bin/env python3
"""Validate a design plan JSON document before serving it to the human reviewer."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any


KEBAB_CASE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
ID_PATTERNS = {
    "decisions": re.compile(r"^D-\d{3}$"),
    "business_rules": re.compile(r"^BR-\d{3}$"),
    "implementation_constraints": re.compile(r"^IC-\d{3}$"),
    "assumptions": re.compile(r"^A-\d{3}$"),
    "risks": re.compile(r"^R-\d{3}$"),
}
PREVIEW_KINDS = {"ui", "api", "full-stack", "workflow", "data", "generic"}
CONSTRAINT_SOURCES = {"agent", "human-request", "approved-upstream"}

MAX_DECISIONS = 7
MIN_DECISIONS = 1
MAX_BUSINESS_RULES = 6
MAX_CONSTRAINTS = 8
MAX_DISCRETION = 8
MAX_ASSUMPTIONS = 5
MAX_RISKS = 5
MAX_FLOW = 6
MAX_NODE_DEPTH = 4

NODE_FIELDS = {
    "titlebar": ("label",),
    "columns": ("columns",),
    "box": (),
    "tabs": ("items",),
    "field": ("placeholder",),
    "grid": ("cells",),
    "rows": ("rows",),
    "total": ("label",),
    "actions": ("buttons",),
    "text": ("text",),
}


class PlanError(ValueError):
    pass


def fail(message: str) -> None:
    raise PlanError(message)


def require_string(value: Any, field: str) -> str:
    if not isinstance(value, str) or not value.strip():
        fail(f"{field} must be a non-empty string")
    return value.strip()


def require_string_list(value: Any, field: str, minimum: int = 0, maximum: int | None = None) -> list[str]:
    if not isinstance(value, list):
        fail(f"{field} must be an array")
    if len(value) < minimum:
        fail(f"{field} must contain at least {minimum} item(s)")
    if maximum is not None and len(value) > maximum:
        fail(f"{field} must contain no more than {maximum} item(s)")
    return [require_string(item, f"{field}[{index}]") for index, item in enumerate(value)]


def require_object_list(value: Any, field: str, minimum: int = 0, maximum: int | None = None) -> list[dict[str, Any]]:
    if not isinstance(value, list):
        fail(f"{field} must be an array")
    if len(value) < minimum:
        fail(f"{field} must contain at least {minimum} item(s)")
    if maximum is not None and len(value) > maximum:
        fail(f"{field} must contain no more than {maximum} item(s)")
    for index, item in enumerate(value):
        if not isinstance(item, dict):
            fail(f"{field}[{index}] must be an object")
    return value


def validate_node(node: Any, field: str, depth: int) -> None:
    if depth > MAX_NODE_DEPTH:
        fail(f"{field} nests deeper than {MAX_NODE_DEPTH} levels")
    if not isinstance(node, dict):
        fail(f"{field} must be an object")
    node_type = require_string(node.get("type"), f"{field}.type")
    if node_type not in NODE_FIELDS:
        fail(f"{field}.type must be one of {sorted(NODE_FIELDS)}")
    for required in NODE_FIELDS[node_type]:
        if node.get(required) in (None, "", []):
            fail(f"{field}.{required} is required for node type '{node_type}'")

    if node_type == "columns":
        columns = node.get("columns")
        if not isinstance(columns, list) or not columns:
            fail(f"{field}.columns must be a non-empty array")
        for index, column in enumerate(columns):
            if not isinstance(column, list):
                fail(f"{field}.columns[{index}] must be an array of nodes")
            for child_index, child in enumerate(column):
                validate_node(child, f"{field}.columns[{index}][{child_index}]", depth + 1)
    elif node_type == "box":
        for index, child in enumerate(node.get("children") or []):
            validate_node(child, f"{field}.children[{index}]", depth + 1)
    elif node_type == "tabs":
        for index, item in enumerate(node.get("items") or []):
            if not isinstance(item, dict):
                fail(f"{field}.items[{index}] must be an object")
            require_string(item.get("label"), f"{field}.items[{index}].label")
    elif node_type == "grid":
        for index, cell in enumerate(node.get("cells") or []):
            if not isinstance(cell, dict):
                fail(f"{field}.cells[{index}] must be an object")
            require_string_list(cell.get("lines"), f"{field}.cells[{index}].lines", minimum=1)
    elif node_type == "rows":
        for index, row in enumerate(node.get("rows") or []):
            if not isinstance(row, dict):
                fail(f"{field}.rows[{index}] must be an object")
            require_string(row.get("label"), f"{field}.rows[{index}].label")
    elif node_type == "actions":
        require_string_list(node.get("buttons"), f"{field}.buttons", minimum=1)


def validate_preview(preview: Any, warnings: list[str]) -> None:
    if not isinstance(preview, dict):
        fail("preview must be an object")
    kind = preview.get("kind")
    if kind not in PREVIEW_KINDS:
        fail(f"preview.kind must be one of {sorted(PREVIEW_KINDS)}")
    require_string(preview.get("summary"), "preview.summary")

    if kind == "ui":
        screens = require_object_list(preview.get("screens"), "preview.screens", minimum=1, maximum=3)
        for index, screen in enumerate(screens):
            require_string(screen.get("caption"), f"preview.screens[{index}].caption")
            blocks = screen.get("blocks")
            if not isinstance(blocks, list) or not blocks:
                fail(f"preview.screens[{index}].blocks must be a non-empty array")
            for block_index, block in enumerate(blocks):
                validate_node(block, f"preview.screens[{index}].blocks[{block_index}]", 1)
    elif preview.get("exchange") is None and preview.get("shape") is None:
        warnings.append(f"preview.kind '{kind}' has neither an exchange nor a shape block to show")

    for index, state in enumerate(preview.get("states") or []):
        if not isinstance(state, dict):
            fail(f"preview.states[{index}] must be an object")
        require_string(state.get("name"), f"preview.states[{index}].name")
        require_string(state.get("detail"), f"preview.states[{index}].detail")

    flow = require_object_list(preview.get("flow"), "preview.flow", minimum=2, maximum=MAX_FLOW)
    for index, step in enumerate(flow):
        require_string(step.get("step"), f"preview.flow[{index}].step")
        require_string_list(step.get("branches") or [], f"preview.flow[{index}].branches")


def validate_identified(
    plan: dict[str, Any],
    field: str,
    minimum: int,
    maximum: int,
    seen: dict[str, str],
) -> list[dict[str, Any]]:
    items = require_object_list(plan.get(field) or [], field, minimum=minimum, maximum=maximum)
    pattern = ID_PATTERNS[field]
    for index, item in enumerate(items):
        item_id = require_string(item.get("id"), f"{field}[{index}].id")
        if not pattern.fullmatch(item_id):
            fail(f"{field}[{index}].id must match {pattern.pattern}")
        if item_id in seen:
            fail(f"duplicate id '{item_id}' used by both {seen[item_id]} and {field}")
        seen[item_id] = field
    return items


def validate_plan(path: Path, repo_root: Path | None = None) -> dict[str, Any]:
    try:
        plan = json.loads(path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        fail(f"design plan does not exist: {path}")
    except json.JSONDecodeError as error:
        fail(f"design plan is not valid JSON: {error}")
    if not isinstance(plan, dict):
        fail("design plan root must be an object")

    warnings: list[str] = []

    if plan.get("schema_version") != 3:
        fail("schema_version must equal 3")

    feature_slug = require_string(plan.get("feature_slug"), "feature_slug")
    if not KEBAB_CASE.fullmatch(feature_slug):
        fail("feature_slug must be kebab-case")
    if path.stem != feature_slug:
        fail(f"file name must match feature_slug: expected {feature_slug}.json, got {path.name}")

    require_string(str(plan.get("design_revision") or ""), "design_revision")
    require_string(plan.get("title"), "title")
    require_string(plan.get("goal"), "goal")
    require_string(plan.get("target_user"), "target_user")
    require_string(plan.get("current_problem"), "current_problem")

    validate_preview(plan.get("preview"), warnings)

    scope = plan.get("scope")
    if not isinstance(scope, dict):
        fail("scope must be an object")
    require_string_list(scope.get("in"), "scope.in", minimum=1, maximum=5)
    require_string_list(scope.get("out"), "scope.out", minimum=1, maximum=5)

    seen: dict[str, str] = {}

    decisions = validate_identified(plan, "decisions", MIN_DECISIONS, MAX_DECISIONS, seen)
    for index, decision in enumerate(decisions):
        require_string(decision.get("question"), f"decisions[{index}].question")
        require_string(decision.get("why"), f"decisions[{index}].why")
        options = require_object_list(decision.get("options"), f"decisions[{index}].options", minimum=2, maximum=4)
        answers: set[str] = set()
        recommended = 0
        for option_index, option in enumerate(options):
            answer = require_string(option.get("answer"), f"decisions[{index}].options[{option_index}].answer")
            if answer in answers:
                fail(f"decisions[{index}] has duplicate option answers")
            answers.add(answer)
            require_string(option.get("tradeoff"), f"decisions[{index}].options[{option_index}].tradeoff")
            if option.get("recommended") is True:
                recommended += 1
        if recommended > 1:
            fail(f"decisions[{index}] marks more than one option as recommended")

    rules = validate_identified(plan, "business_rules", 0, MAX_BUSINESS_RULES, seen)
    for index, rule in enumerate(rules):
        require_string(rule.get("statement"), f"business_rules[{index}].statement")

    constraints = validate_identified(plan, "implementation_constraints", 0, MAX_CONSTRAINTS, seen)
    for index, constraint in enumerate(constraints):
        require_string(constraint.get("statement"), f"implementation_constraints[{index}].statement")
        source = constraint.get("source", "agent")
        if source not in CONSTRAINT_SOURCES:
            fail(f"implementation_constraints[{index}].source must be one of {sorted(CONSTRAINT_SOURCES)}")

    require_string_list(plan.get("ai_discretion") or [], "ai_discretion", maximum=MAX_DISCRETION)

    assumptions = validate_identified(plan, "assumptions", 0, MAX_ASSUMPTIONS, seen)
    for index, assumption in enumerate(assumptions):
        require_string(assumption.get("statement"), f"assumptions[{index}].statement")
        require_string(assumption.get("impact_if_wrong"), f"assumptions[{index}].impact_if_wrong")

    risks = validate_identified(plan, "risks", 0, MAX_RISKS, seen)
    for index, risk in enumerate(risks):
        require_string(risk.get("statement"), f"risks[{index}].statement")
        require_string(risk.get("consequence"), f"risks[{index}].consequence")

    evidence = require_object_list(plan.get("evidence"), "evidence", minimum=1)
    illustrative = 0
    for index, item in enumerate(evidence):
        evidence_path = require_string(item.get("path"), f"evidence[{index}].path")
        require_string(item.get("observation"), f"evidence[{index}].observation")
        if item.get("illustrative") is True:
            illustrative += 1
            continue
        if repo_root is not None:
            candidate = Path(evidence_path)
            if candidate.is_absolute():
                fail(f"evidence[{index}].path must be repository-relative")
            if not (repo_root / candidate).exists():
                fail(f"evidence[{index}].path does not exist in the repository: {evidence_path}")
    if illustrative:
        warnings.append(f"{illustrative} evidence entries are marked illustrative and were not checked against the repository")

    if len(decisions) > 6:
        warnings.append(f"{len(decisions)} required decisions is above the comfortable review size of six")

    return {
        "valid": True,
        "warnings": warnings,
        "feature_slug": feature_slug,
        "design_revision": str(plan["design_revision"]),
        "preview_kind": plan["preview"]["kind"],
        "decision_count": len(decisions),
        "business_rule_count": len(rules),
        "constraint_count": len(constraints),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("plan_path", type=Path)
    parser.add_argument("--repo-root", type=Path, help="Repository root used to verify evidence paths")
    args = parser.parse_args()
    try:
        result = validate_plan(args.plan_path, args.repo_root.resolve() if args.repo_root else None)
    except (OSError, PlanError) as error:
        print(f"invalid: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
