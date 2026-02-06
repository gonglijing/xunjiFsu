# Validation Commands

Run from repo root.

## Preferred quick regression

```bash
GOCACHE=/tmp/gocache go build ./internal/handlers ./internal/driver ./internal/app ./cmd/...
npm --prefix ui/frontend run build
GOCACHE=/tmp/gocache make build-mini
```

## When targeted tests are needed

```bash
GOCACHE=/tmp/gocache go test ./internal/driver ./internal/handlers ./internal/app
```

## Notes

- Some repository-wide tests may fail due to unrelated historical test files; isolate by package when required.
- Keep `GOCACHE=/tmp/gocache` in constrained environments.
