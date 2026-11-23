# HomeCloud Device Bridge ↔ Backend Protocol v1

**Transport:** existing local Backend WebSocket JSON v1 over loopback. No public Bridge listener, Unix socket, named pipe or new framing.

## Envelope

```json
{"v":1,"id":"message-uuid","type":"command","action":"device.bridge.register","data":{}}
```

Use the Backend's existing request correlation and command/result/error/event conventions. `v` is the WS envelope version, not the Device Contracts version. Exact low-level response envelope behavior must match the current Backend WS implementation.

## Service actions

| Action | Direction | Purpose |
|---|---|---|
| `device.bridge.register` | Bridge → Backend | Privileged service registration after WS handshake |
| `device.bridge.ready` | Bridge → Backend | Registry loaded and reconciliation completed |
| `device.bridge.status` | Both | Status |
| `device.bridge.heartbeat` | Both | Control-channel liveness |
| `device.bridge.reconcile` | Bridge → Backend | Persistent mapping reconciliation |

Peer identity: `client_type=device_bridge`, `device_id=homecloud-device-bridge`, active session, loopback remote address, feature flag enabled. This is a routing/local trust boundary, not cryptographic process attestation. Ready requires process health, WS registration, service registration, registry load and successful reconcile.

## Pairing actions

- `device.pairing.candidate.upsert` Bridge → Backend
- `device.pairing.candidate.remove` Bridge → Backend
- `device.pairing.start` Backend → Bridge
- `device.pairing.cancel` Backend → Bridge
- `device.pairing.progress` Bridge → Backend
- `device.pairing.result` Bridge → Backend

Candidate fields: random temporary `candidate_id`, `integration_id`, transport, opaque identity hint, safe display metadata, first/last seen, expiry and supported pairing methods. Candidate ID never becomes HomeCloud device ID. Do not send credentials or raw sensitive advertisement payloads to UI.

Pairing start carries `session_id`, `candidate_id`, requesting user reference, selected timestamp and typed protocol input. Backend authorizes before sending; Bridge rechecks candidate existence/expiry before pairing. PIN/QR input is relayed through Backend only.

## Registration actions

- `device.registry.claim` Bridge → Backend
- `device.registry.result` Backend → Bridge
- `device.registry.changed` Bridge → Backend
- `device.registry.reconcile` both directions

Identity claim fields: `integration_id`, `native_device_id`, `identity_quality` (`stable`, `session_only`, `weak`), Device Type id/version, metadata, capability revision and declared capabilities. Backend alone assigns `device_id`. Only stable, unambiguous identity can auto-rebind. Unknown Device Type remains unsupported without executable capabilities; unknown capabilities are rejected individually.

Registration result includes accepted flag, canonical `device_id`, registry/capability revision, accepted/rejected capability IDs and status.

## Commands

- `device.command.execute` Backend → Bridge
- `device.command.accepted` Bridge → Backend
- `device.command.result` Bridge → Backend

Request fields: `command_id`, `device_id`, `capability_id`, `command_idempotency_key`, operation, typed parameters, capability revision and deadline.

Terminal states: `succeeded`, `failed`, `unknown`. `accepted` only means Bridge accepted responsibility, not physical completion. If the link is lost after a write and result cannot be proven, report `unknown`.

v1 defaults: Backend acceptance deadline 3s; driver execution deadline 10s (bounded integration override); one mutating command at a time per device; different devices may run concurrently. Duplicate command IDs are served from a bounded dedup cache and never executed twice. No automatic command retries and no offline queue.

## State and availability

- `device.state.changed` Bridge → Backend
- `device.availability.changed` Bridge → Backend
- `device.telemetry.batch` Bridge → Backend

State carries device/entity IDs, typed value, optional unit, measured timestamp and capability revision. Backend validates entity membership, type, unit and revision. Availability is independent from registry existence. Reconnect triggers full reconciliation before incremental events resume.

## Error codes

`bridge_not_ready`, `candidate_not_found`, `candidate_expired`, `pairing_in_progress`, `pairing_cancelled`, `pairing_failed`, `identity_conflict`, `identity_not_stable`, `device_not_registered`, `device_offline`, `capability_not_supported`, `capability_revision_mismatch`, `invalid_command`, `command_duplicate`, `command_timeout`, `execution_outcome_unknown`, `driver_unavailable`, `unsupported_contract_version`, `internal_error`.

Errors contain stable code and safe message; no secrets or sensitive protocol payloads.

## Reconnect and versioning

Bridge reconnects with bounded exponential backoff and jitter, repeats service registration, sends a full mapping/availability reconcile, then emits ready. Events are deltas; Backend persists canonical device records and last-known state. No cross-device total ordering is promised; per-device mutating commands are serialized.

Additive optional fields are allowed only if safely ignored. Breaking payload semantics require explicit protocol contract versioning. Envelope v1 is not reused as a contracts version.

## First transport families

BLE: advertisements and supported GATT profiles; bonding only where the selected profile requires it. Wi-Fi/LAN: discovery/control is integration-specific; no assumption that all Wi-Fi devices expose a common local API. Provisioning credentials use short-lived typed flow and are never logged. Zigbee, Matter and Thread are outside the first implementation scope.