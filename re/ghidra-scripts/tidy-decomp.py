"""Shrinks Ghidra C output from libnickel by folding Qt's inlined QString/QArrayData refcount release blocks.

Usage: python3 -I tidy-decomp.py <in.c> > <out.c>
Folded blocks become one line "  ~ref(<var>);" so the control flow that matters stays readable.
"""

import re
import sys

RELEASE_START = re.compile(r"^(\s*)if \(\*\(int \*\)(\w+(?:\[\d+\])?) == 0\) \{$")


def fold_refcount_blocks(lines):
    output = []
    index = 0
    while index < len(lines):
        match = RELEASE_START.match(lines[index])
        end = _release_block_end(lines, index) if match else None
        if end is None:
            output.append(lines[index])
            index += 1
            continue
        output.append(f"{match.group(1)}~ref({match.group(2)});")
        index = end + 1
    return output


def _release_block_end(lines, start):
    """Returns the index of the closing brace of an if/else-if refcount block, or None if it is something else."""
    window = lines[start : start + 16]
    if not any("QArrayData::deallocate" in line for line in window[:4]):
        return None
    if not any("hasExclusiveAccess" in line for line in window):
        return None
    indent = len(lines[start]) - len(lines[start].lstrip())
    for offset in range(5, len(window)):
        line = window[offset]
        if line.strip() == "}" and len(line) - len(line.lstrip()) == indent:
            return start + offset
    return None


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: tidy-decomp.py <in.c>")
    with open(sys.argv[1], encoding="utf-8") as source:
        lines = source.read().split("\n")
    sys.stdout.write("\n".join(fold_refcount_blocks(lines)))


if __name__ == "__main__":
    main()
