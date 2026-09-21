# Staging deployment channel (GitHub → box, staging-only)

> **Purpose.** After a one-time bootstrap, staging deploys need **no owner terminal work**.
> Claude pushes an exact SHA, triggers a GitHub Action, and the channel builds + deploys
> **staging only** and verifies it. Production is never touched.

---

## 1 · What was created

| File | Role |
|---|---|
| `deploy/staging/rahalgo-staging-deploy.sh` | The root-owned wrapper — the **only** thing the deploy identity can run. Validates the SHA, refuses production, extracts the source archive, builds+deploys staging, verifies live identity, proves production unchanged. Fail-closed. |
| `.github/workflows/deploy-staging.yml` | `workflow_dispatch` action: checkout exact SHA → `git archive` → stream to the wrapper over the restricted key → independently verify live staging identity == SHA. |
| `deploy/staging/bootstrap-staging-channel.sh` | One-time owner script (run as root on the box): creates the restricted identity, installs the wrapper, pins the forced-command key, sets permissions, prints PASS/FAIL. |
| `deploy/staging/wrapper-selftest.sh` | Off-box proof of the wrapper's validation boundary (malformed SHA → 10, production-tainted → 11, clean → 0). **9/9 pass.** |

The wrapper reuses the existing, self-consistent build/deploy tools **from the deployed archive**: `deploy/build-artifact.sh` (archive mode), `deploy/preflight-env.sh`, `deploy/promote.sh`. No git working tree is required on the box.

---

## 2 · One-time owner bootstrap (do this once)

```bash
# a) Generate the dedicated deploy keypair (locally, on your machine)
ssh-keygen -t ed25519 -f rahalgo-staging-deploy -N "" -C rahalgo-staging-deploy

# b) On the box, as root, from a checkout of this repo at the approved SHA:
sudo STAGING_DEPLOY_PUBKEY="$(cat rahalgo-staging-deploy.pub)" \
     deploy/staging/bootstrap-staging-channel.sh
#    -> expect:  BOOTSTRAP RESULT: PASS

# c) Capture the box host key for pinning (no blind trust-on-first-use):
ssh-keyscan -t ed25519 195.201.141.130
```

Prerequisites on the box: `docker`, `go`, and `/srv/rahalgo-staging/deploy/staging/.env.staging` present (the persistent staging secrets — already there). The bootstrap is idempotent.

---

## 3 · GitHub configuration (repo Settings → Environments → `staging`)

Add these **Environment secrets** (staging only — never production):

| Secret | Value |
|---|---|
| `STAGING_DEPLOY_KEY` | contents of the **private** key `rahalgo-staging-deploy` |
| `STAGING_SSH_HOST` | `195.201.141.130` |
| `STAGING_SSH_USER` | `rahalgo-staging-deploy` |
| `STAGING_SSH_KNOWN_HOSTS` | the line printed by `ssh-keyscan` in step (c) |
| `STAGING_IDENTITY_URL` | `https://staging-api.rahalgo.com/api/v1/public/identity` |

Then delete the local private key. Optionally add a required reviewer to the `staging`
Environment if you want a manual approval gate before each deploy.

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

1. **Identity:** forced-command key → the deploy user can run **only** the wrapper; no shell,
   `no-pty,no-port-forwarding,no-agent-forwarding,no-X11-forwarding`; password locked; exactly
   one authorized key. The bootstrap validates all of this and that the user cannot edit the wrapper.
2. **Wrapper:** hard-codes `STAGING_ROOT=/srv/rahalgo-staging`; refuses any production-tainted
   `DATABASE_URL/REDIS_URL/COMPOSE_FILE/…` (exit 11); requires the staging DB
   (`:5534/rahalgo_staging`) or fails closed.
3. **envguard:** `stagingctl guard` (the app's own `MustBeSafe`) refuses a non-staging
   environment (exit 3).
4. **Preflight:** `preflight-env.sh` requires the target to report `environment=staging`
   **before** any mutation; `promote.sh` re-checks env before *and* after.
5. **Isolation proof:** the wrapper snapshots the production compose project (`rahalgo`)
   containers before and after and **fails (exit 6) if any identity changed.**
6. **GitHub:** only the `staging` Environment's secrets are used; there is no production job and
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
