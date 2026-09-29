#!/usr/bin/env python3
"""Validate the implementation binding manifest for the VRS feature cases.

This is a routing check, not an acceptance runner. A bound case names a current
Go test; it does not claim that the test has run or passed. Every case that is
not bound keeps the implementation acceptance result incomplete.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import Counter
from pathlib import Path
from typing import Any


SCENARIO_RE = re.compile(r"^Scenario(?: Outline)?:")
AT_TAG_RE = re.compile(r"^@AT-[A-Z0-9_-]+$")
TEST_FUNC_RE = re.compile(
    r"^func\s+((?:Test|Benchmark|Fuzz)[A-Za-z0-9_]*)\s*\(", re.MULTILINE
)


def feature_cases(path: Path) -> tuple[list[tuple[str, int | None]], list[str]]:
    """Return the expected (scenario ID, row ordinal) cases and parse errors."""

    lines = path.read_text(encoding="utf-8").splitlines()
    cases: list[tuple[str, int | None]] = []
    errors: list[str] = []
    pending_tags: list[str] = []
    scenario_id: str | None = None
    is_outline = False
    example_rows: list[list[str]] = []

    def finish() -> None:
        nonlocal scenario_id, is_outline, example_rows
        if scenario_id is None:
            return
        if is_outline:
            if not example_rows:
                errors.append(f"{path.name}: {scenario_id}: outline has no Examples rows")
            cases.extend((scenario_id, row) for row in range(1, len(example_rows) + 1))
        else:
            if example_rows:
                errors.append(f"{path.name}: {scenario_id}: non-outline has Examples rows")
            cases.append((scenario_id, None))
        scenario_id = None
        is_outline = False
        example_rows = []

    index = 0
    while index < len(lines):
        stripped = lines[index].strip()
        if stripped.startswith("@"):
            pending_tags.extend(stripped.split())

        scenario_match = SCENARIO_RE.match(stripped)
        if scenario_match:
            finish()
            ids = [tag[1:] for tag in pending_tags if AT_TAG_RE.fullmatch(tag)]
            if len(ids) != 1:
                errors.append(
                    f"{path.name}: {stripped}: expected one @AT- tag, found {ids or 'none'}"
                )
                scenario_id = ids[0] if ids else f"<invalid:{index + 1}>"
            else:
                scenario_id = ids[0]
            is_outline = stripped.startswith("Scenario Outline:")
            example_rows = []
            pending_tags = []
        elif scenario_id is not None and stripped == "Examples:":
            table: list[str] = []
            next_index = index + 1
            while next_index < len(lines) and lines[next_index].strip().startswith("|"):
                table.append(lines[next_index].strip())
                next_index += 1
            if not table:
                errors.append(f"{path.name}: {scenario_id}: Examples has no table")
            else:
                header = [cell.strip() for cell in table[0].strip("|").split("|")]
                if not header or not all(header):
                    errors.append(f"{path.name}: {scenario_id}: Examples header has an empty cell")
                for row in table[1:]:
                    cells = [cell.strip() for cell in row.strip("|").split("|")]
                    if len(cells) != len(header) or not all(cells):
                        errors.append(f"{path.name}: {scenario_id}: malformed Examples row {row}")
                    else:
                        example_rows.append(cells)
            index = next_index - 1
        index += 1

    finish()
    return cases, errors


def load_expected_features(root: Path) -> tuple[list[tuple[str, str, int | None]], list[str]]:
    expected: list[tuple[str, str, int | None]] = []
    errors: list[str] = []
    feature_paths = sorted((root / "features").glob("*.feature"))
    if not feature_paths:
        return [], ["features: no .feature files found"]

    seen_ids: dict[str, str] = {}
    for path in feature_paths:
        cases, parse_errors = feature_cases(path)
        errors.extend(parse_errors)
        for scenario_id, row in cases:
            if scenario_id in seen_ids and seen_ids[scenario_id] != path.name:
                errors.append(
                    f"{path.name}: duplicate scenario ID {scenario_id}; first seen in {seen_ids[scenario_id]}"
                )
            else:
                seen_ids[scenario_id] = path.name
            expected.append((path.name, scenario_id, row))
    return expected, errors


def test_names(root: Path) -> set[str]:
    names: set[str] = set()
    for path in root.rglob("*_test.go"):
        if any(part in {".git", "vendor"} for part in path.parts):
            continue
        names.update(TEST_FUNC_RE.findall(path.read_text(encoding="utf-8")))
    return names


def display_case(case: dict[str, Any]) -> str:
    row = case.get("row_ordinal")
    suffix = "" if row is None else f" row {row}"
    return f"{case.get('scenario_id', '<missing>')}{suffix}"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root",
        type=Path,
        default=Path(__file__).resolve().parents[1],
        help="repository root (default: the parent of tools/)",
    )
    parser.add_argument(
        "--manifest",
        type=Path,
        default=None,
        help="binding manifest (default: acceptance/bindings.json under --root)",
    )
    args = parser.parse_args()
    root = args.root.resolve()
    manifest_path = (args.manifest or root / "acceptance" / "bindings.json").resolve()

    errors: list[str] = []
    expected, feature_errors = load_expected_features(root)
    errors.extend(feature_errors)
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        errors.append(f"manifest: missing {manifest_path}")
        manifest = {}
    except (OSError, json.JSONDecodeError) as exc:
        errors.append(f"manifest: cannot read {manifest_path}: {exc}")
        manifest = {}

    if not isinstance(manifest, dict):
        errors.append("manifest: top-level value must be an object")
        manifest = {}
    if manifest.get("schema") != "validation.acceptance-bindings.v1":
        errors.append("manifest: unsupported or missing schema")
    spec_file = manifest.get("spec_file")
    if not isinstance(spec_file, str) or not spec_file:
        errors.append("manifest: spec_file must name the authoritative specification")
    elif not (root / spec_file).is_file():
        errors.append(f"manifest: spec_file does not exist: {spec_file}")
    if manifest.get("execution_status") != "not_run":
        errors.append("manifest: execution_status must be not_run for a binding-only manifest")

    entries = manifest.get("cases")
    if not isinstance(entries, list):
        errors.append("manifest: cases must be a list")
        entries = []

    expected_keys = set(expected)
    actual_keys: list[tuple[str, str, int | None]] = []
    actual_by_key: dict[tuple[str, str, int | None], dict[str, Any]] = {}
    known_tests = test_names(root)
    bound = 0
    unbound = 0

    for position, entry in enumerate(entries, start=1):
        if not isinstance(entry, dict):
            errors.append(f"manifest case {position}: must be an object")
            continue
        feature = entry.get("feature")
        scenario_id = entry.get("scenario_id")
        row = entry.get("row_ordinal")
        key = (feature, scenario_id, row)
        actual_keys.append(key)
        if key in actual_by_key:
            errors.append(f"manifest case {position}: duplicate {display_case(entry)}")
        else:
            actual_by_key[key] = entry
        if key not in expected_keys:
            errors.append(f"manifest case {position}: unknown feature/scenario/row {key!r}")

        status = entry.get("status")
        if status not in {"bound", "unbound"}:
            errors.append(f"manifest case {position}: status must be bound or unbound")
            continue
        if status == "bound":
            bound += 1
            test = entry.get("test")
            if not isinstance(test, str) or not test:
                errors.append(f"manifest case {position}: bound case has no test name")
            else:
                base = test.split("/", 1)[0]
                if base not in known_tests:
                    errors.append(f"manifest case {position}: Go test does not exist: {test}")
            if not isinstance(entry.get("evidence"), str) or not entry["evidence"].strip():
                errors.append(f"manifest case {position}: bound case has no evidence note")
        else:
            unbound += 1
            reason = entry.get("reason")
            if not isinstance(reason, str) or not reason.strip():
                errors.append(f"manifest case {position}: unbound case has no reason")
            if "test" in entry and entry["test"] not in (None, ""):
                errors.append(f"manifest case {position}: unbound case must not name a test")

    expected_counter = Counter(expected)
    actual_counter = Counter(actual_keys)
    for key in sorted(expected_counter.keys() - actual_counter.keys(), key=str):
        errors.append(f"manifest: missing {key!r}")
    for key in sorted(actual_counter.keys() - expected_counter.keys(), key=str):
        errors.append(f"manifest: extra {key!r}")
    for key in sorted(expected_counter.keys() & actual_counter.keys(), key=str):
        if expected_counter[key] != actual_counter[key]:
            errors.append(f"manifest: wrong multiplicity for {key!r}")

    print(f"acceptance bindings: {bound} bound, {unbound} unbound, {len(expected)} expected cases")
    if expected:
        by_feature: Counter[str] = Counter()
        for entry in entries:
            if isinstance(entry, dict) and entry.get("status") == "unbound":
                by_feature[str(entry.get("feature", "<missing>"))] += 1
        if by_feature:
            print("unbound cases by feature:")
            for feature, count in sorted(by_feature.items()):
                print(f"  {feature}: {count}")
    if errors:
        print("binding manifest: invalid", file=sys.stderr)
        for error in errors:
            print(error, file=sys.stderr)
        return 1
    if unbound:
        print(
            f"implementation acceptance: incomplete; {unbound} unbound cases are not executed",
            file=sys.stderr,
        )
        return 1
    print("binding manifest: all cases are routed; execution evidence remains not_run")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
