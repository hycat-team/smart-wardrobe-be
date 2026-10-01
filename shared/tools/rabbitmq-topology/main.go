// Command rabbitmq-topology reconciles the queue types declared in the messaging
// registry (shared/infrastructure/messaging/topology.go) with the queue types the
// broker actually holds.
//
// RabbitMQ cannot change x-queue-type on an existing queue. Editing the registry
// to declare quorum against a broker that still holds a classic queue therefore
// makes every service that declares that topology fail at startup with
// 406 PRECONDITION_FAILED, and the only fix is drain → delete → redeclare. This
// command performs that comparison (check) and that migration (migrate), so the
// drift is caught in CI instead of on a crash-looping production box.
//
// Desired state is read from the live registry, never from a copy kept here, so
// the tool cannot drift from topology.go.
//
// The broker queue type is read from the Management HTTP API rather than
// amqp091-go's QueueInspect: declare-ok carries no type field, so AMQP alone
// cannot answer the only question this tool asks.
//
//	go run ./shared/tools/rabbitmq-topology --env-file .env.production check
//	go run ./shared/tools/rabbitmq-topology --env-file .env.production migrate --yes
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"smart-wardrobe-be/shared/infrastructure/messaging"

	"github.com/joho/godotenv"
)

const runbookPath = "deployments/scripts/rabbitmq-topology/README.md"

func main() {
	var (
		mode     = flag.String("mode", "check", "check | migrate")
		envFile  = flag.String("env-file", "", "path to env file holding the RABBITMQ_* settings (.env.production, .env.aws)")
		strict   = flag.Bool("strict", false, "treat MISSING queues as failures (they are ignored by default)")
		confirm  = flag.Bool("yes", false, "required for migrate: acknowledge that queues will be deleted and recreated")
		only     = flag.String("queue", "", "comma-separated queue names to act on (default: all)")
		spillDir = flag.String("spill-dir", "", "directory for drained messages (default "+DefaultSpillDir+"; set it when running in a container)")
		skipTLS  = flag.Bool("insecure-skip-tls-verify", false, "skip TLS verification (self-signed broker certificates)")
	)
	flag.Usage = usage
	flag.Parse()

	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			fatal("load env file %s: %v", *envFile, err)
		}
	}

	settings, err := settingsFromEnv(*skipTLS)
	if err != nil {
		fatal("%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	desired := desiredFromRegistry()
	types, messages, err := settings.FetchQueues(ctx)
	if err != nil {
		fatal("read queue types: %v", err)
	}

	plan := BuildPlan(desired, types, messages)
	filter := splitCSV(*only)
	entries, unknown := plan.Selected(filter)
	if len(unknown) > 0 {
		fatal("no such queue in the registry: %s", strings.Join(unknown, ", "))
	}

	fmt.Printf("broker: %s (vhost %s)  found=%d expected=%d\n",
		settings.Host, normalizeVhost(settings.Vhost), plan.Found, plan.Expected)
	fmt.Println()

	driftCount, missingCount, okCount := 0, 0, 0
	for _, e := range entries {
		if e.State == StateOK {
			okCount++
			continue
		}
		switch e.State {
		case StateMissing:
			missingCount++
			fmt.Printf("MISSING  %-42s want=%-7s (service creates it at startup)\n", e.Queue, e.Want)
		case StateUnknown:
			fmt.Printf("UNKNOWN  %-42s want=%-7s got=%q (management API reported an unrecognised type)\n", e.Queue, e.Want, e.Got)
		default:
			driftCount++
			fmt.Printf("DRIFT    %-42s want=%-7s actual=%-7s messages=%d\n", e.Queue, e.Want, e.Got, e.Message)
		}
	}
	if okCount > 0 {
		fmt.Printf("OK       %d queue(s) already match the registry\n", okCount)
	}

	switch *mode {
	case "check":
		fmt.Println()
		if driftCount > 0 {
			fmt.Printf("%d queue mismatch — RabbitMQ cannot change x-queue-type in place.\n", driftCount)
			fmt.Printf("Migrate is never automatic. See %s\n", runbookPath)
			os.Exit(1)
		}
		if *strict && missingCount > 0 {
			fmt.Printf("\n%d queue(s) missing (--strict)\n", missingCount)
			os.Exit(1)
		}
		fmt.Printf("\nrabbitmq-topology: OK — %d queue(s) match the registry\n", okCount)

	case "migrate":
		fmt.Println()
		if !*confirm {
			fmt.Fprintln(os.Stderr, "migrate deletes and recreates queues; re-run with --yes to confirm.")
			os.Exit(2)
		}
		if err := runMigrate(settings, entries, driftCount, *spillDir); err != nil {
			fatal("%v", err)

		}

	default:
		fatal("unknown mode %q (want check or migrate)", *mode)
	}
}

// runMigrate migrates only the drifted queues. MISSING is skipped on purpose:
// the service creates those itself, so recreating them here would just race the
// next deploy.
func runMigrate(settings Settings, entries []Drift, driftCount int, spillDir string) error {
	if driftCount == 0 {
		fmt.Println("rabbitmq-topology: nothing to migrate — every queue already matches.")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if n, err := settings.ConnectionCount(ctx); err == nil {
		fmt.Printf("open connections on this instance: %d (migrate needs one more)\n\n", n)
	}

	bindings := bindingByQueue()
	m, err := NewMigrator(settings, spillDir)
	if err != nil {
		return err
	}
	defer m.Close()

	migrated, totalDrained := 0, 0
	for _, e := range entries {
		if e.State != StateDrift {
			continue
		}
		binding, ok := bindings[e.Queue]
		if !ok {
			return fmt.Errorf("no registry binding for %s", e.Queue)
		}
		fmt.Printf("migrating %s (%s -> %s, %d message(s))...\n", e.Queue, e.Got, e.Want, e.Message)
		spillPath, drained, err := m.Migrate(binding, e.Want)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Queue, err)
		}
		migrated++
		totalDrained += drained
		fmt.Printf("  OK  drained=%d spill=%s\n", drained, spillPath)
	}

	fmt.Printf("\nrabbitmq-topology: migrated %d queue(s), replayed %d message(s).\n", migrated, totalDrained)
	if totalDrained > 0 {
		fmt.Printf("Spill files kept for audit under %s — delete them once verified.\n", m.spillIn)
	}
	fmt.Println("Restart the services so they bind to the recreated queues.")
	return nil
}

