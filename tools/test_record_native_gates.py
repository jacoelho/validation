#!/usr/bin/env python3
"""Focused tests for the native-gate evidence runner."""

from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


TOOLS = Path(__file__).resolve().parent
ROOT = TOOLS.parent
sys.path.insert(0, str(TOOLS))

import record_native_gates as gates  # noqa: E402


class NativeGateRunnerTests(unittest.TestCase):
    def test_benchmark_parser_keeps_rows_and_units(self) -> None:
        output = """
        goos: darwin
        BenchmarkValidation/valid-8 123 456.7 ns/op 8 B/op 1 allocs/op 4 input-items
        BenchmarkValidation/invalid-8 10 1.2 us/op 0 B/op 0 allocs/op 4 input-items
        """

        rows = gates.parse_benchmark_rows(
            output, compiler="go1.27.0", platform_name="darwin/arm64"
        )

        self.assertEqual([row["name"] for row in rows], [
            "BenchmarkValidation/valid-8",
            "BenchmarkValidation/invalid-8",
        ])
        self.assertEqual(rows[0]["iterations"], 123)
        self.assertEqual(rows[0]["values"]["ns/op"], 456.7)
        self.assertEqual(rows[0]["values"]["B/op"], 8)
        self.assertEqual(rows[0]["category"], "valid_evaluation")
        self.assertEqual(rows[0]["compiler"], "go1.27.0")
        self.assertEqual(rows[0]["platform"], "darwin/arm64")
        self.assertEqual(rows[0]["input_items"], 4)
        self.assertEqual(rows[1]["values"]["us/op"], 1.2)
        self.assertEqual(rows[1]["category"], "failing_evaluation")

    def test_timeout_is_explicitly_not_a_pass(self) -> None:
        result = gates.run_command(
            [sys.executable, "-c", "import time; time.sleep(1)"],
            cwd=ROOT,
            timeout_seconds=0.02,
        )

        self.assertEqual(result["status"], "timed_out")
        self.assertTrue(result["timed_out"])
        self.assertIsNone(result["exit_code"])

    def test_allocation_phase_validation_rejects_missing_phase(self) -> None:
        report = {
            "schema": gates.ALLOCATION_SCHEMA,
            "status": "passed",
            "fixtures": [
                {
                    "name": "scalar",
                    "phases": {
                        "first": {"status": "passed", "mallocs": 0, "bytes": 0}
                    },
                }
            ],
        }

        errors = gates.validate_allocation_report(report)

        self.assertTrue(any("scalar/batch" in error for error in errors))

    def test_benchmark_summary_requires_ten_rows_per_category(self) -> None:
        rows = []
        for category in gates.MANDATED_BENCHMARK_CATEGORIES:
            for ordinal in range(10):
                rows.append(
                    {
                        "name": f"Benchmark/{category}/{ordinal}",
                        "category": category,
                        "compiler": "go1.27.0",
                        "platform": "darwin/arm64",
                        "input_items": 4,
                        "values": {
                            "ns/op": 1,
                            "B/op": 0,
                            "allocs/op": 0,
                            "input-items": 4,
                        },
                    }
                )

        summary = gates.benchmark_summary(rows[:-1])
        self.assertFalse(summary["complete"])
        self.assertIn(gates.MANDATED_BENCHMARK_CATEGORIES[-1], summary["missing_categories"])
        self.assertTrue(gates.benchmark_summary(rows)["complete"])

    def test_quick_report_contains_every_gate_and_stays_incomplete(self) -> None:
        expected_gates = set(gates.BASE_GATES) | set(gates.LINUX_AMD64_GATES)
        with tempfile.TemporaryDirectory(prefix="native-gate-test-") as temporary:
            output = Path(temporary) / "report.json"
            completed = subprocess.run(
                [
                    sys.executable,
                    str(TOOLS / "record_native_gates.py"),
                    "--root",
                    str(ROOT),
                    "--output",
                    str(output),
                    "--quick",
                    "--timeout",
                    "60",
                    "--total-timeout",
                    "120",
                ],
                cwd=ROOT,
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertEqual(completed.returncode, 1, completed.stderr)
            report = json.loads(output.read_text(encoding="utf-8"))

        self.assertEqual(report["schema"], gates.SCHEMA)
        self.assertFalse(report["complete"])
        self.assertEqual(set(report["checks"]), expected_gates)
        self.assertEqual(report["checks"]["spec"]["status"], "passed")
        self.assertEqual(report["checks"]["correctness"]["status"], "passed")
        for gate_id in expected_gates - {"spec", "correctness"}:
            self.assertEqual(report["checks"][gate_id]["status"], "not_run")
        self.assertIn("quick mode is not release evidence", report["incomplete_reasons"])


if __name__ == "__main__":
    unittest.main()
