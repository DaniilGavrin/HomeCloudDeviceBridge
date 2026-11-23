# HomeCloud Device Contracts v1

## Normative source and repository layout

There is deliberately **no standalone contracts repository and no shared Go module**. Backend and Device Bridge each implement the same versioned model in their own `internal/devicecontracts` package. The normative semantics are maintained in this document; JSON Schemas and fixtures are mirrored byte-for-byte in both repositories. Both CI pipelines run the same contract fixtures and a parity/checksum check. Drivers exist only in Device Bridge.

## Independent version axes

- `contracts_version`: Device Contracts model version (initially 1).
- `device_type.version`: definition version of one functional Device Type.
- `capability.version`: definition version of one Capability.
- `capability_revision`: monotonically increasing revision of one device's declared capability set.
- Backend WS envelope: existing `v=1`.
- Backend ↔ Bridge payload contract: Bridge Protocol v1.

## Device Type

Device Type is functional, not vendor/model-specific: `light`, `smart_plug`, `temperature_sensor`, `motion_sensor`, etc. Manufacturer, model, hardware revision and firmware are instance metadata. Default capabilities are hints only; they do not authorize capabilities.

Unknown Device Type is retained as an unsupported record with identity and metadata, but without executable capabilities.

## Capability

A Capability is a normalized HomeCloud function. Its definition includes stable ID/version, commands and parameter schemas, entities, value types, units, ranges/normalization rules, allowlisted UI metadata and permission semantics.

Unknown Capability is rejected individually; other known valid capabilities in the same declaration may be accepted. Unknown capabilities never become executable.

## Entity and state

Entity is a typed value such as `switch_state`, `brightness`, `temperature`, `humidity`, `power`, `voltage`, or `current`. Its definition includes stable ID, value type, optional unit, range/enum, read/write semantics and associated capability.

Current state is stored separately from capability declaration. Each update carries device ID, entity ID, typed value, measured timestamp and capability revision. Backend validates membership, type, unit and constraints before persistence/publication.

## Commands and UI metadata

Commands are declared by Capability and have typed parameter schemas. Backend performs authorization and schema validation; Bridge/driver revalidate runtime support and constraints. UI metadata uses a fixed allowlist of HomeCloud control types. No arbitrary HTML, JavaScript, remote component names or executable UI payloads are allowed.

Effective UI actions are the intersection of device capabilities, current runtime support and user permissions. Capability does not imply permission.

## Compatibility

Additive optional metadata is compatible if unknown optional fields are safely ignored. Removing/renaming IDs, changing value meaning/units/command semantics, or making an optional field required is breaking and requires a new contract version or explicit migration. Unknown required versions are rejected.

## Required shared fixtures

Both repositories carry identical fixtures for:
- light with switch and brightness;
- smart plug with switch and power monitoring;
- temperature/humidity sensor;
- unknown Device Type;
- mixed known and unknown capabilities;
- invalid entity type/unit/range;
- invalid command parameters;
- unsupported contract version;
- capability revision mismatch;
- malformed identity claim.

Both local Go packages must pass decode/validate/round-trip tests against these fixtures.