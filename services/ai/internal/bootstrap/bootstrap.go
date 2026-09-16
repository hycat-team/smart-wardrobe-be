// Package bootstrap owns the AI Engine service app lifecycle.
//
// Composition root wiring (Wire) arrives with US5 (T042); module code
// moves under internal/modules in T033.
package bootstrap

// ServiceName identifies the service in logs and health payloads.
const ServiceName = "ai"
