# local-platform-lab-app-1

An example app-team service repo for the [local-platform-lab](https://github.com/chliddle/local-platform-lab)
platform. Demonstrates self-service deployment: this repo owns its own app
code, tests, container build, semantic versioning, and deployment manifests
end to end -- the platform repo only needs a single `Application` pointer
(see [Onboarding](#onboarding-a-new-app) below) to pick it up, and never
needs to be touched again for routine releases.

## What's here

- `main.go`, `internal/` -- the app itself (a small Go HTTP service:
  `/`, `/health`, `/ready`, `/version`, `/metrics`)
- `Dockerfile` -- multi-arch (amd64/arm64) build, cross-compiled to avoid
  QEMU emulation
- `deploy/base/` -- Kustomize base (Deployment, Service)
- `deploy/overlays/dev/`, `deploy/overlays/prod/` -- per-environment
  overlays; `images:` here is the pinned digest Argo CD deploys
- `.github/workflows/ci.yml` -- lint, test, build, push to GHCR, patch the
  digest into `deploy/overlays/dev`
- `.github/workflows/release.yml` -- runs after `ci.yml` succeeds;
  semantic-release cuts a version from Conventional Commits, then promotes
  the *same* GHCR digest (never rebuilt) into `deploy/overlays/prod`

## The pipeline

```
push to main (feat:/fix:/...)
  -> ci: lint, test, build multi-arch image, push to GHCR (tag: commit SHA)
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
promotion gate is `ci.yml` passing -- lint/test/build -- not yet a live
canary/rollout health check (that arrives with Argo Rollouts, Milestone 4
in the platform repo).

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

## Onboarding a new app (platform team)

Adding a new self-service app repo like this one to the platform requires
exactly one change in `local-platform-lab`: an `Application` manifest under
`gitops/dev/apps/` and `gitops/prod/apps/` pointing at this repo's
`deploy/overlays/dev` / `deploy/overlays/prod`. No Terraform changes --
Argo CD's repo credentials are a URL-prefix template covering every repo
under this GitHub account, not a per-repo secret.
