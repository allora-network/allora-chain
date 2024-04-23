#!/bin/env python3
import random
import json
import os
from datetime import datetime
import pytz

# Linear function parameters
a = 2
b = 3
MAX_DEVIATION = int(os.environ.get('MAX_DEVIATION', 1))

if __name__ == "__main__":
    try:
        # if len(sys.argv) < 1:
        #     raise Exception("Missing command arguments")
        # args = [arg.strip() for arg in sys.argv[2].split(",")]

        deviation = random.uniform(0, MAX_DEVIATION)-MAX_DEVIATION/2

        tzGMT = pytz.timezone("Etc/GMT")
        nowInSec = datetime.now(tzGMT).timestamp()

        # Calculate linear function of current time
        inference = a*int(nowInSec) + b + deviation
        print({"value": f"{inference}"})

    except Exception as e:
        print(json.dumps({"error": f"Error processing request: {e}"}))
