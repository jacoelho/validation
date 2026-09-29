"""Contract tests for native acceptance evidence and matrix aggregation."""

from __future__ import annotations

import copy
import unittest

from tools import record_acceptance as acceptance


def native_record(goos: str = "darwin", goarch: str = "arm64") -> dict:
    source = {
        "specification_id": "VRS-test",
        "spec_sha256": "spec-digest",
        "feature_sha256": {"features/example.feature": "feature-digest"},
    }
    repository = {"commit": "a" * 40, "dirty": False}
    environment = {
        "go_version": "go1.27.0",
        "goos": goos,
        "goarch": goarch,
        "is_native": True,
    }
    gates = set(acceptance.BASE_NATIVE_GATES)
    if goos == "linux" and goarch == "amd64":
        gates.update(acceptance.LINUX_AMD64_NATIVE_GATES)
    gate_checks = {
        name: {"status": "passed", "exit_code": 0, "timed_out": False}
        for name in gates
    }
    benchmark_rows = [
        {
            "category": category,
            "compiler": environment["go_version"],
            "platform": f"{goos}/{goarch}",
            "input_items": 8,
            "values": {"ns/op": 1, "B/op": 0, "allocs/op": 0, "input-items": 8},
        }
        for category in acceptance.MANDATED_BENCHMARK_CATEGORIES
        for _ in range(10)
    ]
    native_gates = {
        "schema": acceptance.NATIVE_GATES_SCHEMA,
        "mode": "full",
        "status": "passed",
        "complete": True,
        "source": copy.deepcopy(source),
        "repository": copy.deepcopy(repository),
        "environment": copy.deepcopy(environment),
        "checks": gate_checks,
        "allocation": {
            "schema": "validation.allocation-evidence.v1",
            "status": "passed",
            "metadata": {
                "compilers": [environment["go_version"]],
                "platforms": [f"{goos}/{goarch}"],
            },
            "fixtures": [
                {
                    "name": "scalar",
                    "phases": {
                        phase: {"status": "passed", "mallocs": 0, "bytes": 0}
                        for phase in ("first", "batch")
                    },
                }
            ],
        },
        "benchmark_summary": {"complete": True},
        "benchmark_rows": benchmark_rows,
    }
    return {
        "schema": acceptance.SCHEMA,
        "source": source,
        "repository": repository,
        "environment": environment,
        "checks": {"local_complete": True, "native_gates_complete": True},
        "native_gates": native_gates,
    }


def is_valid(report: dict) -> tuple[bool, list[str]]:
    return acceptance.native_gates_valid(
        report["native_gates"],
        source=report["source"],
        repository=report["repository"],
        environment=report["environment"],
    )


class NativeEvidenceTests(unittest.TestCase):
    def test_local_release_needs_cases_and_native_gates(self) -> None:
        base = {
            "checks": {
                "tests_complete": True,
                "native_gates_complete": True,
                "worktree_clean": True,
                "source_stable": True,
                "metadata_errors": [],
            },
            "bindings": {
                "manifest_schema_valid": True,
                "unbound": 0,
                "invalid_cases": 0,
            },
        }
        self.assertEqual(acceptance.local_complete(base), (True, []))
        for section, key, value in (
            ("checks", "tests_complete", False),
            ("checks", "native_gates_complete", False),
            ("checks", "worktree_clean", False),
            ("checks", "source_stable", False),
            ("bindings", "unbound", 1),
            ("bindings", "invalid_cases", 1),
            ("bindings", "manifest_schema_valid", False),
        ):
            with self.subTest(missing=key):
                report = copy.deepcopy(base)
                report[section][key] = value
                complete, reasons = acceptance.local_complete(report)
                self.assertFalse(complete)
                self.assertTrue(reasons)

    def test_complete_native_record_is_valid(self) -> None:
        self.assertEqual(is_valid(native_record()), (True, []))
        self.assertEqual(is_valid(native_record("linux", "amd64")), (True, []))

    def test_every_missing_gate_blocks_native_evidence(self) -> None:
        for name in (*acceptance.BASE_NATIVE_GATES, *acceptance.LINUX_AMD64_NATIVE_GATES):
            with self.subTest(gate=name):
                report = native_record("linux", "amd64")
                del report["native_gates"]["checks"][name]
                valid, errors = is_valid(report)
                self.assertFalse(valid)
                self.assertIn(f"native gate {name} did not pass", errors)

    def test_missing_allocation_phase_and_benchmark_rows_block(self) -> None:
        report = native_record()
        del report["native_gates"]["allocation"]["fixtures"][0]["phases"]["first"]
        report["native_gates"]["benchmark_rows"].pop()
        valid, errors = is_valid(report)
        self.assertFalse(valid)
        self.assertTrue(any("lacks passing phases" in error for error in errors), errors)
        self.assertTrue(any("fewer than ten" in error for error in errors), errors)

    def test_stale_identity_and_cross_compilation_block(self) -> None:
        report = native_record()
        report["native_gates"]["source"]["feature_sha256"] = {"other": "hash"}
        report["native_gates"]["repository"]["commit"] = "b" * 40
        report["native_gates"]["environment"]["is_native"] = False
        valid, errors = is_valid(report)
        self.assertFalse(valid)
        self.assertTrue(any("feature identity differs" in error for error in errors), errors)
        self.assertTrue(any("implementation identity differs" in error for error in errors), errors)
        self.assertTrue(any("not native" in error for error in errors), errors)

    def test_matrix_requires_every_native_platform(self) -> None:
        current = native_record()
        linux = native_record("linux", "amd64")
        required = [
            {"go_version": "go1.27.0", "goos": "darwin", "goarch": "arm64"},
            {"go_version": "go1.27.0", "goos": "linux", "goarch": "amd64"},
            {"go_version": "go1.27.0", "goos": "linux", "goarch": "arm64"},
        ]
        gate = acceptance.build_matrix_gate(current, [linux], required)
        self.assertFalse(gate["complete"])
        self.assertEqual(gate["missing"], [required[2]])


if __name__ == "__main__":
    unittest.main()
