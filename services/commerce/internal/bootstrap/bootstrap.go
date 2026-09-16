// Package bootstrap owns the Commerce Core service app lifecycle.
//
// Composition root wiring (Wire) arrives with US5 (T041); module code
// moves under internal/modules in T032.
package bootstrap

// ServiceName identifies the service in logs and health payloads.
const ServiceName = "commerce"
