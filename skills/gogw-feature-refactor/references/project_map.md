# GoGW Project Map

## Runtime and app wiring

- `internal/app/app.go`: application bootstrap
- `internal/app/http.go`: middleware/static/page routing
- `internal/app/routes_api.go`: API route groups

## Backend domains

- `internal/handlers/`: HTTP handlers (split by domain)
- `internal/database/`: persistence APIs and schema-oriented helpers
- `internal/collector/`: collector loop and threshold processing
- `internal/northbound/`: plugin manager and runtime integration
- `internal/driver/`: WASM driver manager/executor/host functions

## Frontend

- `ui/frontend/src/pages/`: route pages
- `ui/frontend/src/sections/`: dashboard and domain sections
- `ui/frontend/src/components/`: shared UI components
- `ui/frontend/src/api.js`: API helper and response unwrapping

## External repos managed as submodules

- `plugin_north` -> `git@github.com:gonglijing/xunjiNorth.git`
- `drvs` -> `git@github.com:gonglijing/xunjiDrvs.git`
