#!/usr/bin/env python3
"""Run the uninstrumented allocation matrix in isolated fresh processes.

Construction, input preparation and reporting happen outside each probe's
MemStats window.  A zero average is not enough: this gate checks the total
object and byte deltas for both one first call and a 1,000-call batch.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
from datetime import datetime, timezone
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]

EXPECTED_FIXTURES = {
    "scalar-signed",
    "scalar-extremes",
    "scalar-zero",
    "scalar-negative",
    "scalar-unsigned-named",
    "scalar-float",
    "scalar-float-infinity",
    "scalar-float-negative-zero",
    "scalar-comparator",
    "strings-named",
    "bytes-named",
    "time",
    "fields-flat-16",
    "fields-flat-1",
    "struct-conditions",
    "fields-deep-1",
    "fields-deep-4",
    "fields-deep-16",
    "presence-optional-absent",
    "presence-required-present",
    "presence-value-absent",
    "presence-value-present",
    "presence-nested-pointer",
    "array-pointer-project",
    "slice-size-0",
    "slice-size-1",
    "slice-size-8",
    "slice-size-64",
    "slice-size-1024",
    "slice-nested-named",
    "slice-structs",
    "membership-size-0",
    "membership-size-1",
    "membership-size-4",
    "membership-size-32",
    "membership-size-1024",
    "map-size-0",
    "map-size-1",
    "map-size-8",
    "map-size-64",
    "map-size-1024",
    "map-size-10000",
    "map-typed-integer-keys",
    "check-last-alternative",
    "generic-methods",
    "scalar-comparable",
    "presence-value-present-zero",
    "control-blank",
    "control-first",
    "control-rare",
    "control-callback",
}

CONTROL_EXPECTATIONS = {
    "control-blank": {"first": "zero", "batch": "zero"},
    "control-first": {"first": "positive", "batch": "positive"},
    "control-rare": {"first": "zero", "batch": "positive"},
    "control-callback": {"first": "positive", "batch": "positive"},
}

ALLOCATION_REPORT_SCHEMA = "validation.allocation-evidence.v1"


def run(command: list[str], *, check: bool = False) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        command,
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=check,
    )


def fail(message: str) -> None:
    print(f"allocation gate failed: {message}", file=sys.stderr)


def new_report() -> dict[str, object]:
    return {
        "schema": ALLOCATION_REPORT_SCHEMA,
        "status": "failed",
        "generated_at": datetime.now(timezone.utc).isoformat(timespec="seconds").replace(
            "+00:00", "Z"
        ),
        "catalog": [],
        "fixtures": [],
        "probes": [],
        "failures": [],
        "summary": {
            "expected_probes": 0,
            "observed_probes": 0,
            "library_fixtures": 0,
            "control_fixtures": 0,
        },
        "metadata": {"compilers": [], "platforms": []},
    }


def record_failure(report: dict[str, object], message: str) -> None:
    failures = report["failures"]
    assert isinstance(failures, list)
    failures.append(message)


def write_report(path: Path | None, report: dict[str, object]) -> bool:
    if path is None:
        return True
    try:
        path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    except (OSError, TypeError) as error:
        fail(f"cannot write JSON report {path}: {error}")
        return False
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--json-output",
        type=Path,
        help="write structured per-fixture evidence to this path",
    )
    args = parser.parse_args()
    report = new_report()

    def finish(status: str, code: int) -> int:
        report["status"] = status
        if not write_report(args.json_output, report):
            return 1
        return code

    with tempfile.TemporaryDirectory(prefix="validation-alloc-") as temporary:
        binary = Path(temporary) / "allocprobe"
        built = run(["go", "build", "-trimpath", "-o", str(binary), "./tools/allocprobe"])
        if built.returncode:
            message = "probe build failed"
            record_failure(report, message)
            fail(message)
            sys.stderr.write(built.stderr)
            return finish("failed", built.returncode)

        listed = run([str(binary), "list"])
        if listed.returncode:
            message = "fixture listing failed"
            record_failure(report, message)
            fail(message)
            sys.stderr.write(listed.stderr)
            return finish("failed", listed.returncode)
        try:
            catalog = json.loads(listed.stdout)
        except json.JSONDecodeError as error:
            message = f"fixture listing is not JSON: {error}"
            record_failure(report, message)
            fail(message)
            return finish("failed", 1)
        if not isinstance(catalog, list):
            message = "fixture listing is not an array"
            record_failure(report, message)
            fail(message)
            return finish("failed", 1)

        report["catalog"] = catalog
        fixtures: list[dict[str, object]] = []
        report["fixtures"] = fixtures

        names = [item.get("name") for item in catalog if isinstance(item, dict)]
        actual = set(names)
        if len(names) != len(actual):
            message = "fixture listing contains duplicate names"
            record_failure(report, message)
            fail(message)
            return finish("failed", 1)
        if actual != EXPECTED_FIXTURES:
            message = (
                "fixture catalogue mismatch; "
                f"missing={sorted(EXPECTED_FIXTURES - actual)} "
                f"unexpected={sorted(actual - EXPECTED_FIXTURES)}"
            )
            record_failure(report, message)
            fail(message)
            return finish("failed", 1)
        for item in catalog:
            if item.get("kind") not in {"allocation-free", "control"}:
                message = f"{item.get('name')}: unknown fixture kind {item.get('kind')!r}"
                record_failure(report, message)
                fail(message)
                return finish("failed", 1)
            if (item["name"] in CONTROL_EXPECTATIONS) != (item["kind"] == "control"):
                message = f"{item['name']}: control catalogue kind does not match expectations"
                record_failure(report, message)
                fail(message)
                return finish("failed", 1)
            fixtures.append(
                {
                    "name": item["name"],
                    "kind": item["kind"],
                    "phases": {"first": None, "batch": None},
                }
            )

        summary = report["summary"]
        assert isinstance(summary, dict)
        summary["expected_probes"] = len(catalog) * 2
        summary["library_fixtures"] = sum(
            item["kind"] == "allocation-free" for item in catalog
        )
        summary["control_fixtures"] = sum(item["kind"] == "control" for item in catalog)

        reports: list[dict[str, object]] = []
        failures: list[str] = []
        fixture_positions = {fixture["name"]: index for index, fixture in enumerate(fixtures)}
        # Each invocation is intentionally sequential and starts a fresh
        # process.  Parallel children would make runtime noise ambiguous.
        for item in catalog:
            name = item["name"]
            kind = item["kind"]
            for phase, expected_calls in (("first", 1), ("batch", 1000)):
                completed = run([str(binary), name, phase])
                label = f"{name}/{phase}"
                if completed.returncode:
                    detail = completed.stderr.strip() or f"exit {completed.returncode}"
                    message = f"{label}: probe failed: {detail}"
                    failures.append(message)
                    fixtures[fixture_positions[name]]["phases"][phase] = {
                        "status": "failed",
                        "error": detail,
                    }
                    continue
                try:
                    probe = json.loads(completed.stdout)
                except json.JSONDecodeError as error:
                    message = f"{label}: invalid JSON: {error}"
                    failures.append(message)
                    fixtures[fixture_positions[name]]["phases"][phase] = {
                        "status": "failed",
                        "error": str(error),
                    }
                    continue
                if not isinstance(probe, dict):
                    message = f"{label}: probe JSON is not an object"
                    failures.append(message)
                    fixtures[fixture_positions[name]]["phases"][phase] = {
                        "status": "failed",
                        "error": message,
                    }
                    continue
                reports.append(probe)
                report_entry = dict(probe)
                report_entry["status"] = "passed"
                fixtures[fixture_positions[name]]["phases"][phase] = report_entry
                failure_count = len(failures)
                if probe.get("fixture") != name:
                    failures.append(f"{label}: report fixture={probe.get('fixture')!r}")
                if probe.get("kind") != kind:
                    failures.append(f"{label}: report kind={probe.get('kind')!r}")
                if probe.get("phase") != phase:
                    failures.append(f"{label}: report phase={probe.get('phase')!r}")
                if probe.get("calls") != expected_calls:
                    failures.append(f"{label}: calls={probe.get('calls')!r}, want {expected_calls}")
                if not probe.get("go") or not probe.get("os") or not probe.get("arch"):
                    failures.append(f"{label}: missing toolchain/platform metadata")

                mallocs = probe.get("mallocs")
                bytes_allocated = probe.get("bytes")
                if not isinstance(mallocs, int) or not isinstance(bytes_allocated, int):
                    failures.append(f"{label}: non-integer allocation deltas")
                    report_entry["status"] = "failed"
                    continue
                if kind == "allocation-free":
                    if mallocs != 0 or bytes_allocated != 0:
                        failures.append(
                            f"{label}: library allocation delta mallocs={mallocs} bytes={bytes_allocated}"
                        )
                    if len(failures) != failure_count:
                        report_entry["status"] = "failed"
                    continue
                expectation = CONTROL_EXPECTATIONS.get(name, {}).get(phase)
                if expectation == "zero" and (mallocs != 0 or bytes_allocated != 0):
                    failures.append(
                        f"{label}: control should be quiet before its trigger, "
                        f"mallocs={mallocs} bytes={bytes_allocated}"
                    )
                elif expectation == "positive" and (mallocs <= 0 or bytes_allocated <= 0):
                    failures.append(
                        f"{label}: deliberate allocation was not observed, "
                        f"mallocs={mallocs} bytes={bytes_allocated}"
                    )
                if len(failures) != failure_count:
                    report_entry["status"] = "failed"

        if failures:
            fail(f"{len(failures)} assertion(s)")
            for failure in failures:
                print(f"  {failure}", file=sys.stderr)
            report["probes"] = reports
            report["failures"] = failures
            summary["observed_probes"] = len(reports)
            report["metadata"] = {
                "compilers": sorted({str(item.get("go")) for item in reports if item.get("go")}),
                "platforms": sorted(
                    {
                        f"{item.get('os')}/{item.get('arch')}"
                        for item in reports
                        if item.get("os") and item.get("arch")
                    }
                ),
            }
            return finish("failed", 1)

        compiler = {str(probe["go"]) for probe in reports}
        platforms = {f"{probe['os']}/{probe['arch']}" for probe in reports}
        report["probes"] = reports
        report["failures"] = []
        summary["observed_probes"] = len(reports)
        report["metadata"] = {
            "compilers": sorted(compiler),
            "platforms": sorted(platforms),
        }
        print(
            "allocation gate passed: "
            f"{len(catalog) - len(CONTROL_EXPECTATIONS)} library fixtures + "
            f"{len(CONTROL_EXPECTATIONS)} controls, "
            f"{len(reports)} fresh-process probes, "
            f"phases=first(1),batch(1000), "
            f"compiler={','.join(sorted(compiler))}, "
            f"platform={','.join(sorted(platforms))}"
        )
        print("library matrix:")
        for item in catalog:
            if item["kind"] == "allocation-free":
                print(f"  {item['name']}")
        print("controls:")
        for probe in reports:
            if probe["kind"] == "control":
                print(
                    f"  {probe['fixture']}/{probe['phase']}: "
                    f"mallocs={probe['mallocs']} bytes={probe['bytes']}"
                )
        return finish("passed", 0)


if __name__ == "__main__":
    raise SystemExit(main())
