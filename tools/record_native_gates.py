#!/usr/bin/env python3
"""Run the required native Go 1.27 verification gates and record one report.

The report is deliberately useful when a gate fails or times out.  Every gate
has a stable name and a planned command, and the command result is recorded
even when a later gate cannot be started.  ``--quick`` runs only the cheap
specification and correctness smoke checks; it is a report-generation aid,
not a release result.
"""

from __future__ import annotations

import argparse
from collections import Counter
import hashlib
import json
import os
import platform
import re
import shlex
import signal
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable


ROOT = Path(__file__).resolve().parents[1]
SCHEMA = "validation.native-gates.v1"
ALLOCATION_SCHEMA = "validation.allocation-evidence.v1"
REQUIRED_GO_VERSION = "go1.27.0"
DEFAULT_TIMEOUT_SECONDS = 600.0
DEFAULT_TOTAL_TIMEOUT_SECONDS = 1800.0
MAX_OUTPUT_TAIL = 16 * 1024

BASE_GATES = ("spec", "correctness", "allocation", "alloc_tests", "benchmark")
LINUX_AMD64_GATES = (
    "staticcheck",
    "race",
    "fuzz_paths",
    "fuzz_numeric",
    "fuzz_slices",
    "fuzz_maps",
)
FUZZ_TARGETS = {
    "fuzz_paths": "FuzzPaths",
    "fuzz_numeric": "FuzzNumericBounds",
    "fuzz_slices": "FuzzSliceOccurrences",
    "fuzz_maps": "FuzzMapIssueOrder",
}
MANDATED_BENCHMARK_CATEGORIES = (
    "construction",
    "valid_evaluation",
    "failing_evaluation",
    "error_traversal",
    "formatting",
)
BENCHMARK_METRICS = ("ns/op", "B/op", "allocs/op", "input-items")


def now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace(
        "+00:00", "Z"
    )


def command_text(command: list[str]) -> str:
    return shlex.join(command)


def tail(value: str) -> str:
    if len(value) <= MAX_OUTPUT_TAIL:
        return value
    return "...<truncated>...\n" + value[-MAX_OUTPUT_TAIL:]


def text_output(value: str | bytes | None) -> str:
    if value is None:
        return ""
    if isinstance(value, bytes):
        return value.decode("utf-8", errors="replace")
    return value


