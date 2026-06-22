#!/usr/bin/env python3
import json
import os
import subprocess
import sys
from pathlib import Path

def run(cmd):
    """Run a shell command and return output."""
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.stdout.strip()

def get_all_components():
    """Get all component directories."""
    repo_root = Path(__file__).parent.parent
    components = [
        d.name for d in repo_root.iterdir()
        if d.is_dir() and not d.name.startswith('.')
    ]
    return sorted(components)

def get_changed_components():
    """Detect changed components based on git diff."""
    ref = os.getenv('GITHUB_REF', 'refs/heads/main')

    if ref in ('refs/heads/main', 'refs/heads/dev'):
        # For main/dev, compare HEAD~1...HEAD
        cmd = 'git diff --name-only HEAD~1...HEAD 2>/dev/null || git ls-tree -r --name-only HEAD'
    else:
        # For feature branches, compare origin/main...HEAD
        cmd = 'git diff --name-only origin/main...HEAD 2>/dev/null || echo ""'

    changed_files = run(cmd).split('\n')

    # Extract component names (first directory in path)
    components = set()
    for file_path in changed_files:
        if file_path:
            component = file_path.split('/')[0]
            # Filter out dot directories
            if not component.startswith('.'):
                components.add(component)

    return sorted(components)

def main():
    # Check for manual inputs
    build_all = os.getenv('BUILD_ALL', 'false').lower() == 'true'
    manual_components = os.getenv('MANUAL_COMPONENTS', '').strip()

    if build_all:
        components = get_all_components()
    elif manual_components:
        components = [c.strip() for c in manual_components.split(',')]
    else:
        components = get_changed_components()

    # Output as JSON array
    output = json.dumps(components)
    print(output)

if __name__ == '__main__':
    main()
