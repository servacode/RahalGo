# Staging deployment channel (GitHub → box, staging-only)

> **Purpose.** After a one-time bootstrap, staging deploys need **no owner terminal work**.
> Claude pushes an exact SHA, triggers a GitHub Action, and the channel builds + deploys
> **staging only** and verifies it. Production is never touched.

---

## 1 · What was created

| File | Role |
|---|---|
| `deploy/staging/rahalgo-staging-deploy.sh` | The root-owned wrapper — the **only** thing the deploy identity can run (via `sudo`). Validates the SHA, refuses production, **cryptographically verifies the git bundle resolves to the SHA**, builds+deploys staging, verifies live identity, proves production unchanged (container-ID level). Fail-closed. |
| `.github/workflows/deploy-staging.yml` | `workflow_dispatch(sha)`: checkout exact SHA → **git bundle bound to the SHA** → stream to the wrapper over the restricted key → independently verify live staging identity == SHA. |
| `deploy/staging/bootstrap-staging-channel-one-shot.sh` | **Self-contained** one-time installer (no checkout): embeds the wrapper + public key + sudoers; creates the restricted identity; validates; PASS/FAIL. Generated from the wrapper (drift-guarded by the self-test). |
| `deploy/staging/bootstrap-staging-channel.sh` | Same install, for anyone who already has a repo checkout (reads the sibling wrapper + `.pub`). |
| `deploy/staging/rahalgo-staging-deploy.pub` | The deploy identity's **public** key (safe to commit); bootstrap installs it. The private key lives only in the GitHub `staging` secret. |
| `deploy/staging/wrapper-selftest.sh` | Off-box proof: SHA validation, production refusal, and the **bundle↔SHA binding** (content A + SHA B → rejected). **19/19 pass.** |

The wrapper reuses the existing, self-consistent build/deploy tools **from the verified commit**: `deploy/build-artifact.sh` (archive mode), `deploy/preflight-env.sh`, `deploy/promote.sh`. No git working tree is required on the box.

---

## 2 · One-time owner action — ONE command (no checkout needed)

Everything else (deploy keypair, GitHub secrets, host-key pinning) is already done by Claude.
On the box, as root:

```bash
wget -qO- https://raw.githubusercontent.com/servacode/RahalGo/stg-channel/deploy/staging/bootstrap-staging-channel-one-shot.sh | sudo bash
```

Expect the last line: **`BOOTSTRAP RESULT: PASS`**. It only prints PASS after a real **on-box
readiness** check runs through the exact deploy path (`deploy-user → sudo → wrapper as root`,
no staging mutation) and verifies: `go` resolves on the final PATH and builds; `HOME` is writable
for the Go cache; `git/tar/curl/docker/docker compose` are present; `.env.staging` normalizes and
parses (no values printed); docker works only as root (not directly as the deploy user); enough
disk/memory for the build (or a clear capacity warning/failure); and the fail-closed guards
(malformed SHA, production-tainted target) still hold. So PASS means **genuinely deploy-ready**,
not merely installed.

This **self-contained** installer embeds the root-owned wrapper + the deploy **public** key + the
sudoers policy — no git checkout, no repo, no file copying. The private key lives only in the GitHub
`staging` secret. If the console mangles the pipe, use two lines instead:

```bash
wget -qO /root/rg.sh https://raw.githubusercontent.com/servacode/RahalGo/stg-channel/deploy/staging/bootstrap-staging-channel-one-shot.sh
sudo bash /root/rg.sh
```

Prerequisites on the box: `docker`, `go`, and `/srv/rahalgo-staging/deploy/staging/.env.staging`
(already present). Idempotent. (The in-repo `deploy/staging/bootstrap-staging-channel.sh` remains
for anyone who already has a checkout.)

---

## 3 · GitHub configuration — already done

Claude configured the `staging` GitHub Environment and its secrets (staging only, never
production): `STAGING_DEPLOY_KEY` (private key piped via stdin — never printed),
`STAGING_SSH_HOST`, `STAGING_SSH_USER`, `STAGING_SSH_KNOWN_HOSTS` (pinned via `ssh-keyscan`),
`STAGING_IDENTITY_URL`. The local private key was deleted after upload. **Nothing for the owner
to add.** (Optionally add a required reviewer to the `staging` Environment for a manual approval
gate before each deploy.)

