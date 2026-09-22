# local-platform-lab-app-template

A **template** app-team service repo for the [local-platform-lab](https://github.com/chliddle/local-platform-lab)
platform. Demonstrates and enables self-service deployment: this repo
owns its own app code, tests, container build, semantic versioning, and
deployment manifests end to end -- the platform repo only needs an
`Application` pointer (see [Onboarding](#onboarding-a-new-app-platform-team)
below) to pick it up, and never needs to be touched again for routine
releases.

**If you're an app team onboarding a new service:** click "Use this
template" on GitHub to create your own repo from this one, then push.
A one-time `template-init` workflow renames everything (Go module path,
app/binary/Kubernetes-object name, container image) to match your new
repo automatically, then no-ops on every push after (guarded by a
sentinel file, not self-deletion -- the default `GITHUB_TOKEN` can never
modify files under `.github/workflows/`, under any permissions
configuration). From then on, all you touch is `main.go`/`internal/` --
your application code. Nothing else needs any manual configuration.

## What's here

- `main.go`, `internal/` -- the app itself (a small Go HTTP service:
  `/`, `/health`, `/ready`, `/version`, `/metrics`) -- **replace this with
  your own app's logic**; everything below works unmodified
- `Dockerfile` -- multi-arch (amd64/arm64) build, cross-compiled to avoid
  QEMU emulation
- `deploy/base/` -- Kustomize base (Deployment, Service)
- `deploy/overlays/dev/`, `deploy/overlays/prod/` -- per-environment
  overlays; `images:` here is the pinned digest Argo CD deploys
- `scripts/smoke-test.sh` -- asserts the platform's app contract (the 5
  endpoints above); generic, no per-app changes needed
- `.github/workflows/ci.yml` -- lint, test, ephemeral-cluster smoke test,
  build, push to GHCR, patch the digest into `deploy/overlays/dev`
- `.github/workflows/release.yml` -- runs after `ci.yml` succeeds;
  semantic-release cuts a version from Conventional Commits, then promotes
  the *same* GHCR digest (never rebuilt) into `deploy/overlays/prod`
- `.github/workflows/template-init.yml` -- exists only in the template;
  runs once on a newly-generated repo's first push, then removes itself

## The pipeline

```
push to main (feat:/fix:/...)
  -> ci: lint, test, ephemeral-cluster smoke test, build multi-arch image,
     push to GHCR (tag: commit SHA)
  -> ci: patch deploy/overlays/dev with the resulting digest, commit
  -> release (gated on ci succeeding): semantic-release computes next version
       - if a release is warranted:
           -> re-tag the same GHCR digest with the semver (no rebuild)
           -> patch deploy/overlays/prod with the same digest, commit
  -> Argo CD (running in the platform repo's dev/prod clusters) reconciles
     both overlays automatically
```

No manual approval gate: trunk-based development wants small changes to
ship often, and a human-in-the-loop step just creates a queue. The
promotion gate is `ci.yml` passing -- lint/test/ephemeral-cluster smoke
test -- not yet a live canary/rollout health check (that arrives with
Argo Rollouts, Milestone 5 in the platform repo).

**Commit messages on `main` must follow [Conventional Commits](https://www.conventionalcommits.org/)**
(`feat:`, `fix:`, `chore:`, ...) or `release.yml` never cuts a version.

## Local development

```bash
make build   # go build
make test    # go test ./...
make run     # build + run on :8080
```

```bash
docker build -t hello-world:local .
docker run -p 8080:8080 -e ENVIRONMENT=local hello-world:local
```

## Pre-commit hooks

[gitleaks](https://github.com/gitleaks/gitleaks) scans every commit for
hardcoded secrets before it's made. [zizmor](https://github.com/zizmorcore/zizmor)
scans every workflow change for dangerous GitHub Actions patterns
(`pull_request_target` misuse, template injection, excessive
permissions, unpinned actions) -- the same check also runs as a CI job
(`security lint`), with online audits enabled there. **Not** a required
status check here specifically: this repo's own automation (`ci.yml`'s
digest-bump commit, `release.yml`'s prod promotion, `template-init.yml`)
pushes bot-authored commits directly to `main`, and GitHub rejects any
direct push whose commit hasn't already had a required check run against
it -- which a just-created commit never has. Required on the platform
repo (`local-platform-lab`), which has no such automation. Both hook
layers carry over automatically to every repo generated from this
template.

```bash
brew install pre-commit   # or: pip install pre-commit
pre-commit install        # once per clone -- wires the hook into .git/hooks/
```

## Onboarding a new app (platform team)

The app team's side is fully automatic (see above) -- these are the
platform-team steps, in `local-platform-lab`:

1. **Add an `Application` manifest** under `gitops/dev/apps/` and
   `gitops/prod/apps/` pointing at the new repo's `deploy/overlays/dev` /
   `deploy/overlays/prod`. No new secret needed -- Argo CD's repo
   credentials are a URL-prefix template already covering any repo under
   this GitHub account.
2. **If the new app needs its own namespace** (the default `deploy/base/`
   this template ships with names Kubernetes objects, including the
   namespace, after the app -- see each overlay's `namespace:` field):
   add a `kubernetes_namespace_v1` + `kubernetes_secret_v1` (GHCR pull
   secret) for it in `terraform/environments/{dev,prod}/main.tf`, copying
   the existing `hello_world`/`ghcr_pull` pattern. This is the one place
   onboarding still touches Terraform -- always on the platform side,
   never the app team's.

## License

[MIT](LICENSE) -- carries over automatically to every repo generated
from this template.
