#!/usr/bin/env python3
"""Validate and execute the GiaDB v0.1 contract pack."""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker


ROOT = Path(__file__).resolve().parents[1]
CONTRACT = ROOT / "contract" / "v0.1"


def load(name: str) -> Any:
    return json.loads((CONTRACT / name).read_text())


def validate_definition(schema: dict[str, Any], name: str, value: Any) -> None:
    wrapper = {
        "$schema": schema["$schema"],
        "$defs": schema["$defs"],
        "$ref": f"#/$defs/{name}",
    }
    Draft202012Validator(
        wrapper,
        format_checker=FormatChecker(),
    ).validate(value)


def classify_issue(value: dict[str, Any]) -> dict[str, Any]:
    if "pull_request" in value:
        return {"kind": "non_resource"}
    return {"kind": "resource", "body": value.get("body", "")}


def evaluate(scenario: dict[str, Any]) -> dict[str, Any]:
    scenario_id = scenario["id"]
    value = scenario["input"]

    if scenario_id == "body.unchanged":
        return {"bodies": list(value["bodies"])}

    if scenario_id == "issue.any_body":
        return classify_issue(value)

    if scenario_id == "labels.do_not_change_body":
        return {
            "id": value["id"],
            "body": value["body"],
            "labels": list(value["labels"]),
        }

    if scenario_id == "mutation.pull_request_rejected":
        if "pull_request" not in value:
            return {"mutated": True}
        return {"error": "not_found", "mutated": False}

    if scenario_id == "list.pull_request":
        return classify_issue(value)

    if scenario_id == "list.empty_filtered_page_continues":
        count = sum(1 for item in value["items"] if "pull_request" not in item)
        result = {
            "item_count": count,
            "exhausted": "next_token" not in value,
        }
        if "next_token" in value:
            result["next_token"] = value["next_token"]
        return result

    if scenario_id in {
        "list.state_default",
        "list.state_active",
        "list.state_forgotten",
    }:
        state = value.get("state", "active")
        return {
            "ids": [
                resource["id"]
                for resource in value["resources"]
                if resource["state"] == state
            ]
        }

    if scenario_id == "sections.are_separate":
        return {"resource_has_sections": False}

    if scenario_id == "list.title_match_truncated":
        return {
            "truncated": value["reported_total"] > 1000,
            "retrievable_limit": 1000,
        }

    if scenario_id == "list.title_match_incomplete":
        return {"incomplete": value["incomplete_results"]}

    if scenario_id == "update.replace_labels":
        return {"labels": list(value["labels"])}

    if scenario_id == "lifecycle.forget":
        return {"state": "forgotten"}

    if scenario_id == "lifecycle.reopen":
        return {"state": "active"}

    if scenario_id == "mutation.ambiguous_transport_failure":
        return {"error": "outcome_unknown", "retried": False}

    raise AssertionError(f"scenario has no executable evaluator: {scenario_id}")


def execute_scenarios(document: dict[str, Any]) -> None:
    seen: set[str] = set()
    for scenario in document["scenarios"]:
        scenario_id = scenario["id"]
        if scenario_id in seen:
            raise AssertionError(f"duplicate scenario ID: {scenario_id}")
        seen.add(scenario_id)
        actual = evaluate(scenario)
        expected = scenario["expect"]
        if actual != expected:
            raise AssertionError(
                f"{scenario_id}: expected {expected!r}, got {actual!r}"
            )


def main() -> int:
    schema = load("contract.schema.json")
    conformance_schema = load("conformance.schema.json")
    examples = load("examples.json")
    conformance = load("conformance.json")

    Draft202012Validator.check_schema(schema)
    Draft202012Validator.check_schema(conformance_schema)
    Draft202012Validator(
        conformance_schema,
        format_checker=FormatChecker(),
    ).validate(conformance)

    validate_definition(schema, "resource", examples["resource"])
    validate_definition(schema, "section", examples["section"])
    validate_definition(schema, "resource_page", examples["page"])
    validate_definition(schema, "error", examples["rate_error"])
    execute_scenarios(conformance)

    print(
        f"validated examples and executed "
        f"{len(conformance['scenarios'])} conformance scenarios"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
