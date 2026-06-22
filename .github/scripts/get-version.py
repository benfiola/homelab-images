#!/usr/bin/env python3
import os
import subprocess
import sys
from pathlib import Path

def run(cmd):
    """Run a shell command and return output."""
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return result.returncode, result.stdout.strip(), result.stderr.strip()

def main():
    component_name = os.getenv('COMPONENT_NAME', '')
    rc = os.getenv('RC', '').strip()
    alpha = os.getenv('ALPHA', '').strip()
    metadata = os.getenv('METADATA', '').strip()

    # Validate ALPHA and METADATA are together
    if alpha and not metadata:
        print("error: ALPHA requires METADATA to be set", file=sys.stderr)
        sys.exit(1)
    if metadata and not alpha:
        print("error: METADATA requires ALPHA to be set", file=sys.stderr)
        sys.exit(1)

    # Build svu command flags
    flags = []
    if rc:
        flags.append(f"--prerelease rc.{rc}")
    if alpha:
        flags.append(f"--prerelease alpha.{alpha}")
    if metadata:
        flags.append(f"--metadata {metadata}")

    # Build svu command
    cmd = (
        f"svu next "
        f"--tag.prefix={component_name}/v "
        f"--tag.pattern={component_name}/v* "
        f"--tag.output=v "
        f"--always=true "
    )
    if flags:
        cmd += " ".join(flags)

    # Execute svu
    returncode, stdout, stderr = run(cmd)

    if returncode == 0:
        print(stdout)
    else:
        # If svu fails, construct a valid semver from the base version
        base_version = "v1.0.0"

        if rc:
            print(f"{base_version}-rc.{rc}")
        elif alpha and metadata:
            print(f"{base_version}-alpha.{alpha}+{metadata}")
        elif alpha:
            print(f"{base_version}-alpha.{alpha}")
        else:
            print(base_version)

if __name__ == '__main__':
    main()
