# Submodule Checklist

## Child repository

```bash
git -C <child-path> status --short
git -C <child-path> log --oneline -n 1
git -C <child-path> push origin main
```

## Parent repository

```bash
git add .gitmodules <child-path>
git submodule absorbgitdirs <child-path>
git submodule init <child-path>
git submodule status
git ls-files --stage <child-path>
```

## Push parent

```bash
git commit -m "chore: update <child-path> submodule"
git push
```
