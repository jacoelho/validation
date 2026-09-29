#!/usr/bin/env python3
"""Run mapped acceptance tests and write current-revision evidence.

The binding checker answers whether a feature case has a route to a Go test.
This tool answers the next question: what happened when those routes were run?
It deliberately keeps the two jobs separate.  An evidence report is complete
only when every case is bound and passed, the repository and specification
inputs are stable and clean, and all required native matrix records are
available for the same revision.

The command writes a report even when the gate is incomplete and exits with
status 1 in that case.  This makes missing evidence visible without allowing a
partial local run to look like a release certificate.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
import re
import shlex
import subprocess
import sys
import time
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable


SCHEMA = "validation.acceptance-evidence.v1"
MANIFEST_SCHEMA = "validation.acceptance-bindings.v1"
NATIVE_GATES_SCHEMA = "validation.native-gates.v1"
DEFAULT_TIMEOUT_SECONDS = 600.0
DEFAULT_TOTAL_TIMEOUT_SECONDS = 1800.0
DEFAULT_REQUIRED_MATRIX = (
    {"go_version": "go1.27.0", "goos": "linux", "goarch": "amd64"},
    {"go_version": "go1.27.0", "goos": "linux", "goarch": "arm64"},
    {"go_version": "go1.27.0", "goos": "darwin", "goarch": "arm64"},
)
BASE_NATIVE_GATES = ("spec", "correctness", "allocation", "alloc_tests", "benchmark")
LINUX_AMD64_NATIVE_GATES = (
    "staticcheck", "race", "fuzz_paths", "fuzz_numeric", "fuzz_slices", "fuzz_maps"
)
MANDATED_BENCHMARK_CATEGORIES = (
    "construction", "valid_evaluation", "failing_evaluation", "error_traversal", "formatting"
)
SPEC_ID_RE = re.compile(r"^\*\*Specification:\*\*\s*(\S+)", re.MULTILINE)
GO_VERSION_RE = re.compile(r"\bgo\d+(?:\.\d+)+(?:[-+][^\s]+)?\b")


def now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def run_capture(command: list[str], *, cwd: Path) -> subprocess.CompletedProcess[str]:
    """Run a metadata command without raising; its failure is reportable."""

    return subprocess.run(
        command,
        cwd=cwd,
        text=True,
        capture_output=True,
        check=False,
    )


def git_metadata(root: Path) -> dict[str, Any]:
    commit_result = run_capture(["git", "rev-parse", "HEAD"], cwd=root)
    status_result = run_capture(
        ["git", "status", "--porcelain=v1", "--untracked-files=all"], cwd=root
    )
    commit = commit_result.stdout.strip() if commit_result.returncode == 0 else None
    status = status_result.stdout.splitlines() if status_result.returncode == 0 else []
    errors: list[str] = []
    if commit is None:
        detail = commit_result.stderr.strip() or f"exit {commit_result.returncode}"
        errors.append(f"cannot determine git commit: {detail}")
    if status_result.returncode != 0:
        detail = status_result.stderr.strip() or f"exit {status_result.returncode}"
        errors.append(f"cannot determine git status: {detail}")
    return {
        "commit": commit,
        "dirty": bool(status) or status_result.returncode != 0,
        "status": status,
        "errors": errors,
    }


def source_snapshot(root: Path, manifest: dict[str, Any]) -> dict[str, Any]:
    """Hash the exact spec and feature files named by the binding manifest."""

    errors: list[str] = []
    spec_name = manifest.get("spec_file")
    spec_path: Path | None = None
    if isinstance(spec_name, str) and spec_name:
        spec_path = root / spec_name
    else:
        errors.append("manifest has no non-empty spec_file")

    feature_name = manifest.get("feature_directory")
    feature_directory: Path | None = None
    if isinstance(feature_name, str) and feature_name:
        feature_directory = root / feature_name
    else:
        errors.append("manifest has no non-empty feature_directory")

    spec_hash: str | None = None
    specification_id: str | None = None
    if spec_path is not None:
        try:
            spec_bytes = spec_path.read_bytes()
            spec_hash = sha256_bytes(spec_bytes)
            match = SPEC_ID_RE.search(spec_bytes.decode("utf-8"))
            specification_id = match.group(1) if match else None
            if specification_id is None:
                errors.append(f"specification ID not found in {spec_name}")
        except (OSError, UnicodeDecodeError) as exc:
            errors.append(f"cannot hash spec {spec_name}: {exc}")

    feature_hashes: dict[str, str] = {}
    if feature_directory is not None:
        try:
            feature_paths = sorted(feature_directory.glob("*.feature"))
        except OSError as exc:
            feature_paths = []
            errors.append(f"cannot list feature directory {feature_name}: {exc}")
        if not feature_paths:
            errors.append(f"feature directory has no .feature files: {feature_name}")
        for path in feature_paths:
            try:
                relative = path.relative_to(root).as_posix()
                feature_hashes[relative] = sha256_file(path)
            except OSError as exc:
                errors.append(f"cannot hash feature {path}: {exc}")

    feature_index = "".join(f"{name}\0{digest}\n" for name, digest in feature_hashes.items())
    return {
        "specification_id": specification_id,
        "spec_file": spec_name,
        "spec_sha256": spec_hash,
        "feature_directory": feature_name,
        "feature_sha256": feature_hashes,
        "feature_index_sha256": sha256_bytes(feature_index.encode("utf-8")),
        "errors": errors,
    }


def load_manifest(path: Path) -> tuple[dict[str, Any], list[dict[str, Any]], list[str]]:
    errors: list[str] = []
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        return {}, [], [f"cannot read binding manifest {path}: {exc}"]
    if not isinstance(value, dict):
        return {}, [], ["binding manifest top-level value is not an object"]
    entries = value.get("cases")
    if not isinstance(entries, list):
        errors.append("binding manifest cases is not a list")
        entries = []
    cases = [entry for entry in entries if isinstance(entry, dict)]
    if len(cases) != len(entries):
        errors.append("binding manifest contains a non-object case")
    if value.get("schema") != MANIFEST_SCHEMA:
        errors.append(f"unsupported binding manifest schema: {value.get('schema')!r}")
    return value, cases, errors


def normalize_goos(value: str) -> str:
    return {"darwin": "darwin", "macos": "darwin", "linux": "linux"}.get(
        value.lower(), value.lower()
    )


def normalize_goarch(value: str) -> str:
    value = value.lower()
    return {"x86_64": "amd64", "x64": "amd64", "aarch64": "arm64"}.get(value, value)


def native_host() -> dict[str, str]:
    system = platform.system().lower()
    machine = platform.machine().lower()
    return {"goos": normalize_goos(system), "goarch": normalize_goarch(machine)}


def go_metadata(go_binary: str, root: Path) -> tuple[dict[str, Any], list[str]]:
    errors: list[str] = []
    env_result = run_capture(
        [go_binary, "env", "GOVERSION", "GOOS", "GOARCH", "GOFLAGS", "GOTOOLCHAIN", "CGO_ENABLED"],
        cwd=root,
    )
    values = env_result.stdout.splitlines()
    while len(values) < 7:
        values.append("")
    if env_result.returncode != 0:
        detail = env_result.stderr.strip() or f"exit {env_result.returncode}"
        errors.append(f"cannot query Go environment: {detail}")
    go_version = values[0].strip() or None
    if go_version is None:
        version_result = run_capture([go_binary, "version"], cwd=root)
        match = GO_VERSION_RE.search(version_result.stdout)
        go_version = match.group(0) if match else None
        if go_version is None:
            errors.append("cannot determine exact Go version")
    target = {
        "go_version": go_version,
        "goos": normalize_goos(values[1].strip()) if values[1].strip() else None,
        "goarch": normalize_goarch(values[2].strip()) if values[2].strip() else None,
        "goflags": values[3],
        "gotoolchain": values[4],
        "cgo_enabled": values[5],
        "version_output": None,
    }
    version_result = run_capture([go_binary, "version"], cwd=root)
    target["version_output"] = version_result.stdout.strip() or version_result.stderr.strip()
    if version_result.returncode != 0:
        errors.append(f"go version failed: {target['version_output'] or version_result.returncode}")
    host = native_host()
    target["host_goos"] = host["goos"]
    target["host_goarch"] = host["goarch"]
    target["is_native"] = (
        target["goos"] == host["goos"] and target["goarch"] == host["goarch"]
    )
    return target, errors


def read_command(command: str, go_binary: str) -> tuple[list[str] | None, str | None]:
    try:
        tokens = shlex.split(command)
    except ValueError as exc:
        return None, f"invalid command quoting: {exc}"
    if len(tokens) < 2 or Path(tokens[0]).name != "go" or tokens[1] != "test":
        return None, "bound command must be a direct 'go test' command"
    if any(token in {"&&", "||", ";", "|"} for token in tokens):
        return None, "bound command must not contain shell operators"
    tokens[0] = go_binary
    if "-json" not in tokens[2:]:
        tokens.insert(2, "-json")
    return tokens, None


def terminal_test_status(events: list[dict[str, Any]]) -> str:
    terminal = [
        event.get("action", event.get("Action"))
        for event in events
        if event.get("action", event.get("Action")) in {"pass", "fail", "skip"}
    ]
    if not terminal:
        return "missing"
    if "fail" in terminal:
        return "failed"
    if "skip" in terminal:
        return "skipped"
    return "passed"


def run_test_command(
    command: list[str],
    targets: list[str],
    *,
    root: Path,
    timeout_seconds: float,
    dry_run: bool,
) -> dict[str, Any]:
    started = now()
    monotonic_start = time.monotonic()
    if dry_run:
        return {
            "command": command,
            "started_at": started,
            "finished_at": started,
            "duration_seconds": 0.0,
            "exit_code": None,
            "timed_out": False,
            "dry_run": True,
            "status": "not_run",
            "targets": {
                target: {"status": "not_run", "events": []} for target in targets
            },
            "packages": {},
            "diagnostics": [],
        }

    try:
        completed = subprocess.run(
            command,
            cwd=root,
            text=True,
            capture_output=True,
            timeout=timeout_seconds,
            check=False,
        )
        timed_out = False
        exit_code: int | None = completed.returncode
        stdout = completed.stdout
        stderr = completed.stderr
    except subprocess.TimeoutExpired as exc:
        timed_out = True
        exit_code = None
        stdout = (exc.stdout or "") if isinstance(exc.stdout, str) else ""
        stderr = (exc.stderr or "") if isinstance(exc.stderr, str) else ""
    finished = now()
    duration = round(time.monotonic() - monotonic_start, 6)
    events_by_test: dict[str, list[dict[str, Any]]] = {}
    package_status: dict[str, str] = {}
    diagnostics: list[str] = []
    malformed_json_lines = 0
    for line in stdout.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            malformed_json_lines += 1
            continue
        if not isinstance(event, dict):
            malformed_json_lines += 1
            continue
        action = event.get("Action")
        package = event.get("Package")
        if isinstance(package, str) and action in {"pass", "fail", "skip"} and not event.get("Test"):
            package_status[package] = action
        test_name = event.get("Test")
        if isinstance(test_name, str):
            compact = {
                key: event[key]
                for key in ("Time", "Action", "Package", "Test", "Elapsed", "Output")
                if key in event
            }
            events_by_test.setdefault(test_name, []).append(compact)
    if stderr.strip():
        diagnostics.append(stderr[-65536:])
    if malformed_json_lines:
        diagnostics.append(f"ignored {malformed_json_lines} non-JSON stdout line(s)")
    if timed_out:
        diagnostics.append(f"go test exceeded timeout of {timeout_seconds:g}s")

    target_results: dict[str, dict[str, Any]] = {}
    for target in targets:
        matched: list[dict[str, Any]] = []
        for name, events in events_by_test.items():
            if name == target or name.startswith(target + "/"):
                matched.extend(events)
        # Keep the full event sequence for exact auditability, while reducing
        # the public status to the terminal outcome for this mapped test.
        status = terminal_test_status(matched)
        if timed_out or (exit_code is not None and exit_code != 0 and status == "passed"):
            status = "failed"
        target_results[target] = {"status": status, "events": matched}

    statuses = [result["status"] for result in target_results.values()]
    if timed_out or exit_code not in (0, None):
        run_status = "failed"
    elif any(status == "failed" for status in statuses):
        run_status = "failed"
    elif any(status == "missing" for status in statuses):
        run_status = "missing"
    elif any(status in {"skipped", "not_run"} for status in statuses):
        run_status = "skipped"
    else:
        run_status = "passed"
    return {
        "command": command,
        "started_at": started,
        "finished_at": finished,
        "duration_seconds": duration,
        "exit_code": exit_code,
        "timed_out": timed_out,
        "dry_run": False,
        "status": run_status,
        "targets": target_results,
        "packages": package_status,
        "diagnostics": diagnostics,
    }


def parse_required_matrix(values: list[str]) -> tuple[list[dict[str, str]], list[str]]:
    if not values:
        return [dict(item) for item in DEFAULT_REQUIRED_MATRIX], []
    required: list[dict[str, str]] = []
    errors: list[str] = []
    for value in values:
        match = re.fullmatch(r"([^:@/]+)[@:](?:(\w+)/(\w+))", value)
        if match is None:
            errors.append(
                f"invalid --required-native {value!r}; use GO_VERSION@GOOS/GOARCH, e.g. go1.27.0@linux/amd64"
            )
            continue
        go_version, goos, goarch = match.groups()
        required.append(
            {"go_version": go_version, "goos": normalize_goos(goos), "goarch": normalize_goarch(goarch)}
        )
    return required, errors


def matrix_key(environment: dict[str, Any]) -> tuple[str | None, str | None, str | None]:
    return (
        environment.get("go_version"),
        environment.get("goos"),
        environment.get("goarch"),
    )


def load_matrix_records(paths: Iterable[Path]) -> tuple[list[dict[str, Any]], list[str]]:
    records: list[dict[str, Any]] = []
    errors: list[str] = []
    for path in paths:
        try:
            value = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
            errors.append(f"cannot read matrix evidence {path}: {exc}")
            continue
        if not isinstance(value, dict) or value.get("schema") != SCHEMA:
            errors.append(f"matrix evidence {path} has unsupported schema")
            continue
        records.append(value)
    return records, errors


def load_native_gates(path: Path | None) -> tuple[dict[str, Any] | None, list[str]]:
    if path is None:
        return None, ["native gate record was not supplied"]
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as exc:
        return None, [f"cannot read native gate record {path}: {exc}"]
    if not isinstance(value, dict):
        return None, [f"native gate record {path} is not an object"]
    return value, []


def native_gates_valid(
    record: dict[str, Any] | None,
    *,
    source: dict[str, Any],
    repository: dict[str, Any],
    environment: dict[str, Any],
) -> tuple[bool, list[str]]:
    if record is None:
        return False, ["native gate record is missing"]
    errors: list[str] = []
    if record.get("schema") != NATIVE_GATES_SCHEMA:
        errors.append("native gate record schema differs")
    if record.get("mode") != "full" or record.get("status") != "passed" or record.get("complete") is not True:
        errors.append("native gate record is incomplete")
    gate_source = record.get("source", {})
    if not isinstance(gate_source, dict) or any(
        gate_source.get(key) != source.get(key)
        for key in ("specification_id", "spec_sha256", "feature_sha256")
    ):
        errors.append("native gate record specification or feature identity differs")
    gate_repository = record.get("repository", {})
    if not isinstance(gate_repository, dict) or gate_repository.get("commit") != repository.get("commit") or gate_repository.get("dirty") or gate_repository.get("changed_during_run"):
        errors.append("native gate record implementation identity differs or is dirty")
    gate_environment = record.get("environment", {})
    if not isinstance(gate_environment, dict) or any(
        gate_environment.get(key) != environment.get(key)
        for key in ("go_version", "goos", "goarch")
    ) or gate_environment.get("is_native") is not True:
        errors.append("native gate record toolchain/platform differs or is not native")
    required = set(BASE_NATIVE_GATES)
    if environment.get("goos") == "linux" and environment.get("goarch") == "amd64":
        required.update(LINUX_AMD64_NATIVE_GATES)
    checks = record.get("checks", {})
    if not isinstance(checks, dict):
        checks = {}
    for gate in sorted(required):
        result = checks.get(gate)
        if not isinstance(result, dict) or result.get("status") != "passed" or result.get("exit_code") != 0 or result.get("timed_out"):
            errors.append(f"native gate {gate} did not pass")
    allocation = record.get("allocation", {})
    if not isinstance(allocation, dict) or allocation.get("schema") != "validation.allocation-evidence.v1" or allocation.get("status") != "passed":
        errors.append("native allocation evidence did not pass")
    else:
        metadata = allocation.get("metadata", {})
        if not isinstance(metadata, dict) or metadata.get("compilers") != [environment.get("go_version")] or metadata.get("platforms") != [f"{environment.get('goos')}/{environment.get('goarch')}"]:
            errors.append("native allocation evidence toolchain/platform differs")
        fixtures = allocation.get("fixtures", [])
        if not isinstance(fixtures, list) or not fixtures:
            errors.append("native allocation fixture evidence is missing")
        else:
            for fixture in fixtures:
                if not isinstance(fixture, dict):
                    errors.append("native allocation fixture is malformed")
                    break
                phases = fixture.get("phases", {})
                if not isinstance(phases, dict) or any(
                    not isinstance(phases.get(phase), dict) or phases[phase].get("status") != "passed"
                    for phase in ("first", "batch")
                ):
                    errors.append(f"native allocation fixture {fixture.get('name')} lacks passing phases")
                    break
    benchmark_summary = record.get("benchmark_summary", {})
    if not isinstance(benchmark_summary, dict) or benchmark_summary.get("complete") is not True:
        errors.append("native benchmark categories or measurements are incomplete")
    benchmark_rows = record.get("benchmark_rows", [])
    if not isinstance(benchmark_rows, list) or not benchmark_rows:
        errors.append("native benchmark rows are missing")
    else:
        categories: Counter[str] = Counter()
        for row in benchmark_rows:
            values = row.get("values") if isinstance(row, dict) else None
            if not isinstance(row, dict) or not isinstance(values, dict) or any(
                metric not in values for metric in ("ns/op", "B/op", "allocs/op", "input-items")
            ) or row.get("compiler") != environment.get("go_version") or row.get("platform") != f"{environment.get('goos')}/{environment.get('goarch')}" or row.get("input_items") is None:
                errors.append("native benchmark row lacks required metrics")
                break
            category = row.get("category")
            if isinstance(category, str):
                categories[category] += 1
        for category in MANDATED_BENCHMARK_CATEGORIES:
            if categories[category] < 10:
                errors.append(f"native benchmark category {category} has fewer than ten rows")
    return not errors, errors


def local_complete(report: dict[str, Any]) -> tuple[bool, list[str]]:
    reasons: list[str] = []
    checks = report["checks"]
    bindings = report["bindings"]
    if bindings.get("manifest_schema_valid") is not True:
        reasons.append("binding manifest is invalid")
    if bindings.get("unbound", 0):
        reasons.append(f"{bindings['unbound']} acceptance case(s) are unbound")
    if bindings.get("invalid_cases", 0):
        reasons.append(f"{bindings['invalid_cases']} acceptance case(s) are malformed")
    if not checks.get("tests_complete"):
        reasons.append("one or more mapped tests are missing, skipped, failed or not run")
    if not checks.get("native_gates_complete"):
        reasons.append("native correctness, allocation, benchmark, race, fuzz or source gates are incomplete")
    if not checks.get("worktree_clean"):
        reasons.append("worktree is dirty or git status is unavailable")
    if not checks.get("source_stable"):
        reasons.append("specification or feature files changed during the run")
    if checks.get("metadata_errors"):
        reasons.append("toolchain or repository metadata is incomplete")
    return not reasons, reasons


def mapped_cases_valid(record: dict[str, Any], expected_keys: set[str]) -> bool:
    bindings = record.get("bindings")
    checks = record.get("checks")
    repository = record.get("repository")
    cases = record.get("cases")
    runs = record.get("runs")
    if not all(isinstance(value, dict) for value in (bindings, checks, repository)) or not isinstance(cases, list) or not isinstance(runs, list):
        return False
    if (
        not isinstance(repository.get("commit"), str)
        or repository.get("commit_after") != repository.get("commit")
        or repository.get("dirty") is not False
        or repository.get("dirty_before") is not False
        or repository.get("dirty_after") is not False
        or bindings.get("manifest_schema_valid") is not True
        or bindings.get("bound") != len(expected_keys)
        or bindings.get("unbound") != 0
        or bindings.get("invalid_cases") != 0
        or checks.get("tests_complete") is not True
        or checks.get("worktree_clean") is not True
        or checks.get("source_stable") is not True
        or checks.get("metadata_errors") != []
    ):
        return False
    keys: set[str] = set()
    for case in cases:
        if not isinstance(case, dict) or not isinstance(case.get("key"), str):
            return False
        key = case["key"]
        if key in keys or case.get("binding_status") != "bound" or case.get("result") != "passed":
            return False
        keys.add(key)
    if keys != expected_keys or not runs:
        return False
    for run in runs:
        if not isinstance(run, dict) or run.get("status") != "passed" or run.get("timed_out") is True:
            return False
        targets = run.get("targets")
        if not isinstance(targets, dict) or not targets or any(
            not isinstance(target, dict) or target.get("status") != "passed"
            for target in targets.values()
        ):
            return False
    for case in cases:
        index = case.get("run_index")
        test = case.get("test")
        if type(index) is not int or index < 0 or index >= len(runs) or not isinstance(test, str) or test not in runs[index]["targets"]:
            return False
    return True


def build_matrix_gate(
    current: dict[str, Any],
    previous: list[dict[str, Any]],
    required: list[dict[str, str]],
    expected_case_keys: set[str],
    *,
    include_current: bool = True,
) -> dict[str, Any]:
    records = [*previous, *([current] if include_current else [])]
    current_spec = current.get("source", {})
    current_commit = current.get("repository", {}).get("commit")
    observed: list[dict[str, Any]] = []
    usable: dict[tuple[str | None, str | None, str | None], dict[str, Any]] = {}
    rejected: list[dict[str, Any]] = []
    for record in records:
        environment = record.get("environment", {})
        gate_valid, _ = native_gates_valid(
            record.get("native_gates"),
            source=record.get("source", {}),
            repository=record.get("repository", {}),
            environment=environment,
        )
        identity = {
            "go_version": environment.get("go_version"),
            "goos": environment.get("goos"),
            "goarch": environment.get("goarch"),
            "native": bool(environment.get("is_native")),
            "commit": record.get("repository", {}).get("commit"),
            "source_match": record.get("source", {}).get("spec_sha256") == current_spec.get("spec_sha256")
            and record.get("source", {}).get("feature_sha256") == current_spec.get("feature_sha256"),
            "local_complete": bool(record.get("checks", {}).get("local_complete")),
            "mapped_cases_complete": mapped_cases_valid(record, expected_case_keys),
            "native_gates_complete": bool(record.get("checks", {}).get("native_gates_complete")) and gate_valid,
        }
        observed.append(identity)
        key = matrix_key(environment)
        if not identity["native"]:
            rejected.append({**identity, "reason": "record is not native"})
        elif identity["commit"] != current_commit:
            rejected.append({**identity, "reason": "implementation commit differs"})
        elif not identity["source_match"]:
            rejected.append({**identity, "reason": "specification or feature hash differs"})
        elif not identity["mapped_cases_complete"]:
            rejected.append({**identity, "reason": "mapped case evidence is incomplete"})
        elif not identity["local_complete"] or not identity["native_gates_complete"]:
            rejected.append({**identity, "reason": "local evidence is incomplete"})
        else:
            usable[key] = record

    missing = [item for item in required if (item["go_version"], item["goos"], item["goarch"]) not in usable]
    return {
        "required": required,
        "observed": observed,
        "missing": missing,
        "rejected": rejected,
        "complete": not missing,
    }


def case_key(case: dict[str, Any]) -> str:
    row = case.get("row_ordinal")
    return f"{case.get('feature')}:{case.get('scenario_id')}:{row if row is not None else '-'}"


def write_json(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--manifest", type=Path, default=None)
    parser.add_argument("--output", type=Path, required=True, help="JSON report path")
    parser.add_argument("--go", default="go", help="Go executable (default: go)")
    parser.add_argument("--timeout", type=float, default=DEFAULT_TIMEOUT_SECONDS)
    parser.add_argument("--total-timeout", type=float, default=DEFAULT_TOTAL_TIMEOUT_SECONDS)
    parser.add_argument("--matrix-record", type=Path, action="append", default=[])
    parser.add_argument(
        "--native-gates",
        type=Path,
        help="native correctness, allocation, benchmark, race, fuzz and source evidence",
    )
    parser.add_argument(
        "--required-native",
        action="append",
        default=[],
        metavar="GO_VERSION@GOOS/GOARCH",
        help="required native pair; repeat to override the default Go 1.27 matrix",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="record planned commands without executing tests; never produces a complete gate",
    )
    parser.add_argument(
        "--local-only",
        action="store_true",
        help="exit on local evidence while retaining the full matrix status in the report",
    )
    parser.add_argument(
        "--aggregate-only",
        action="store_true",
        help="verify supplied native records without rerunning mapped tests on this host",
    )
    args = parser.parse_args()
    if args.timeout <= 0 or args.total_timeout <= 0:
        parser.error("--timeout and --total-timeout must be positive")
    if args.aggregate_only and args.local_only:
        parser.error("--aggregate-only and --local-only cannot be combined")
    if args.aggregate_only and args.dry_run:
        parser.error("--aggregate-only and --dry-run cannot be combined")
    root = args.root.resolve()
    manifest_path = (args.manifest or root / "acceptance" / "bindings.json").resolve()
    output_path = args.output.resolve()
    started_at = now()

    manifest, cases, manifest_errors = load_manifest(manifest_path)
    binding_check = run_capture(
        [
            sys.executable,
            str(root / "tools" / "check_acceptance.py"),
            "--root",
            str(root),
            "--manifest",
            str(manifest_path),
        ],
        cwd=root,
    )
    source_before = source_snapshot(root, manifest)
    git_before = git_metadata(root)
    environment, environment_errors = go_metadata(args.go, root)
    required_matrix, matrix_arg_errors = parse_required_matrix(args.required_native)
    previous_records, previous_errors = load_matrix_records(
        path.resolve() for path in args.matrix_record if path.resolve() != output_path
    )
    native_gate_record, native_gate_load_errors = load_native_gates(args.native_gates)
    native_gate_complete, native_gate_errors = native_gates_valid(
        native_gate_record,
        source=source_before,
        repository=git_before,
        environment=environment,
    )

    valid_case_count = 0
    invalid_case_count = 0
    bound_case_count = 0
    unbound_case_count = 0
    runs: list[dict[str, Any]] = []
    run_by_command: dict[tuple[str, ...], int] = {}
    case_results: list[dict[str, Any]] = []
    for case in cases:
        status = case.get("status")
        if status not in {"bound", "unbound"}:
            invalid_case_count += 1
            case_results.append({"key": case_key(case), "status": "invalid"})
            continue
        valid_case_count += 1
        base = {
            "key": case_key(case),
            "feature": case.get("feature"),
            "scenario_id": case.get("scenario_id"),
            "row_ordinal": case.get("row_ordinal"),
            "binding_status": status,
        }
        if status == "unbound":
            unbound_case_count += 1
            case_results.append({**base, "result": "unbound", "reason": case.get("reason")})
            continue
        bound_case_count += 1
        test_name = case.get("test")
        command_text = case.get("command")
        if not isinstance(test_name, str) or not test_name or not isinstance(command_text, str) or not command_text:
            invalid_case_count += 1
            case_results.append({**base, "result": "missing", "test": test_name, "reason": "bound case lacks test or command"})
            continue
        command, command_error = read_command(command_text, args.go)
        if command is None:
            invalid_case_count += 1
            case_results.append({**base, "test": test_name, "result": "missing", "reason": command_error})
            continue
        command_key = tuple(command)
        if command_key not in run_by_command:
            run_by_command[command_key] = len(runs)
            runs.append({"command": command, "targets": [], "case_keys": []})
        run_index = run_by_command[command_key]
        runs[run_index]["targets"].append(test_name)
        runs[run_index]["case_keys"].append(case_key(case))
        case_results.append({**base, "test": test_name, "run_index": run_index})

    deadline = time.monotonic() + args.total_timeout
    for run in runs:
        # Repeated outline rows can map to the same test.  One test event is
        # enough for each row, but only retain one target in the subprocess
        # result to keep the evidence compact and unambiguous.
        run["targets"] = list(dict.fromkeys(run["targets"]))
        remaining = deadline - time.monotonic()
        if remaining <= 0:
            run.update(
                {
                    "started_at": now(),
                    "finished_at": now(),
                    "duration_seconds": 0.0,
                    "exit_code": None,
                    "timed_out": True,
                    "dry_run": False,
                    "status": "not_run",
                    "targets": {name: {"status": "not_run", "events": []} for name in run["targets"]},
                    "packages": {},
                    "diagnostics": ["total acceptance runtime limit exceeded"],
                }
            )
            continue
        result = run_test_command(
            run["command"],
            run["targets"],
            root=root,
            timeout_seconds=min(args.timeout, remaining),
            dry_run=args.dry_run or args.aggregate_only,
        )
        run.clear()
        run.update(result)

    tests_complete = True
    for result in case_results:
        if result.get("binding_status") != "bound":
            continue
        if "run_index" not in result:
            result["result"] = result.get("result", "missing")
            tests_complete = False
            continue
        run = runs[result["run_index"]]
        target = run["targets"].get(result["test"], {"status": "missing"})
        result["result"] = target.get("status", "missing")
        if result["result"] != "passed" or run.get("status") != "passed":
            tests_complete = False

    source_after = source_snapshot(root, manifest)
    git_after = git_metadata(root)
    source_stable = (
        source_before.get("spec_sha256") == source_after.get("spec_sha256")
        and source_before.get("feature_sha256") == source_after.get("feature_sha256")
        and not source_before.get("errors")
        and not source_after.get("errors")
    )
    worktree_clean = (
        not git_before.get("dirty")
        and not git_after.get("dirty")
        and git_before.get("commit") is not None
        and git_before.get("commit") == git_after.get("commit")
    )
    metadata_errors = [
        *manifest_errors,
        *source_before.get("errors", []),
        *source_after.get("errors", []),
        *git_before.get("errors", []),
        *git_after.get("errors", []),
        *environment_errors,
        *matrix_arg_errors,
        *previous_errors,
        *([] if args.aggregate_only else native_gate_load_errors),
    ]
    if unbound_case_count == 0 and binding_check.returncode != 0:
        metadata_errors.append(
            "binding manifest checker rejected complete case routing: "
            + (binding_check.stderr.strip() or binding_check.stdout.strip())
        )
    report: dict[str, Any] = {
        "schema": SCHEMA,
        "generated_at": started_at,
        "finished_at": now(),
        "source": source_before,
        "repository": {
            "commit": git_before.get("commit"),
            "commit_after": git_after.get("commit"),
            "dirty": bool(git_before.get("dirty") or git_after.get("dirty")),
            "dirty_before": bool(git_before.get("dirty")),
            "dirty_after": bool(git_after.get("dirty")),
            "status_before": git_before.get("status", []),
            "status_after": git_after.get("status", []),
        },
        "environment": environment,
        "build_flags": {
            "GOFLAGS": environment.get("goflags", ""),
            "GOTOOLCHAIN": environment.get("gotoolchain", ""),
            "CGO_ENABLED": environment.get("cgo_enabled", ""),
            "go_test_commands": [run["command"] for run in runs],
        },
        "bindings": {
            "manifest": str(manifest_path.relative_to(root)) if manifest_path.is_relative_to(root) else str(manifest_path),
            "schema": manifest.get("schema"),
            "execution_status": manifest.get("execution_status"),
            "declared_acceptance_status": manifest.get("acceptance_status"),
            "total": len(cases),
            "valid": valid_case_count,
            "invalid_cases": invalid_case_count,
            "bound": bound_case_count,
            "unbound": unbound_case_count,
            "manifest_schema_valid": (
                manifest.get("schema") == MANIFEST_SCHEMA
                and not manifest_errors
                and (unbound_case_count != 0 or binding_check.returncode == 0)
            ),
            "structure_check_exit_code": binding_check.returncode,
        },
        "runs": runs,
        "cases": case_results,
        "checks": {
            "tests_complete": tests_complete and not args.dry_run,
            "native_gates_complete": native_gate_complete,
            "native_gate_errors": native_gate_errors,
            "worktree_clean": worktree_clean,
            "source_stable": source_stable,
            "metadata_errors": metadata_errors,
            "local_complete": False,
        },
        "matrix": {},
        "matrix_records": [str(path.resolve()) for path in args.matrix_record],
        "native_gates": native_gate_record,
        "limits": {
            "dry_run": args.dry_run,
            "aggregate_only": args.aggregate_only,
            "timeout_seconds": args.timeout,
            "total_timeout_seconds": args.total_timeout,
            "native_matrix_is_required": True,
            "cross_compilation_counts_as_native": False,
        },
    }
    complete_local, local_reasons = local_complete(report)
    report["checks"]["local_complete"] = complete_local
    report["checks"]["local_reasons"] = local_reasons
    report["matrix"] = build_matrix_gate(
        report, previous_records, required_matrix, {case_key(case) for case in cases},
        include_current=not args.aggregate_only
    )
    if args.aggregate_only:
        release_reasons = []
        if bound_case_count == 0 or unbound_case_count or invalid_case_count or binding_check.returncode:
            release_reasons.append("acceptance bindings are incomplete or invalid")
        if not worktree_clean:
            release_reasons.append("worktree is dirty or git status is unavailable")
        if not source_stable:
            release_reasons.append("specification or feature files changed during aggregation")
        if metadata_errors:
            release_reasons.append("toolchain or repository metadata is incomplete")
    else:
        release_reasons = list(local_reasons)
    if not report["matrix"]["complete"]:
        missing = ", ".join(
            f"{item['go_version']} {item['goos']}/{item['goarch']}" for item in report["matrix"]["missing"]
        )
        release_reasons.append(f"native matrix is incomplete; missing {missing or 'required records'}")
    report["release_acceptance"] = {
        "status": "accepted" if not release_reasons else "incomplete",
        "complete": not release_reasons,
        "reasons": release_reasons,
        "claim": (
            "All mapped cases and required native matrix records passed for this revision."
            if not release_reasons
            else "No release acceptance claim; required evidence is incomplete."
        ),
    }
    report["finished_at"] = now()
    write_json(output_path, report)
    status = report["release_acceptance"]["status"]
    print(f"acceptance evidence: {status}; wrote {output_path}")
    if args.local_only:
        for reason in local_reasons:
            print(f"  {reason}", file=sys.stderr)
        return 0 if complete_local else 1
    if release_reasons:
        for reason in release_reasons:
            print(f"  {reason}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
