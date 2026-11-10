"""Reproduce free LogFlow checks and publishable, synthetic-only measurements.

Run from any directory: python tools/verify.py --date YYYY-MM-DD
Results intentionally exclude usernames, hostnames, tokens and launcher URLs.
"""
import argparse
import hashlib
import json
import os
import platform
import re
import statistics
import subprocess
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--date", required=True, help="Report date in your local YYYY-MM-DD")
    args = parser.parse_args()
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", args.date):
        parser.error("date must be YYYY-MM-DD")
    output = ROOT / "docs/results"
    output.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, LOGFLOW_NO_BROWSER="1", GOMAXPROCS="4")
    checks = []
    source_files = [ROOT / "go.mod"]
    for directory in ("cmd", "internal", "tools", ".github"):
        source_files.extend(p for p in (ROOT / directory).rglob("*")
                            if p.is_file() and p.suffix in (".go", ".html", ".css", ".py", ".yml", ".yaml"))
    manifest = {p.relative_to(ROOT).as_posix(): hashlib.sha256(p.read_bytes().replace(b"\r\n", b"\n")).hexdigest()
                for p in sorted(source_files)}
    fingerprint = hashlib.sha256(json.dumps(manifest, sort_keys=True).encode()).hexdigest()
    # Normalize line endings so Git's Windows checkout conversion does not affect identity.


    def run(name, command, custom_env=None):
        print(f"Running {name}...", flush=True)
        result = subprocess.run(command, cwd=ROOT, env=custom_env or env,
                                capture_output=True, text=True, encoding="utf-8",
                                errors="replace", timeout=600, check=False)
        checks.append({"name": name, "command": " ".join(str(x) for x in command),
                       "passed": result.returncode == 0})
        if result.returncode:
            raise RuntimeError(f"{name} failed (exit {result.returncode}); inspect locally before publishing")
        return result.stdout

    with tempfile.TemporaryDirectory(prefix="logflow-verification-") as temporary:
        scratch = Path(temporary)
        env["LOGFLOW_DATA_DIR"] = str(scratch / "app-parent")
        coverage = scratch / "coverage.out"
        tests = run("Race tests and coverage", ["go", "test", "-race", "-count=1", "-json",
                                                f"-coverprofile={coverage}", "./..."])
        events = [json.loads(line) for line in tests.splitlines() if line.startswith("{")]
        passed = [{"package": e["Package"], "test": e["Test"]} for e in events
                  if e.get("Action") == "pass" and "Test" in e]
        skipped = [{"package": e["Package"], "test": e["Test"]} for e in events
                   if e.get("Action") == "skip" and "Test" in e]
        cover = run("Coverage summary", ["go", "tool", "cover", f"-func={coverage}"])
        coverage_pct = float(re.search(r"total:.*?([\d.]+)%", cover).group(1))
        run("Go vet", ["go", "vet", "./..."])
        binary = scratch / ("logflow.exe" if os.name == "nt" else "logflow")
        run("Native build", ["go", "build", "-trimpath", "-o", str(binary), "./cmd/logflow"])
        run("Packaged workflow smoke", [os.sys.executable, ".github/scripts/smoke.py", str(binary)])
        replay = json.loads(run("Synthetic replay", [str(binary), "-replay"]))
        run("Parser fuzz (15 seconds)", ["go", "test", "./internal/parser", "-run", "^$",
                                         "-fuzz", "^FuzzParseLine$", "-fuzztime", "15s", "-parallel", "2"])
        run("Analysis fuzz (15 seconds)", ["go", "test", "./internal/analysis", "-run", "^$",
                                           "-fuzz", "^FuzzProcessReportInvariants$", "-fuzztime", "15s", "-parallel", "2"])
        benchmark = run("In-memory benchmarks", ["go", "test", "./internal/analysis", "-run", "^$",
                                                 "-bench", "^BenchmarkProcess$", "-benchmem", "-benchtime=3x", "-count=3"])
        linux_env = dict(env, GOOS="linux", GOARCH="amd64", CGO_ENABLED="0")
        run("Linux amd64 cross-build (not executed)", ["go", "build", "-trimpath", "-o",
                                                       str(scratch / "logflow-linux"), "./cmd/logflow"], linux_env)
        go_version = subprocess.check_output(["go", "version"], cwd=ROOT, text=True).strip()
        base = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()

    samples = {}
    for line in benchmark.splitlines():
        if not line.startswith("BenchmarkProcess/"):
            continue
        fields = line.split()
        values = {fields[i + 1]: float(fields[i]) for i in range(2, len(fields) - 1, 2)}
        samples.setdefault(fields[0], []).append(values)
    if len(samples) != 4 or any(len(v) != 3 for v in samples.values()):
        raise RuntimeError("Unexpected benchmark sample count")
    summary = [{"case": case, "samples": len(rows),
                "median_records_per_second": round(statistics.median(r["records/s"] for r in rows)),
                "min_records_per_second": round(min(r["records/s"] for r in rows)),
                "max_records_per_second": round(max(r["records/s"] for r in rows)),
                "median_allocated_bytes_per_operation": round(statistics.median(r["B/op"] for r in rows)),
                "median_allocations_per_operation": round(statistics.median(r["allocs/op"] for r in rows))}
               for case, rows in samples.items()]
    # Replace ephemeral absolute paths and local interpreter identity in recorded commands.
    for check in checks:
        check["command"] = check["command"].replace(str(scratch), "<temporary>").replace(os.sys.executable, "python")
    result = {"date": args.date, "source_base_commit": base,
              "source_code_sha256_lf_normalized": fingerprint, "source_manifest": manifest,
              "source_note": "Measured working tree including this report's accompanying code changes; not the base commit alone.",
              "environment": {"go": go_version, "os": platform.system(), "os_release": platform.release(),
                              "architecture": platform.machine(), "logical_cpus": os.cpu_count(),
                              "GOMAXPROCS": 4, "cpu": platform.processor()},
              "checks": checks, "coverage_percent": coverage_pct,
              "top_level_tests_passed": sum("/" not in p["test"] for p in passed),
              "subtests_passed": sum("/" in p["test"] for p in passed),
              "passed_tests": passed, "skipped_tests": skipped,
              "replay": replay, "benchmarks": summary,
              "limitations": ["Synthetic ordered JSONL only; no field detection accuracy measured.",
                              "Benchmarks exclude fixture generation, disk I/O, UI, JSON export and Jira.",
                              "B/op is total allocated bytes per complete analysis, NOT peak RSS or retained memory.",
                              "Three repeated samples, each with three analyses; not a capacity/SLA claim.",
                              "Visual/accessibility and live Jira checks remain manual and unverified."]}
    (output / "results.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    (output / "benchmarks.txt").write_text(benchmark, encoding="utf-8")
    rows = "\n".join(f'| {r["case"].removeprefix("BenchmarkProcess/").rsplit("-", 1)[0]} | {r["median_records_per_second"]:,} | {r["min_records_per_second"]:,}–{r["max_records_per_second"]:,} | {r["median_allocated_bytes_per_operation"]:,} |' for r in summary)
    report = f'''# Verification results — {args.date}

All {len(checks)} recorded automated checks passed. {result["top_level_tests_passed"]} top-level tests and {result["subtests_passed"]} subtests passed; {len(skipped)} tests skipped.
Statement coverage: **{coverage_pct}%**, measured with race detection. Coverage is not a correctness or attack-detection score.

Environment: {go_version}; {platform.system()} {platform.release()}, {platform.machine()};
{os.cpu_count()} logical CPUs available, Go constrained to GOMAXPROCS=4. CPU: {platform.processor()}.

Reproduce: `python tools/verify.py --date YYYY-MM-DD`. Requires Go, a race-compatible
C compiler and Python 3.10+. All application data is temporary and browser opening
is disabled. Jira tests use local TLS mocks; no credentials, paid APIs or cloud resources.

## In-memory processing benchmarks

| Input / sources / workers | Median records/s | Observed range | Allocated bytes/analysis |
| --- | ---: | ---: | ---: |
{rows}

Each row summarizes three repeated samples, three complete analyses per sample.
Fixture generation is excluded. Times include parsing, ordered reduction, hashing,
evidence and sensitivity; exclude disk I/O, serialization, UI and Jira.
**Allocated bytes are cumulative Go allocations, not peak process memory.**
All records are matching failures spaced 1 ms apart, round-robin over the stated
number of IPs. Defaults apply except worker count. Each analysis must finish
complete with the exact record count and one alert per source. Benchmarks use
normal builds, not race-instrumented builds. Do not extrapolate to production workloads.

## Detection behavior

Four fixed synthetic replays agree with expectations: concentrated failures **1**
alert; slow attempts **0**; distributed sources **0**; benign repeated failures **1**.
This demonstrates two blind spots and a benign alert, **not 100% detection accuracy**.
The benign label is supplied by the fixture, not inferred from log contents.

## Functional evidence and limits

See [the functionality matrix](../FUNCTIONALITY.md), [every passed test](results.json),
[raw benchmark samples](benchmarks.txt), and [the manual checklist](../../TESTING.md).
Manual observations are not claimed. Linux was cross-compiled, not executed locally;
GitHub's separate OS jobs provide additional evidence only after they actually pass.

Source fingerprint (LF-normalized code/test/tool/workflow files): `{fingerprint}`.
The per-file SHA-256 manifest is recorded in results.json; verify it against the
checkout before attributing these measurements to another revision.

Source baseline: `{base}` plus the code changes accompanying this report.
The baseline hash alone does not identify the measured working tree. Results are
machine-specific observations; reruns need not have identical timings.
'''
    (output / "README.md").write_text(report, encoding="utf-8")
    print(f"PASS: wrote docs/results (coverage {coverage_pct}%, {len(passed)} passed test cases)")


if __name__ == "__main__":
    main()
