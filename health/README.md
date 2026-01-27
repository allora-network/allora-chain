

# Nurse healthcheck service

`nurse` leverages Go's `pprof` tooling to monitor various system and application metrics.  The operator can configure per-resource thresholds which, when surpassed, trigger the gathering of a suite of metrics which are then dumped to disk along with a `nurse.log` file describing the reason for the dump.

To enable `nurse`, set the `NURSE_TOML_PATH` environment variable and create a `nurse.toml` file with the following schema:

```toml
profile-root           = "/tmp/nurse" # where the profiles will be dumped
poll-interval          = "1m"         # frequency with which to check thresholds
gather-duration        = "60s"        # how long to gather samples when profiling
max-profile-size       = "100mb"      # maximum size of the profile
cpu-profile-rate       = "5"          # samples per second
mem-profile-rate       = "5"          # samples per second
block-profile-rate     = "..."        # samples per second
mutex-profile-fraction = "..."        # fraction of mutex events that are reports (1 / rate)
mem-threshold          = "..."        # maximum memory usage of the process before a dump is triggered
goroutine-threshold    = "..."        # maximum number of goroutines before a dump is triggered
```

