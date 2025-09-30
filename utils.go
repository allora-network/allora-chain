// File: adapter/api-worker-reputer/utils.go

package api_worker_reputer

import (
    "log"
)

// LogStartup logs a startup message for the worker/reputer node.
func LogStartup(nodeType string) {
    log.Printf("Starting %s node for Allora Network...\n", nodeType)
}
