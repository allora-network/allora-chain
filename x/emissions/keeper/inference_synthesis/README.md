# Inference Synthesis

This package contains the logic from the Inference Synthesis Section of the Allora litepaper, which combine current inferences, forecasts, previous losses and current regrets into current network losses and new latest regrets per worker or combination of workers (and topic).

# Incoming Data Limitation

The internal numerical data uses a large Decimal type. However, the incoming data layer imposes limitations on the values that can be sent.

# Benchmarks

Benchmark tests in this package are opt-in and are not executed by `go test ./...`.

Run all benchmarks in this package:

`go test ./x/emissions/keeper/inference_synthesis -run '^$' -bench . -benchmem`

