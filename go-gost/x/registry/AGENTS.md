# GO-GOST REGISTRY KNOWLEDGE BASE

**Generated:** Wed Feb 04 2026

## OVERVIEW
Central registration point for all pluggable GOST components (handlers, listeners, dialers, etc.).
Allows the configuration system to resolve string types (e.g., "socks5") to actual Go implementations.

## STRUCTURE
One file per component type, exporting a standard Registry interface.
```
go-gost/x/registry/
├── handler.go    # RegisterHandler(name, newFunc)
├── listener.go   # RegisterListener(name, newFunc)
├── dialer.go     # RegisterDialer(name, newFunc)
└── ...           # Same pattern for auth, bypass, admission
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Register a new component | `go-gost/x/registry/{type}.go` | Use `Register{Type}(name, creator)` |
| Component lookup | `go-gost/x/registry/{type}.go` | `Get{Type}(name)` returns the creator function |
| Default registrations | `go-gost/x/` (init functions) | Most components register themselves in their package `init()` |

## CONVENTIONS
- Thread-safe maps used for storage.
- Names are case-sensitive (usually lowercase).
- Components must be registered *before* the configuration parser runs (usually done via `import _ "..."` in `main.go`).
