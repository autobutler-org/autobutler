#! /usr/bin/env bash

# This script finds lines marked with "// coverage: ignore" and marks them as covered in the coverage file

set -euo pipefail

COVERAGE_FILE="${1:-coverage.out}"

if [[ ! -f "${COVERAGE_FILE}" ]]; then
    echo "Error: Coverage file '${COVERAGE_FILE}' not found"
    exit 1
fi

# Create a new file for the ignored coverage data
IGNORED_FILE="${COVERAGE_FILE}.ignored"
cp "${COVERAGE_FILE}" "${IGNORED_FILE}"

# Find all lines with "// coverage: ignore" comment
# Format: ./filepath:linenum:code content
grep -rn '// coverage: ignore' . --include="*.go" 2>/dev/null | while IFS=: read -r filepath linenum rest; do
    # Remove leading ./ if present
    filepath="${filepath#./}"

    # Convert to module path format used in coverage.out
    modpath="autobutler/$filepath"

    # Use awk to find and modify coverage lines that include this line number
    # Coverage format: filepath:startline.startcol,endline.endcol numstmt count
    awk -v file="$modpath" -v target_line="$linenum" '
    {
        # Check if this is a coverage line for our file
        if ($0 ~ "^" file ":") {
            # Parse the line: filepath:start.col,end.col stmts count
            split($1, parts, ":")
            split(parts[2], range, ",")
            split(range[1], start, ".")
            split(range[2], end, ".")

            start_line = start[1]
            end_line = end[1]

            # If target line is in this range and count is 0, change to 1
            if (start_line <= target_line && target_line <= end_line && $NF == 0) {
                $NF = 1
            }
        }
        print
    }
    ' "$IGNORED_FILE" > "${IGNORED_FILE}.new" && mv "${IGNORED_FILE}.new" "$IGNORED_FILE"
done
