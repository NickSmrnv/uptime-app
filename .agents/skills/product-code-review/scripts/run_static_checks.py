#!/usr/bin/env python3
"""Run diff-aware static checks for uptime-app backend and frontend changes."""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[4]


def run(label: str, command: list[str], cwd: Path) -> bool:
    print(f"\n[{label}] {' '.join(command)}")
    result = subprocess.run(command, cwd=cwd, text=True)
    print(f"[{label}] {'PASS' if result.returncode == 0 else 'FAIL'}")
    return result.returncode == 0


def changed_paths(args: argparse.Namespace) -> list[str]:
    if args.working_tree or args.staged:
        command = ["git", "diff", "--name-only"]
        if args.staged:
            command.append("--cached")
        if args.working_tree:
            command.append("HEAD")
    else:
        base = args.base or "HEAD^"
        command = ["git", "diff", "--name-only", f"{base}...HEAD"]
    result = subprocess.run(command, cwd=ROOT, text=True, capture_output=True, check=True)
    paths = [line for line in result.stdout.splitlines() if line]
    if args.working_tree:
        untracked = subprocess.run(
            ["git", "ls-files", "--others", "--exclude-standard"],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=True,
        )
        paths.extend(line for line in untracked.stdout.splitlines() if line)
    return sorted(set(paths))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", help="Git ref to compare with the reviewed commit")
    parser.add_argument("--staged", action="store_true", help="Review staged changes")
    parser.add_argument("--working-tree", action="store_true", help="Review unstaged working-tree changes")
    args = parser.parse_args()

    if args.staged and args.working_tree:
        parser.error("choose only one of --staged and --working-tree")

    paths = changed_paths(args)
    backend = any(path == "backend" or path.startswith("backend/") for path in paths)
    frontend = any(path == "frontend" or path.startswith("frontend/") for path in paths)
    print(f"Review root: {ROOT}")
    print(f"Changed paths: {len(paths)} (backend={backend}, frontend={frontend})")
    for path in paths:
        print(f"  {path}")

    passed = True
    if backend:
        backend_dir = ROOT / "backend"
        passed &= run("backend:gofmt", ["sh", "-c", "test -z \"$(gofmt -l .)\""], backend_dir)
        passed &= run("backend:test", ["go", "test", "./..."], backend_dir)
        passed &= run("backend:vet", ["go", "vet", "./..."], backend_dir)
    if frontend:
        frontend_dir = ROOT / "frontend"
        passed &= run("frontend:lint", ["npm", "run", "lint"], frontend_dir)
        passed &= run("frontend:test", ["npm", "test", "--", "--run"], frontend_dir)
        tsconfig = frontend_dir / "tsconfig.json"
        if tsconfig.exists():
            passed &= run("frontend:typecheck", ["npx", "tsc", "--noEmit"], frontend_dir)
    if not backend and not frontend:
        print("No backend/frontend paths changed; no application checks were selected.")
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())
