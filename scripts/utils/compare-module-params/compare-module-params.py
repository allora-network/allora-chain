#!/usr/bin/env python3

import subprocess
import json
import sys
import os
from typing import Dict
from difflib import unified_diff
import shutil
from datetime import datetime

MODULES = [
    "auth", "bank", "consensus", "distribution",
    "emissions", "feemarket", "gov", "ibc-transfer", "mint", 
    "slashing", "staking"
]

ALLORAD_CMD_1 = os.environ.get("ALLORAD_CMD_1", shutil.which("allorad"))
ALLORAD_CMD_2 = os.environ.get("ALLORAD_CMD_2", shutil.which("allorad"))

if ALLORAD_CMD_1 is None or ALLORAD_CMD_2 is None:
    print("Error: allorad binary not found. Set ALLORAD_CMD_1 and ALLORAD_CMD_2 or ensure 'allorad' is in PATH.", file=sys.stderr)
    sys.exit(1)

def fetch_module_params(cmd: str, rpc: str, module: str) -> Dict:
    try:
        result = subprocess.run(
            [cmd, "q", module, "params", "--node", rpc, "--output", "json"],
            capture_output=True, check=True, text=True
        )
        return json.loads(result.stdout)
    except subprocess.CalledProcessError as e:
        print(f"Error querying module '{module}' from {rpc} using {cmd}: {e.stderr}", file=sys.stderr)
        return {}
    except json.JSONDecodeError as e:
        print(f"Failed to parse JSON output from module '{module}' at {rpc} using {cmd}: {str(e)}", file=sys.stderr)
        return {}

def compare_dicts(dict1: Dict, dict2: Dict, module: str, output_lines: list):
    json1 = json.dumps(dict1, indent=2, sort_keys=True).splitlines(keepends=True)
    json2 = json.dumps(dict2, indent=2, sort_keys=True).splitlines(keepends=True)

    diff = list(unified_diff(
        json1, json2,
        fromfile=f"{module} (RPC 1)",
        tofile=f"{module} (RPC 2)",
        lineterm=""
    ))

    if diff:
        output_lines.append(f"\n=== Differences in module: {module} ===\n")
        output_lines.extend(diff)
    else:
        output_lines.append(f"Module {module}: ✅ No differences.\n")

def main():
    if len(sys.argv) < 3 or len(sys.argv) > 4:
        print("""
              Usage: diff-allora-modules <rpc1> <rpc2> [output_file]
              Accepts env vars: ALLORAD_CMD_1, ALLORAD_CMD_2
              Example: ALLORAD_CMD_1=allorad-v0.11.0 ALLORAD_CMD_2=allorad-v0.12.0 diff-allora-modules https://network-v0.11.0:26657 http://network-v0.12.0:26657/ /tmp/allora-diff-report-20250515-101010.log
              """, file=sys.stderr)
        sys.exit(1)

    rpc1, rpc2 = sys.argv[1], sys.argv[2]
    timestamp = datetime.utcnow().strftime("%Y%m%d-%H%M%S")
    default_output = f"allora-diff-report-{timestamp}.log"
    output_filename = sys.argv[3] if len(sys.argv) == 4 else default_output
    output_lines = []

    header = (
        f"Comparing modules between:\n- RPC 1: {rpc1} (using {ALLORAD_CMD_1})\n"
        f"- RPC 2: {rpc2} (using {ALLORAD_CMD_2})\n"
    )
    output_lines.append(header)

    for module in MODULES:
        params1 = fetch_module_params(ALLORAD_CMD_1, rpc1, module)
        params2 = fetch_module_params(ALLORAD_CMD_2, rpc2, module)

        if params1 == {} or params2 == {}:
            output_lines.append(f"Skipping module {module} due to previous errors.\n")
            continue

        compare_dicts(params1, params2, module, output_lines)

    with open(output_filename, "w") as f:
        for line in output_lines:
            f.write(line if line.endswith("\n") else line + "\n")

    print(f"\n✅ Diff report written to {output_filename}")

if __name__ == "__main__":
    main()
