import json
import subprocess
import sys
from pathlib import Path


def main():
    custom_components_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(".")

    if not custom_components_dir.is_dir():
        print(f"Error: Directory '{custom_components_dir}' does not exist", file=sys.stderr)
        sys.exit(1)

    print(f"Installing custom component dependencies from {custom_components_dir}")

    manifests = sorted(custom_components_dir.glob("*/manifest.json"))

    if not manifests:
        print("No custom components found")
        return

    for manifest_path in manifests:
        component_name = manifest_path.parent.name

        try:
            with open(manifest_path) as f:
                manifest = json.load(f)
        except json.JSONDecodeError as e:
            print(f"⚠ Warning: Failed to parse {manifest_path}: {e}", file=sys.stderr)
            continue

        requirements = manifest.get("requirements", [])

        if not requirements:
            continue

        print(f"Installing requirements for component: {component_name}")

        cmd = [
            sys.executable,
            "-m",
            "uv",
            "pip",
            "install",
            "--quiet",
            *requirements,
            "--index-strategy",
            "unsafe-first-match",
            "--upgrade",
            "--constraint",
            "/usr/src/homeassistant/homeassistant/package_constraints.txt",
        ]

        try:
            subprocess.run(cmd, check=True)
            print(f"✓ Installed {len(requirements)} dependencies for {component_name}")
        except subprocess.CalledProcessError:
            print(
                f"✗ Failed to install dependencies for {component_name}",
                file=sys.stderr,
            )
            sys.exit(1)

    print("Done installing custom component dependencies")


if __name__ == "__main__":
    main()
