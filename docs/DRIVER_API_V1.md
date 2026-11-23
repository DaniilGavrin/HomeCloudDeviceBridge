# HomeCloud Device Bridge — Driver API v1

**Runtime:** statically linked Go driver packages inside one Bridge process. No dynamic executable plugins in v1.

## Responsibilities and package boundaries

A driver adapts one protocol/profile to HomeCloud Device Contracts. It does not own global HomeCloud identity, permissions, UI, Backend storage or authorization.

Suggested package boundaries: `internal/devicecontracts`, `internal/driver`, `internal/registry`, `internal/discovery`, `internal/pairing`, `internal/router`, `internal/integrations/ble`, `internal/integrations/wifi`. Create packages as functionality lands, not as empty scaffolding.

## Design-level Go API

```go
type Driver interface {
    ID() string
    Start(ctx context.Context, host Host) error
    Stop(ctx context.Context) error
}

type Host interface {
    Registry() DeviceRegistry
    Candidates() CandidateSink
    Events() EventSink
    Logger() *slog.Logger
}

type DeviceDriver interface {
    Discover(ctx context.Context) error
    Pair(ctx context.Context, candidate Candidate, input PairingInput) (PairedDevice, error)
    Connect(ctx context.Context, device LocalDevice) (Connection, error)
    Execute(ctx context.Context, device LocalDevice, command Command) (CommandResult, error)
    ReadState(ctx context.Context, device LocalDevice, entities []string) ([]EntityState, error)
}

type Connection interface {
    Done() <-chan struct{}
    Close() error
}
```

Exact names may align with actual repository conventions, but semantics are normative. A protocol adapter may manage many devices and split transport driver from per-device profile implementation.

## Lifecycle and concurrency

- Start once per configured integration; initialize then run context-owned workers.
- Stop is idempotent and releases sockets, scans, subscriptions, timers and goroutines.
- Every goroutine has an owner, cancellation path and bounded shutdown.
- No `os.Exit`; drivers do not mutate Backend state.
- Mutating operations are serialized per device in v1. Independent devices may execute concurrently within explicit adapter limits.
- Avoid one goroutine per idle device unless the transport requires it.
- Blocking protocol calls must honor context cancellation or run behind bounded workers.

## Discovery and pairing defaults

Candidate includes random candidate ID, integration/transport, opaque native identity hint, safe display metadata, first/last seen, expiry and pairing methods.

- Candidate TTL: 90s after last observation; configurable 30–600s per integration.
- Pairing session: 5m default, one extension, hard maximum 10m.
- One active pairing per adapter/coordinator by default; raise only when stack safely supports it.
- Candidate is revalidated immediately before pairing.
- Cancellation stops protocol work and releases temporary resources.

## Credentials and identity

One-time PIN/QR values are not persisted after pairing. Long-lived bond keys/tokens are stored only when protocol requires them, accessible only to Bridge service account; files mode 0600 and parent directory 0700 where applicable. Never log secrets. If required credentials cannot be stored safely on a platform, mark integration unavailable rather than writing plaintext secrets to ordinary config/logs.

Identity quality is `stable`, `session_only` or `weak`. Never synthesize stable identity from volatile IP or random BLE address. Resolve BLE private addresses through platform bond/identity mechanisms when possible. Prefer protocol-issued/authenticated identifiers for Wi-Fi devices. Backend assigns HomeCloud `device_id`.

## Command behavior

Router resolves HomeCloud device ID through persistent mapping and checks runtime availability. Backend validates permission, capability, operation and schema; Bridge/driver revalidate mapping, runtime capability, revision, parameters and deadline. Return definitive success only with protocol evidence; uncertain write outcome returns `unknown`. No offline command queue.

## BLE profile boundary

BLE integration handles scanning and transport. Concrete GATT profiles declare services/characteristics and map them to normalized capabilities. Pair/bond only when required. Subscribe to notifications where supported and release subscriptions on disconnect/Stop. A device without a supported profile may be discovered but remains unsupported; never guess writable characteristics.

## Wi-Fi/LAN profile boundary

Wi-Fi is a transport family, not one device protocol. Each local API/profile is a separate integration. Discovery may use mDNS/DNS-SD, SSDP, broadcast, carefully bounded subnet scan or manual address entry according to profile. No aggressive broad scan by default. Prefer authenticated identity and TLS where available. Validate target addresses and prevent arbitrary URL/SSRF-style access from device metadata. HTTP/TCP/WebSocket clients require connect/read/write timeouts. Local integrations must not require vendor cloud access.

## Tests

Every driver provides unit tests for identity, capability mapping, parameter validation and error mapping; fake-based discovery/pairing tests; cancellation/shutdown tests; timeout/unknown-result tests; tests that unsupported characteristics/endpoints are not executable; and shared Device Contracts fixture tests. Hardware-in-the-loop tests are required before claiming support for a concrete device model.