// bindingByQueue indexes the registry so a drifted queue name maps back to the
// args, routing keys and exchange needed to recreate it exactly.
func bindingByQueue() map[string]messaging.Binding {
	out := make(map[string]messaging.Binding)
	for _, b := range append(append([]messaging.Binding{}, messaging.Topology...), messaging.DLQTopology...) {
		out[b.Queue] = b
	}
	return out
}

// settingsFromEnv reads the RABBITMQ_* variables both .env.production (VPS,
// self-hosted broker) and .env.aws (CloudAMQP) already define, so the same code
// serves both environments.
func settingsFromEnv(skipTLSVerify bool) (Settings, error) {
	s := Settings{
		Host:          strings.TrimSpace(os.Getenv("RABBITMQ_HOST")),
		User:          strings.TrimSpace(os.Getenv("RABBITMQ_USER")),
		Password:      os.Getenv("RABBITMQ_PASSWORD"),
		Vhost:         os.Getenv("RABBITMQ_VHOST"),
		MgmtUser:      strings.TrimSpace(os.Getenv("RABBITMQ_USER")),
		MgmtPassword:  os.Getenv("RABBITMQ_PASSWORD"),
		SkipTLSVerify: skipTLSVerify,
	}
	if s.Host == "" {
		return s, fmt.Errorf("RABBITMQ_HOST is empty — pass --env-file .env.production (VPS) or .env.aws (AWS)")
	}
	if s.MgmtUser == "" {
		return s, fmt.Errorf("RABBITMQ_USER is empty — pass --env-file .env.production (VPS) or .env.aws (AWS)")
	}

	var err error
	if s.Port, err = intEnv("RABBITMQ_PORT", 0); err != nil {
		return s, err
	}
	if s.Port == 0 {
		s.Port = 5672
		if s.TLSEnabled, _ = boolEnv("RABBITMQ_TLS_ENABLED"); s.TLSEnabled {
			s.Port = 5671
		}
	}
	if s.TLSEnabled, err = boolEnv("RABBITMQ_TLS_ENABLED"); err != nil {
		return s, err
	}
	if s.MgmtTLS, err = boolEnv("RABBITMQ_MANAGEMENT_TLS_ENABLED"); err != nil {
		return s, err
	}
	if s.MgmtPort, err = intEnv("RABBITMQ_MANAGEMENT_PORT", 15672); err != nil {
		return s, err
	}
	return s, nil
}

func intEnv(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a number", key, raw)
	}
	return v, nil
}

func boolEnv(key string) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return false, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s=%q is not a boolean", key, raw)
	}
	return v, nil
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func usage() {
	fmt.Fprintf(os.Stderr, `rabbitmq-topology — compare and migrate RabbitMQ queue types.

Usage:
  rabbitmq-topology --env-file <path> [--mode check]  [flags]
  rabbitmq-topology --env-file <path> --mode migrate --yes [flags]

Modes:
  check    read-only; exit 1 when a queue type does not match the registry
  migrate  drain -> delete -> redeclare -> replay; requires --yes

Flags:
`)
	flag.PrintDefaults()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "rabbitmq-topology: "+format+"\n", args...)
	os.Exit(2)
}
