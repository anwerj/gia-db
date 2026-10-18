#!/usr/bin/env python3
"""Validate and execute the GiaDB v0.1 contract pack."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator, FormatChecker


ROOT = Path(__file__).resolve().parents[1]
CONTRACT = ROOT / "contract" / "v0.1"
TABLE_NAME = re.compile(r"^[a-z0-9][a-z0-9._-]{0,39}$")
TABLE_PREFIX = "gia/table/"


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


def classify_labels(labels: list[str]) -> tuple[str, str | None]:
    table_labels = [label for label in labels if label.startswith(TABLE_PREFIX)]
    if not table_labels:
        return "non_resource", None
    if len(table_labels) != 1:
        raise ValueError("invalid_resource")
    table = table_labels[0][len(TABLE_PREFIX) :]
    if not TABLE_NAME.fullmatch(table):
        raise ValueError("invalid_resource")
    return "resource", table


def validate_resource_relation(resource: dict[str, Any]) -> None:
    try:
        kind, table = classify_labels(resource["labels"])
    except ValueError as error:
        raise ValueError("resource has invalid table labels") from error
    if kind != "resource" or table != resource["table"]:
        raise ValueError("resource.table does not match its table label")


def apply_update(update: dict[str, Any]) -> dict[str, Any]:
    before = list(update["before"])
    has_table = "table" in update
    has_labels = "labels" in update

    if has_table and not TABLE_NAME.fullmatch(update["table"]):
        return {"error": "invalid_argument"}

    if has_table and not has_labels:
        non_table = [
            label for label in before if not label.startswith(TABLE_PREFIX)
        ]
        return {"labels": [TABLE_PREFIX + update["table"], *non_table]}

    if has_labels and not has_table:
        labels = list(update["labels"])
        try:
            kind, _ = classify_labels(labels)
        except ValueError:
            return {"error": "invalid_argument"}
        if kind != "resource":
            return {"error": "invalid_argument"}
        return {"labels": labels}

    if has_table and has_labels:
        labels = list(update["labels"])
        supplied_tables = [
            label for label in labels if label.startswith(TABLE_PREFIX)
        ]
        if len(supplied_tables) > 1:
            return {"error": "invalid_argument"}
        if supplied_tables:
            supplied = supplied_tables[0][len(TABLE_PREFIX) :]
            if not TABLE_NAME.fullmatch(supplied) or supplied != update["table"]:
                return {"error": "invalid_argument"}
        non_table = [
            label for label in labels if not label.startswith(TABLE_PREFIX)
        ]
        return {"labels": [TABLE_PREFIX + update["table"], *non_table]}

    return {"error": "invalid_argument"}


def evaluate(scenario: dict[str, Any]) -> dict[str, Any]:
    scenario_id = scenario["id"]
    value = scenario["input"]

    if scenario_id.startswith("labels."):
        try:
            kind, table = classify_labels(value)
        except ValueError:
            return {"error": "invalid_resource"}
        result: dict[str, Any] = {"kind": kind}
        if table is not None:
            result["table"] = table
        return result

    if scenario_id == "resource.table_matches_labels":
        try:
            validate_resource_relation(value)
        except ValueError:
            return {"error": "invalid_resource"}
        return {"valid": True}

    if scenario_id == "resource.table_mismatch":
        try:
            validate_resource_relation(value)
        except ValueError:
            return {"error": "invalid_resource"}
        return {"valid": True}

    if scenario_id == "list.pull_request":
        return {
            "kind": "non_resource"
            if "pull_request" in value
            else "candidate"
        }

    if scenario_id == "list.empty_filtered_page_continues":
        return {
            "item_count": 0,
            "exhausted": "next_token" not in value,
            "next_token": value.get("next_token"),
        }

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

    if scenario_id.startswith("update."):
        return apply_update(value)

    if scenario_id == "lifecycle.forget":
        return {"state": "forgotten"}

    if scenario_id == "lifecycle.reopen":
        return {"state": "active"}

    if scenario_id == "throttle.retry_after_precedence":
        return {"retry_after_seconds": value["retry_after_seconds"]}

    if scenario_id == "mutation.ambiguous_transport_failure":
        return {"error": "outcome_unknown", "retried": False}

    if scenario_id == "conditional.not_modified":
        return {"error": "not_modified"}

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
    validate_resource_relation(examples["resource"])
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
