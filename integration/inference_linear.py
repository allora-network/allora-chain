import random
import os
from datetime import datetime
import pytz

# Linear function parameters
a = 2
b = 3
MAX_DEVIATION = os.environ.get('MAX_DEVIATION', 1)

if __name__ == "__main__":
    try:
        # if len(sys.argv) < 1:
        #     raise Exception("Missing command arguments")
        # args = [arg.strip() for arg in sys.argv[2].split(",")]

        import random

        deviation = random.random(MAX_DEVIATION)

        tzGMT = pytz.timezone("Etc/GMT")
        nowInSec = datetime.now(tzGMT).timestamp()

        # Calculate linear function of current time
        inference = a*int(nowInSec) + b + deviation
        print({"value": f"{inference}"})

    except Exception as e:
        print(json.dumps({"error": f"Error processing request: {e}"}))
