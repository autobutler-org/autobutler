#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
from pathlib import Path


WRK_SCENARIOS = [
    "albums_list.txt",
    "cirrus_list.txt",
    "photos_list.txt",
    "photos_metadata.txt",
    "thumbnails.txt",
]


def fmt(value: str | None, fallback: str = "n/a") -> str:
    return value if value else fallback


def parse_wrk_file(path: Path) -> dict[str, str]:
    stats: dict[str, str] = {}
    for raw_line in path.read_text().splitlines():
        line = raw_line.strip()
        if line.startswith("Requests/sec:"):
            stats["requests_per_sec"] = line.split()[-1]
        elif line.startswith("Transfer/sec:"):
            stats["transfer_per_sec"] = line.split()[-1]
        elif match := re.match(r"^(50|75|90|99)%\s+(\S+)$", line):
            stats[f"p{match.group(1)}"] = match.group(2)
        elif match := re.match(r"^(\d+) requests in ([\d.]+)s,.*$", line):
            stats["requests_total"] = match.group(1)
            stats["duration"] = f"{match.group(2)}s"
        elif line.startswith("Non-2xx or 3xx responses:"):
            stats["non2xx"] = line.split()[-1]
        elif line.startswith("Socket errors:"):
            errors = []
            for kind, value in re.findall(r"(connect|read|write|timeout)\s+(\d+)", line):
                if value != "0":
                    errors.append(f"{kind} {value}")
            if errors:
                stats["socket_errors"] = ", ".join(errors)
    return stats


def parse_benchmark_line(line: str) -> dict[str, str] | None:
    parts = line.split()
    if len(parts) < 4 or not parts[0].startswith("Benchmark"):
        return None

    result: dict[str, str] = {
        "name": parts[0].removeprefix("Benchmark"),
        "iterations": parts[1],
        "time_per_op": " ".join(parts[2:4]),
        "b_per_op": "n/a",
        "allocs_per_op": "n/a",
    }

    index = 4
    while index + 1 < len(parts):
        value, unit = parts[index], parts[index + 1]
        if unit == "B/op":
            result["b_per_op"] = f"{value} {unit}"
        elif unit == "allocs/op":
            result["allocs_per_op"] = f"{value} {unit}"
        else:
            break
        index += 2

    return result


def parse_benchmarks(path: Path) -> list[dict[str, str]]:
    benchmarks: list[dict[str, str]] = []
    for raw_line in path.read_text().splitlines():
        benchmark = parse_benchmark_line(raw_line.strip())
        if benchmark is not None:
            benchmarks.append(benchmark)
    return benchmarks


def render_wrk_section(results_dir: Path) -> list[str]:
    lines: list[str] = []
    section_name = results_dir.name.replace("-", " ").title()
    if not section_name.lower().endswith("test"):
        section_name = f"{section_name} Test"
    lines.append(f"### {section_name}")
    lines.append("")

    available_files = [results_dir / filename for filename in WRK_SCENARIOS if (results_dir / filename).exists()]
    if not available_files:
        lines.append("_No wrk outputs were found._")
        lines.append("")
        return lines

    lines.append("| Scenario | Requests | Req/s | P50 | P90 | P99 | Transfer/sec | Notes |")
    lines.append("| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |")

    for file_path in available_files:
        stats = parse_wrk_file(file_path)
        notes = []
        if non2xx := stats.get("non2xx"):
            notes.append(f"non-2xx {non2xx}")
        if socket_errors := stats.get("socket_errors"):
            notes.append(socket_errors)

        lines.append(
            "| "
            + file_path.stem.replace("_", " ")
            + " | "
            + f"{fmt(stats.get('requests_total'))} / {fmt(stats.get('duration'))} | "
            + f"{fmt(stats.get('requests_per_sec'))} | "
            + f"{fmt(stats.get('p50'))} | "
            + f"{fmt(stats.get('p90'))} | "
            + f"{fmt(stats.get('p99'))} | "
            + f"{fmt(stats.get('transfer_per_sec'))} | "
            + (", ".join(notes) if notes else "ok")
            + " |"
        )

    upload_status = results_dir / "upload_status_codes.txt"
    if upload_status.exists():
        codes = [line.strip() for line in upload_status.read_text().splitlines() if line.strip()]
        successful = sum(1 for code in codes if code == "200")
        failures = len(codes) - successful
        lines.append("")
        lines.append("#### Upload Stress")
        lines.append("")
        lines.append("| Total uploads | Successful | Failed |")
        lines.append("| ---: | ---: | ---: |")
        lines.append(f"| {len(codes)} | {successful} | {failures} |")

    lines.append("")
    return lines


def render_benchmark_section(bench_results: Path) -> list[str]:
    lines: list[str] = []
    lines.append("### Benchmark Suite")
    lines.append("")

    if not bench_results.exists():
        lines.append("_No benchmark output was found._")
        lines.append("")
        return lines

    benchmarks = parse_benchmarks(bench_results)
    if not benchmarks:
        lines.append("_No benchmark lines were found in the benchmark output._")
        lines.append("")
        return lines

    lines.append(f"Benchmarks run: {len(benchmarks)}")
    lines.append("")
    lines.append("| Benchmark | Iterations | Time/op | B/op | Allocs/op |")
    lines.append("| --- | ---: | ---: | ---: | ---: |")
    for benchmark in benchmarks:
        lines.append(
            f"| {benchmark['name']} | {benchmark['iterations']} | {benchmark['time_per_op']} | "
            f"{benchmark['b_per_op']} | {benchmark['allocs_per_op']} |"
        )

    lines.append("")
    return lines


def main() -> None:
    parser = argparse.ArgumentParser(description="Render a Markdown performance summary.")
    parser.add_argument(
        "--wrk-dir",
        action="append",
        default=[],
        dest="wrk_dirs",
        help="Directory containing wrk result text files.",
    )
    parser.add_argument(
        "--bench-results",
        type=Path,
        help="Path to the benchmark output file.",
    )
    args = parser.parse_args()

    print("## Performance Dashboard")
    print("")
    print("A quick summary of the latest load, stress, and benchmark runs.")
    print("")

    for wrk_dir in args.wrk_dirs:
        directory = Path(wrk_dir)
        if directory.exists():
            print("\n".join(render_wrk_section(directory)), end="")

    if args.bench_results is not None:
        print("\n".join(render_benchmark_section(args.bench_results)), end="")


if __name__ == "__main__":
    main()