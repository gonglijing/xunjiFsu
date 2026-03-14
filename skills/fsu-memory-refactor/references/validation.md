# Validation

Use the narrowest command set that covers the touched path:

- `go test ./internal/collector -count=1`
- `go test ./internal/driver -count=1`
- `go test ./internal/database -count=1`
- `go test ./internal/... -count=1`

When full `internal/...` fails only on `internal/auth`, rerun:

- `go test ./internal/auth -count=1`

If the change affects the device runtime UI or API payloads, also run:

- `go test ./internal/handlers -count=1`
- `npm --prefix ui/frontend run build`