---

## 4 · How Claude triggers a staging deploy afterward

```bash
git push origin <branch>                       # exact approved SHA on the remote
gh workflow run deploy-staging.yml -f sha=<40-char-lowercase-sha>
gh run watch                                   # or poll gh run list
```

The workflow refuses anything that is not an exact 40-char lowercase SHA, deploys it to
staging, and **independently** confirms `environment=staging`, `source_commit=<sha>`, and a
`migration_version` before reporting success. Claude then runs the emulator live-witness batch.

---

## 5 · Rollback

- **App:** re-run the workflow with the previous good SHA. Images are keyed by commit
  (`release-<commit>`), so a prior build is reused; the wrapper re-verifies identity.
- **Schema:** migration `0159` is additive/nullable (`request_fingerprint bytea NULL`) — no
  reversal needed. Per house rule, never edit a merged migration; any reversal is a new
  forward migration. The wrapper never auto-drops columns.
- The channel itself is unaffected by app rollbacks (wrapper + key are installed once).

---

## 6 · Why production is excluded (defence in depth)

1. **Identity (no host-root equivalence):** the deploy user is **not in the docker group**,
   password-locked, with a forced-command key (`no-pty,no-port-forwarding,no-agent-forwarding,
   no-X11-forwarding`, one key). It can do exactly one thing: `sudo -n /usr/local/sbin/rahalgo-staging-deploy`.
   A sudoers rule permits **only** that wrapper. The bootstrap validates: `sudo -l` lists only the
   wrapper, the user has no direct docker, and cannot edit the wrapper.
2. **Source binding:** the wrapper accepts a **git bundle** and cryptographically verifies it
   resolves to the exact requested SHA (`refs/heads/deploy-target` == SHA, content-addressed)
   **before** building. Content from commit A under a request for SHA B is rejected (exit 14).
3. **Wrapper:** hard-codes `STAGING_ROOT=/srv/rahalgo-staging`; refuses any production-tainted
   `DATABASE_URL/REDIS_URL/COMPOSE_FILE/…` (exit 11); requires the staging DB
   (`:5534/rahalgo_staging`) or fails closed; runs as root only via the sudo forced command (exit 15 otherwise).
4. **envguard:** `stagingctl guard` (the app's own `MustBeSafe`) refuses a non-staging
   environment (exit 3).
5. **Preflight:** `preflight-env.sh` requires the target to report `environment=staging`
   **before** any mutation; `promote.sh` re-checks env before *and* after.
6. **Isolation proof:** the wrapper snapshots the production compose project (`rahalgo`) with
   **container ID + image ID + state** before and after, and **fails (exit 6) if any changed.**
7. **GitHub:** only the `staging` Environment's secrets are used; there is no production job and
   no production secret in this workflow.

Result: **Production mutations = 0**, structurally.

---

## Secret hygiene (verified 2026-09-21)

The real staging secrets live **only on the box**, never in Git:

- `deploy/staging/.env.staging` was **never committed** — `git log --all -- deploy/staging/.env.staging`
  is empty across all 1488 commits; it is gitignored (`.gitignore` lines for `.env`, `.env.*`, and an
  explicit `deploy/staging/.env.staging`). The only value-carrying copy is the untracked file on the box.
- The tracked, public `deploy/staging/.env.staging.example` has **empty** sensitive values
  (`STAGING_DB_PASSWORD=`, `STAGING_JWT_SECRET=`, `STAGING_ADMIN_PASSWORD=`) — a template, not secrets.
- Full-history scan (1488 commits): no private keys, no `.pem/.key/credential/serviceaccount`
  files, no known token formats. FCM credentials load at runtime from `FCM_CREDENTIALS_JSON`/
  `FCM_CREDENTIALS_FILE`, never committed.

This channel always uses the persistent `/srv/rahalgo-staging/deploy/staging/.env.staging` on the box
and never copies production secrets, so no staging secret ever enters Git through it.