def run_command(
    command: list[str], *, cwd: Path, timeout_seconds: float
) -> dict[str, Any]:
    """Run one command with a hard timeout and return serialisable evidence."""

    started_at = now()
    started = time.monotonic()
    stdout = ""
    stderr = ""
    timed_out = False
    exit_code: int | None = None
    spawn_error: str | None = None
    process: subprocess.Popen[str] | None = None
    try:
        process = subprocess.Popen(
            command,
            cwd=cwd,
            text=True,
            encoding="utf-8",
            errors="replace",
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            start_new_session=os.name == "posix",
        )
        try:
            stdout, stderr = process.communicate(timeout=timeout_seconds)
            exit_code = process.returncode
        except subprocess.TimeoutExpired as error:
            timed_out = True
            stdout = text_output(error.stdout)
            stderr = text_output(error.stderr)
            if process.poll() is None:
                if os.name == "posix":
                    try:
                        os.killpg(process.pid, signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                else:
                    process.kill()
            remaining_stdout, remaining_stderr = process.communicate()
            stdout += text_output(remaining_stdout)
            stderr += text_output(remaining_stderr)
    except OSError as error:
        spawn_error = str(error)
        stderr = spawn_error
    duration_seconds = round(time.monotonic() - started, 6)
    finished_at = now()
    if timed_out:
        status = "timed_out"
        exit_code = None
    elif spawn_error is not None or exit_code not in (0, None):
        status = "failed"
    else:
        status = "passed"
    result: dict[str, Any] = {
        "command": command,
        "command_text": command_text(command),
        "started_at": started_at,
        "finished_at": finished_at,
        "duration_seconds": duration_seconds,
        "exit_code": exit_code,
        "timed_out": timed_out,
        "status": status,
        "stdout_tail": tail(stdout),
        "stderr_tail": tail(stderr),
    }
    if spawn_error is not None:
        result["spawn_error"] = spawn_error
    # These private fields are consumed by the benchmark/allocation adapters
    # and removed before the report is serialised.
    result["_stdout"] = stdout
    result["_stderr"] = stderr
    return result


def not_run(command: list[str], reason: str, *, status: str = "not_run") -> dict[str, Any]:
    timestamp = now()
    return {
        "command": command,
        "command_text": command_text(command),
        "started_at": timestamp,
        "finished_at": timestamp,
        "duration_seconds": 0.0,
        "exit_code": None,
        "timed_out": False,
        "status": status,
        "reason": reason,
        "stdout_tail": "",
        "stderr_tail": "",
    }


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def source_snapshot(root: Path) -> dict[str, Any]:
    errors: list[str] = []
    spec_path = root / "docs" / "validation-refactor-spec.md"
    feature_directory = root / "features"
    spec_hash: str | None = None
    specification_id: str | None = None
    try:
        spec_bytes = spec_path.read_bytes()
        spec_hash = hashlib.sha256(spec_bytes).hexdigest()
        match = re.search(
            rb"^\*\*Specification:\*\*\s*(\S+)", spec_bytes, re.MULTILINE
        )
        specification_id = match.group(1).decode("utf-8") if match else None
        if specification_id is None:
            errors.append("specification ID not found")
    except (OSError, UnicodeDecodeError) as error:
        errors.append(f"cannot hash specification: {error}")

    feature_hashes: dict[str, str] = {}
    try:
        feature_paths = sorted(feature_directory.glob("*.feature"))
    except OSError as error:
        feature_paths = []
        errors.append(f"cannot list feature directory: {error}")
    if not feature_paths:
        errors.append("feature directory has no .feature files")
    for path in feature_paths:
        try:
            feature_hashes[path.relative_to(root).as_posix()] = sha256_file(path)
        except OSError as error:
            errors.append(f"cannot hash feature {path}: {error}")
    feature_index = "".join(
        f"{name}\0{digest}\n" for name, digest in feature_hashes.items()
    )
    return {
        "specification_id": specification_id,
        "spec_file": "docs/validation-refactor-spec.md",
        "spec_sha256": spec_hash,
        "feature_directory": "features",
        "feature_sha256": feature_hashes,
        "feature_index_sha256": hashlib.sha256(feature_index.encode()).hexdigest(),
        "errors": errors,
    }


def git_capture(root: Path, command: list[str]) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(
            command,
            cwd=root,
            text=True,
            encoding="utf-8",
            errors="replace",
            capture_output=True,
            timeout=10,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as error:
        return subprocess.CompletedProcess(command, 1, "", str(error))


def git_metadata(root: Path) -> dict[str, Any]:
    commit_result = git_capture(root, ["git", "rev-parse", "HEAD"])
    status_result = git_capture(
        root, ["git", "status", "--porcelain=v1", "--untracked-files=all"]
    )
    errors: list[str] = []
    commit = commit_result.stdout.strip() if commit_result.returncode == 0 else None
    if commit is None:
        errors.append(commit_result.stderr.strip() or "cannot determine commit")
    status_lines = status_result.stdout.splitlines()
    if status_result.returncode != 0:
        errors.append(status_result.stderr.strip() or "cannot determine worktree status")
    return {
        "commit": commit,
        "dirty": bool(status_lines) or status_result.returncode != 0,
        "status": status_lines,
        "errors": errors,
    }


def host_platform() -> tuple[str, str]:
    system = platform.system().lower()
    machine = platform.machine().lower()
    goos = {"darwin": "darwin", "macos": "darwin"}.get(system, system)
    goarch = {
        "x86_64": "amd64",
        "amd64": "amd64",
        "aarch64": "arm64",
        "arm64": "arm64",
    }.get(machine, machine)
    return goos, goarch


def collect_environment(
    go_binary: str, root: Path, *, timeout_seconds: float
) -> dict[str, Any]:
    command = [go_binary, "env", "GOVERSION", "GOOS", "GOARCH"]
    result = run_command(command, cwd=root, timeout_seconds=timeout_seconds)
    values = result["_stdout"].splitlines()
    while len(values) < 3:
        values.append("")
    go_version = values[0].strip() or None
    goos = values[1].strip() or None
    goarch = values[2].strip() or None
    host_goos, host_goarch = host_platform()
    result.pop("_stdout", None)
    result.pop("_stderr", None)
    result["go_version"] = go_version
    result["goos"] = goos
    result["goarch"] = goarch
    result["host_goos"] = host_goos
    result["host_goarch"] = host_goarch
    result["is_native"] = goos == host_goos and goarch == host_goarch
    result["go_version_matches"] = go_version == REQUIRED_GO_VERSION
    result["required_go_version"] = REQUIRED_GO_VERSION
    result["toolchain_ok"] = result["status"] == "passed" and result["go_version_matches"]
    return result


NUMBER_RE = re.compile(r"[-+]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][-+]?\d+)?")
BENCHMARK_RE = re.compile(r"^(Benchmark\S+)\s+(\d+)(?:\s+(.*))?$")


def parse_number(value: str) -> int | float | str:
    if not NUMBER_RE.fullmatch(value):
        return value
    try:
        number = float(value)
    except ValueError:
        return value
    return int(number) if number.is_integer() and "." not in value and "e" not in value.lower() else number


def benchmark_category(name: str) -> str:
    lowered = name.lower()
    if "construction" in lowered:
        return "construction"
    if "traversal" in lowered:
        return "error_traversal"
    if re.search(r"/(?:[^/]+/)?reporting(?:[-/]|$)", lowered):
        return "formatting"
    if re.search(r"/(?:[^/]+/)?invalid(?:[-/]|$)", lowered):
        return "failing_evaluation"
    if re.search(r"/(?:[^/]+/)?valid(?:[-/]|$)", lowered):
        return "valid_evaluation"
    return "other"


def input_item_count(value: int | float | str | None) -> int | float | None:
    if isinstance(value, int):
        return value
    if isinstance(value, float) and value.is_integer():
        return int(value)
    return value if isinstance(value, float) else None


def parse_benchmark_rows(
    output: str, *, compiler: str | None = None, platform_name: str | None = None
) -> list[dict[str, Any]]:
    """Parse standard ``go test -benchmem`` rows without assuming benchmark names."""

    rows: list[dict[str, Any]] = []
    for line in output.splitlines():
        match = BENCHMARK_RE.match(line.strip())
        if match is None:
            continue
        name, iterations_text, metrics_text = match.groups()
        tokens = (metrics_text or "").split()
        values: dict[str, int | float | str] = {}
        metrics: list[dict[str, Any]] = []
        index = 0
        while index + 1 < len(tokens):
            value, unit = tokens[index], tokens[index + 1]
            if NUMBER_RE.fullmatch(value):
                parsed = parse_number(value)
                metrics.append({"value": parsed, "unit": unit})
                if unit not in values:
                    values[unit] = parsed
                index += 2
            else:
                index += 1
        rows.append(
            {
                "name": name,
                "iterations": int(iterations_text),
                "category": benchmark_category(name),
                "compiler": compiler,
                "platform": platform_name,
                "input_items": input_item_count(values.get("input-items")),
                "metrics": metrics,
                "values": values,
                "raw": line.strip(),
            }
        )
    return rows


def benchmark_summary(rows: list[dict[str, Any]]) -> dict[str, Any]:
    counts = Counter(
        row.get("category") for row in rows if isinstance(row.get("category"), str)
    )
    missing_categories = [
        category
        for category in MANDATED_BENCHMARK_CATEGORIES
        if counts.get(category, 0) < 10
    ]
    invalid_rows: list[dict[str, Any]] = []
    for row in rows:
        values = row.get("values", {})
        missing = [
            metric for metric in BENCHMARK_METRICS if not isinstance(values.get(metric), (int, float))
        ]
        if row.get("compiler") is None:
            missing.append("compiler")
        if row.get("platform") is None:
            missing.append("platform")
        if row.get("input_items") is None:
            missing.append("input-items")
        if missing:
            invalid_rows.append({"name": row.get("name"), "missing": missing})
    return {
        "required_categories": list(MANDATED_BENCHMARK_CATEGORIES),
        "minimum_rows_per_category": 10,
        "counts": dict(sorted(counts.items())),
        "missing_categories": missing_categories,
        "invalid_rows": invalid_rows,
        "complete": not missing_categories and not invalid_rows,
    }


def validate_allocation_report(value: Any) -> list[str]:
    """Check the allocation checker contract, including every fixture phase."""

    if not isinstance(value, dict):
        return ["allocation report is not an object"]
    errors: list[str] = []
    if value.get("schema") != ALLOCATION_SCHEMA:
        errors.append(f"unexpected allocation schema: {value.get('schema')!r}")
    if value.get("status") != "passed":
        errors.append(f"allocation status is {value.get('status')!r}")
    failures = value.get("failures")
    if not isinstance(failures, list):
        errors.append("allocation failures are missing")
    elif failures:
        errors.append(f"allocation report contains {len(failures)} failure(s)")
    fixtures = value.get("fixtures")
    if not isinstance(fixtures, list) or not fixtures:
        errors.append("allocation fixtures are missing or empty")
        return errors
    for fixture in fixtures:
        if not isinstance(fixture, dict):
            errors.append("allocation fixture is not an object")
            continue
        name = fixture.get("name", "<unnamed>")
        phases = fixture.get("phases")
        if not isinstance(phases, dict):
            errors.append(f"{name}: phases are missing")
            continue
        for phase_name in ("first", "batch"):
            phase = phases.get(phase_name)
            if not isinstance(phase, dict):
                errors.append(f"{name}/{phase_name}: phase result is missing")
                continue
            if phase.get("status") != "passed":
                errors.append(f"{name}/{phase_name}: status is {phase.get('status')!r}")
            if not isinstance(phase.get("mallocs"), int):
                errors.append(f"{name}/{phase_name}: mallocs is not an integer")
            if not isinstance(phase.get("bytes"), int):
                errors.append(f"{name}/{phase_name}: bytes is not an integer")
    summary = value.get("summary")
    if not isinstance(summary, dict):
        errors.append("allocation summary is missing")
    else:
        expected = summary.get("expected_probes")
        observed = summary.get("observed_probes")
        if not isinstance(expected, int) or not isinstance(observed, int):
            errors.append("allocation probe counts are missing")
        elif expected != observed:
            errors.append(f"allocation probe count is {observed}, expected {expected}")
    return errors


def public_result(result: dict[str, Any]) -> dict[str, Any]:
    public = dict(result)
    public.pop("_stdout", None)
    public.pop("_stderr", None)
    return public


def write_report(path: Path, report: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def main(argv: Iterable[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    parser.add_argument("--output", type=Path, required=True, help="JSON report path")
    parser.add_argument("--go", default="go", help="Go executable (default: go)")
    parser.add_argument("--timeout", type=float, default=DEFAULT_TIMEOUT_SECONDS)
    parser.add_argument("--total-timeout", type=float, default=DEFAULT_TOTAL_TIMEOUT_SECONDS)
    parser.add_argument(
        "--quick",
        action="store_true",
        help="run only specification and correctness smoke checks; always incomplete",
    )
    args = parser.parse_args(list(argv) if argv is not None else None)
    if args.timeout <= 0 or args.total_timeout <= 0:
        parser.error("--timeout and --total-timeout must be positive")

    root = args.root.resolve()
    output_path = args.output.resolve()
    report: dict[str, Any] = {
        "schema": SCHEMA,
        "mode": "quick" if args.quick else "full",
        "started_at": now(),
        "finished_at": None,
        "status": "incomplete",
        "complete": False,
        "required_gates": [],
        "checks": {},
        "allocation": None,
        "allocation_validation": {"complete": False, "errors": ["not run"]},
        "benchmark_rows": [],
        "benchmark_summary": {
            "required_categories": list(MANDATED_BENCHMARK_CATEGORIES),
            "minimum_rows_per_category": 10,
            "counts": {},
            "missing_categories": list(MANDATED_BENCHMARK_CATEGORIES),
            "invalid_rows": [],
            "complete": False,
        },
        "environment": {},
        "repository": {},
        "source": {},
        "incomplete_reasons": [],
    }
    exit_code = 1
    deadline = time.monotonic() + args.total_timeout

    try:
        source_before = source_snapshot(root)
        repository_before = git_metadata(root)
        report["source"] = source_before
        report["repository"] = repository_before

        remaining = max(0.001, deadline - time.monotonic())
        report["environment"] = collect_environment(
            args.go, root, timeout_seconds=min(args.timeout, remaining)
        )

        with tempfile.TemporaryDirectory(prefix="validation-native-") as temporary:
            allocation_path = Path(temporary) / "allocation.json"

            def run_gate(gate_id: str, command: list[str]) -> dict[str, Any]:
                remaining_seconds = deadline - time.monotonic()
                if remaining_seconds <= 0:
                    result = not_run(command, "total timeout exhausted")
                else:
                    result = run_command(
                        command,
                        cwd=root,
                        timeout_seconds=min(args.timeout, remaining_seconds),
                    )
                report["checks"][gate_id] = public_result(result)
                return result

            commands: dict[str, list[str]] = {
                "spec": [sys.executable, str(root / "tools" / "check_spec.py")],
                "correctness": [args.go, "test", "./..."],
                "allocation": [
                    sys.executable,
                    str(root / "tools" / "check_allocation.py"),
                    "--json-output",
                    str(allocation_path),
                ],
                "alloc_tests": [args.go, "test", "-run", "^TestAlloc", "./..."],
                "benchmark": [
                    args.go,
                    "test",
                    "-run",
                    "^$",
                    "-bench",
                    ".",
                    "-benchmem",
                    "-count=10",
                    "./...",
                ],
                "staticcheck": ["make", "staticcheck"],
                "race": [args.go, "test", "-race", "./..."],
            }
            for gate_id, target in FUZZ_TARGETS.items():
                commands[gate_id] = [
                    args.go,
                    "test",
                    "-run",
                    "^$",
                    "-fuzz",
                    f"^{target}$",
                    "-fuzztime=60s",
                    "-parallel=4",
                    ".",
                ]

            for gate_id in ("spec", "correctness"):
                run_gate(gate_id, commands[gate_id])

            if args.quick:
                for gate_id in BASE_GATES[2:] + LINUX_AMD64_GATES:
                    report["checks"][gate_id] = not_run(
                        commands[gate_id], "quick mode omits native gate"
                    )
            else:
                for gate_id in ("allocation", "alloc_tests", "benchmark"):
                    result = run_gate(gate_id, commands[gate_id])
                    if gate_id == "allocation":
                        if allocation_path.is_file():
                            try:
                                report["allocation"] = json.loads(
                                    allocation_path.read_text(encoding="utf-8")
                                )
                            except (OSError, UnicodeDecodeError, json.JSONDecodeError) as error:
                                report["allocation"] = {
                                    "schema": ALLOCATION_SCHEMA,
                                    "status": "invalid",
                                    "error": str(error),
                                }
                        else:
                            report["allocation"] = {
                                "schema": ALLOCATION_SCHEMA,
                                "status": "not_available",
                                "error": "allocation checker did not write a report",
                            }
                        allocation_errors = validate_allocation_report(report["allocation"])
                        report["allocation_validation"] = {
                            "complete": not allocation_errors,
                            "errors": allocation_errors,
                        }
                    if gate_id == "benchmark":
                        environment = report["environment"]
                        report["benchmark_rows"] = parse_benchmark_rows(
                            result.get("_stdout", ""),
                            compiler=environment.get("go_version"),
                            platform_name=(
                                f"{environment.get('goos')}/{environment.get('goarch')}"
                                if environment.get("goos") and environment.get("goarch")
                                else None
                            ),
                        )
                        report["benchmark_summary"] = benchmark_summary(
                            report["benchmark_rows"]
                        )
                host_goos = report["environment"].get("goos")
                host_goarch = report["environment"].get("goarch")
                if (
                    report["environment"].get("is_native")
                    and host_goos == "linux"
                    and host_goarch == "amd64"
                ):
                    for gate_id in ("staticcheck", "race", *FUZZ_TARGETS):
                        run_gate(gate_id, commands[gate_id])
                else:
                    for gate_id in LINUX_AMD64_GATES:
                        report["checks"][gate_id] = not_run(
                            commands[gate_id],
                            "linux/amd64-only gate on another native platform",
                            status="not_applicable",
                        )

        report["source"]["end"] = source_snapshot(root)
        report["repository"]["end"] = git_metadata(root)
        report["repository"]["changed_during_run"] = (
            report["repository"].get("commit")
            != report["repository"]["end"].get("commit")
            or report["repository"].get("status")
            != report["repository"]["end"].get("status")
        )

        required_gates = list(BASE_GATES)
        if (
            report["environment"].get("is_native")
            and report["environment"].get("goos") == "linux"
            and report["environment"].get("goarch") == "amd64"
        ):
            required_gates.extend(LINUX_AMD64_GATES)
        report["required_gates"] = required_gates
        gate_statuses = [report["checks"].get(gate, {}).get("status") for gate in required_gates]
        failed = any(status in {"failed", "timed_out"} for status in gate_statuses)
        all_required_passed = all(status == "passed" for status in gate_statuses)
        if failed:
            report["status"] = "failed"
            report["incomplete_reasons"].append("one or more native gates failed")
        elif not all_required_passed:
            report["status"] = "incomplete"
            report["incomplete_reasons"].append("one or more required native gates did not run")
        if args.quick:
            report["incomplete_reasons"].append("quick mode is not release evidence")
        environment = report["environment"]
        if not environment.get("is_native"):
            report["incomplete_reasons"].append("Go target is not the host platform")
        if not environment.get("go_version_matches"):
            report["incomplete_reasons"].append(
                f"Go version is not {REQUIRED_GO_VERSION}"
            )
        if environment.get("status") != "passed":
            report["incomplete_reasons"].append("Go environment probe failed")
        if report["source"].get("errors") or report["source"].get("end", {}).get("errors"):
            report["incomplete_reasons"].append("source identity could not be established")
        if report["source"].get("spec_sha256") != report["source"].get("end", {}).get(
            "spec_sha256"
        ) or report["source"].get("feature_sha256") != report["source"].get("end", {}).get(
            "feature_sha256"
        ):
            report["incomplete_reasons"].append("specification or feature files changed")
        if report["repository"].get("end", {}).get("dirty"):
            report["incomplete_reasons"].append("worktree is dirty")
        if report["repository"].get("changed_during_run"):
            report["incomplete_reasons"].append("repository changed during the run")
        if not args.quick:
            allocation_report = report.get("allocation")
            allocation_status = (
                allocation_report.get("status")
                if isinstance(allocation_report, dict)
                else None
            )
            if allocation_status != "passed" or not report["allocation_validation"].get(
                "complete"
            ):
                report["incomplete_reasons"].append(
                    "allocation checker report or fixture phases are missing or did not pass"
                )
                if allocation_status in {"failed", "invalid"} or not report[
                    "allocation_validation"
                ].get("complete"):
                    report["status"] = "failed"
            if not report["benchmark_summary"].get("complete"):
                report["incomplete_reasons"].append(
                    "benchmark rows do not cover all required categories and metrics"
                )

        report["incomplete_reasons"] = list(dict.fromkeys(report["incomplete_reasons"]))
        report["complete"] = (
            report["status"] != "failed"
            and all_required_passed
            and not report["incomplete_reasons"]
        )
        if report["complete"]:
            report["status"] = "passed"
            exit_code = 0
        elif report["status"] != "failed":
            report["status"] = "incomplete"
            exit_code = 1
    except Exception as error:  # Keep the report contract on unexpected failures.
        report["status"] = "failed"
        report["complete"] = False
        report["incomplete_reasons"].append(f"runner error: {error}")
        exit_code = 1
    finally:
        report["finished_at"] = now()
        try:
            write_report(output_path, report)
        except OSError as error:
            print(f"cannot write native gate report {output_path}: {error}", file=sys.stderr)
            exit_code = 1

    print(
        f"native gates: status={report['status']} complete={report['complete']} "
        f"report={output_path}",
        file=sys.stderr,
    )
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
