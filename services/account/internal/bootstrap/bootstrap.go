// Package bootstrap owns the Account service app lifecycle.
//
// Composition root wiring (Wire) arrives with US5 (T040); module code
// moves under internal/modules in T031.
package bootstrap

// ServiceName identifies the service in logs and health payloads.
const ServiceName = "account"
