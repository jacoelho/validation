#!/usr/bin/env python3
"""Check feature copies and requirement/scenario routing.

This checks document structure only. It does not execute acceptance scenarios.
"""
from pathlib import Path
from collections import Counter
import re
import sys

root = Path(__file__).resolve().parents[1]
spec = (root / "docs/validation-refactor-spec.md").read_text()
body, appendix = spec.split("## Appendix A. Complete Gherkin acceptance suite", 1)
requirement_ids = re.findall(r"^#### (REQ-[A-Z0-9-]+) ", body, re.M)
requirements = set(requirement_ids)
blocks = re.findall(r"### (\d\d_[a-z_]+\.feature)\n\n```gherkin\n(.*?)\n```", appendix, re.S)
categories = {"behaviour", "compile", "source", "allocation", "benchmark", "race", "fuzz", "traceability", "release"}
errors = []
for identifier, count in Counter(requirement_ids).items():
    if count != 1:
        errors.append(f"duplicate requirement {identifier}: {count} declarations")
for name, count in Counter(name for name, _ in blocks).items():
    if count != 1:
        errors.append(f"duplicate feature block {name}: {count} copies")
seen = set()
covered = set()
if len(blocks) != 16:
    errors.append(f"expected 16 feature blocks, found {len(blocks)}")
for name, text in blocks:
    path = root / "features" / name
    if not path.exists() or path.read_text() != text + "\n":
        errors.append(f"{name}: missing or differs from spec")
    lines = text.splitlines()
    pending = []
    outline = None
    examples = None
    for line in lines:
        stripped = line.strip()
        if stripped.startswith("@"):
            pending.extend(stripped.split())
        elif stripped.startswith("Scenario:") or stripped.startswith("Scenario Outline:"):
            if outline is not None and (examples is None or len(examples) < 2):
                errors.append(f"{name}: outline {outline} has no complete Examples rows")
            outline = stripped if stripped.startswith("Scenario Outline:") else None
            examples = None
            scenarios = [tag for tag in pending if tag.startswith("@AT-")]
            links = [tag[1:] for tag in pending if tag.startswith("@REQ-")]
            kinds = [tag[1:] for tag in pending if tag[1:] in categories]
            if len(scenarios) != 1 or not links or not kinds:
                errors.append(f"{name}: incomplete tags at {stripped}")
            for scenario in scenarios:
                if scenario in seen:
                    errors.append(f"duplicate scenario {scenario}")
                seen.add(scenario)
            for link in links:
                if link not in requirements:
                    errors.append(f"{name}: unknown requirement {link}")
                covered.add(link)
            pending = []
        elif stripped == "Examples:" and outline is not None:
            examples = []
        elif stripped.startswith("|") and outline is not None and examples is not None:
            cells = [cell.strip() for cell in stripped.strip("|").split("|")]
            if not all(cells):
                errors.append(f"{name}: empty Examples cell in {outline}")
            examples.append(cells)
            if len(examples) > 1 and len(cells) != len(examples[0]):
                errors.append(f"{name}: incomplete Examples row in {outline}")
        elif stripped.startswith("Feature:"):
            pending = []
        elif stripped and not stripped.startswith("#") and not stripped.startswith("Background:"):
            # Tag lines may precede a Scenario after whitespace/comments only.
            if pending and not stripped.startswith("Scenario"):
                errors.append(f"{name}: tags separated from scenario by {stripped}")
                pending = []
    if outline is not None and (examples is None or len(examples) < 2):
        errors.append(f"{name}: outline {outline} has no complete Examples rows")
for missing in sorted(requirements - covered):
    errors.append(f"unlinked requirement {missing}")
for error in errors:
    print(error, file=sys.stderr)
print(f"{len(requirements)} requirements, {len(seen)} scenarios, {len(blocks)} feature copies; implementation acceptance not run")
sys.exit(bool(errors))
