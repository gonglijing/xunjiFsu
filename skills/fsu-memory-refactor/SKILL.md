---
name: fsu-memory-refactor
description: Reduce memory usage in the fsu project. Use when tasks prioritize lower allocations or lower steady-state memory in collector, driver, database, or northbound hot paths, and when you need a repeatable workflow for caching static execution context, avoiding duplicate field-map copies, splitting cache-only write paths, and validating changes safely.
---

# FSU Memory Refactor

Use this skill when the first priority is reducing memory use in `fsu`.

## Follow this workflow

1. Start from the hottest path, not the largest file.
2. Prefer removing repeated allocations in per-collect execution before changing broad architecture.
3. Cache static per-device state when it is rebuilt on every collect.
4. Prefer typed structs over `map[string]interface{}` in JSON encode/decode paths.
5. Split fast paths from optional paths when a branch is hit on most collects.
6. Reuse maps only when behavior is unchanged; clone only for override paths.
7. Validate touched packages first, then run broader validation from `references/validation.md`.

## Target hotspots first

- `internal/collector`: task scheduling, runtime snapshots, collect pipeline, threshold checks, northbound command polling
- `internal/driver`: prepared device config, driver context, plugin input/output parsing, field normalization
- `internal/database`: cache-only writes, statement preparation, cleanup throttling
- `internal/northbound`: adapter snapshots, per-send copies, queue stats

## Preferred optimization patterns

- Move static device execution data from per-collect construction to task/device refresh time.
- Avoid duplicate `DriverResult -> map[string]string` conversions across packages.
- Keep cache-only writes free of history-specific work.
- Avoid generic `map[string]interface{}` decoding when only a few fields are needed.
- Delay map allocation until data actually exists.
- Remove stale queue/heap/task nodes eagerly if they can accumulate.

## Guardrails

- Preserve external API shape unless the task explicitly changes it.
- Preserve formatting behavior when collector and driver intentionally differ.
- Do not trade reduced allocations for unsafe shared mutable state across concurrent devices.
- If `go test ./internal/...` fails only on `internal/auth`, rerun `go test ./internal/auth -count=1` to check for test flakiness before blaming the refactor.
