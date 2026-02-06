---
name: gogw-submodule-publisher
description: Manage gogw external repositories as submodules (plugin_north and drvs), including child-repo commit/push, parent-repo submodule pointer updates, .gitmodules maintenance, and safe release sequencing. Use when splitting code out of the main repo or updating submodule pointers.
---

# GoGW Submodule Publisher

Use this skill to safely publish `plugin_north` or `drvs` changes and sync pointers in the parent repository.

## Enforce release order

1. Commit and push the child repository first (`plugin_north` or `drvs`).
2. Confirm the target child commit exists on remote.
3. Update submodule pointer in parent repo.
4. Commit parent repo submodule pointer change.
5. Push parent repo.

## Keep parent repository clean

- Track only submodule gitlink, not child files.
- Ensure `.gitmodules` has correct `path`, `url`, and `branch`.
- Use `git submodule absorbgitdirs <path>` after converting embedded repo to submodule layout.
- Use `git submodule init <path>` so clone/update workflows remain stable.

## Verification checklist

- `git submodule status` shows expected commit.
- `git ls-files --stage <submodule-path>` shows mode `160000`.
- Parent branch and remote branch point to the same commit after push.
