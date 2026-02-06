---
name: gogw-feature-refactor
description: Refactor and implement features in the gogw project (Go backend, WASM driver runtime, northbound, and SolidJS frontend). Use when tasks involve handler splitting, driver runtime and northbound management APIs, route registration changes, frontend API/data-flow cleanup, and binary-size-safe build validation.
---

# GoGW Feature Refactor

Use this skill to deliver backend + frontend + WASM changes in `gogw` with low regression risk.

## Follow this workflow

1. Read `references/project_map.md` to locate impacted modules.
2. Implement changes in small, focused files.
3. Keep route signatures stable unless API changes are requested.
4. Run `scripts/verify.sh` from repo root.
5. If full test suites are noisy, run targeted build/validation from `references/validation.md`.

## Apply project conventions

- Split large handlers by responsibility (`*_crud.go`, `*_runtime.go`, `*_helpers.go`, `*_file.go`, `*_exec.go`).
- Reuse existing normalization helpers instead of duplicating validation logic.
- Keep `internal/app/http.go` focused on middleware/static/page wiring and move API route groups to `internal/app/routes_api.go`.
- Maintain existing API response envelope (`success`, `data`, `error`) via shared helpers in `internal/handlers/response.go`.
- Preserve small-binary goal: keep build flags consistent and validate `make build-mini`.

## Output checklist

- Changed files remain scoped to requested feature/refactor.
- Backend compiles (`go build`).
- Frontend bundles (`npm --prefix ui/frontend run build`).
- Mini binary build remains green and size-aware (`make build-mini`).
- Documentation is updated when API routes change.
