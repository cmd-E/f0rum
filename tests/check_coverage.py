"""Fail CI when measured statement coverage falls below the checked-in floor."""
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1]
minimum = json.loads((root / ".github/coverage-baseline.json").read_text())["minimumStatementPercent"]
lines = (root / ".artifacts/coverage.out").read_text().splitlines()
if not lines or not lines[0].startswith("mode: "):
    raise SystemExit("Missing or invalid Go coverage report")
# -coverpkg may repeat a block across test packages. A block is covered if any
# package exercised it; count each source location once.
blocks = {}
for line in lines[1:]:
    location, statements, hits = line.rsplit(" ", 2)
    count, covered = int(statements), int(hits) > 0
    if location in blocks:
        old_count, old_covered = blocks[location]
        if old_count != count:
            raise SystemExit("Inconsistent coverage report")
        covered |= old_covered
    blocks[location] = count, covered
total = sum(count for count, _ in blocks.values())
covered = sum(count for count, hit in blocks.values() if hit)
if total == 0:
    raise SystemExit("No statements in coverage report")
percent = covered * 100 / total
print(f"Statement coverage: {covered}/{total} ({percent:.2f}%); required: {minimum}%")
if percent < minimum:
    raise SystemExit("Coverage regression")
