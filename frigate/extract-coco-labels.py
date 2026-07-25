#!/usr/bin/env python3
"""Extract COCO class names from a YAML file."""

import sys
import yaml

def main():
    input_path = sys.argv[1] if len(sys.argv) > 1 else "data/coco.yaml"
    output_path = sys.argv[2] if len(sys.argv) > 2 else "coco.txt"

    with open(input_path, 'r') as f:
        data = yaml.safe_load(f)

    names = data.get('names', {})
    with open(output_path, 'w') as f:
        for i in range(len(names)):
            f.write(f"{names[i]}\n")

    print(f"Extracted {len(names)} class names to {output_path}")

if __name__ == "__main__":
    main()
