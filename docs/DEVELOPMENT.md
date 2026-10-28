# Local development preparation

The current machine has Go 1.26.3, Node 24.15.0, pnpm 11.1.3, and Docker CLI
29.8.2. `psql` is absent. Docker Engine access could not be verified from this
sandbox, so the Compose file has not been started. Compose syntax validation
passed with a placeholder password. No other project was changed.

## Isolated resources

| Service | Planned host binding | Status checked 22 Oct 2026 |
| --- | --- | --- |
| Frontend | `127.0.0.1:5181` | No listener seen |
| Go API | `127.0.0.1:8089` | Moved from 8088 on 27 Oct: 8088 is now used by another project's container |
| PostgreSQL | `127.0.0.1:5440` | No listener seen |

The Compose project is named `framecheck` and uses only the
`framecheck_pg_data` volume. PostgreSQL binds to loopback. Recheck ports before
starting because another process may take them later. Other applications were
observed on 5173, 5432, 5438, 8080, and 8087; do not stop them.

## When the build gate opens

1. Recheck the Official Rules and record the allowed build period and submission
   requirements in `RULES-AND-ACCESS.md`.
2. Confirm Meta account entitlement and actual credit balance/terms in Billing.
   The participant credit form says a payment method is required for the first
   API call. Obtain explicit authorization before adding one or making a paid
   request. A spend alert does not enforce a hard cap.
3. Copy `.env.example` to `.env`, replace the database password placeholder in
   both `FRAMECHECK_DB_PASSWORD` and `FRAMECHECK_DATABASE_URL`, and keep any
   real API key only in server environment configuration. `.env` is ignored by
   Git. Leave `MODEL_API_ENABLED=false` until the access gate is cleared.
4. Recheck selected ports. Start only this database with
   `docker compose --project-name framecheck up -d postgres` when Docker Engine
   is available. Use `docker compose --project-name framecheck down` for this
   project only; do not remove volumes casually.
5. Implement migrations and the vertical slice, then verify image validation,
   evidence mapping, review persistence, and export against an unchanged control.

`storage/` is an ignored local upload directory. Do not put secrets, personal
screenshots, or raw provider responses in committed fixtures.

## Running the first slice (27 Oct 2026)

```sh
# from the framecheck/ folder; .env holds a random local DB password
docker compose --project-name framecheck up -d postgres
cd framecheck_BE && go run ./cmd/api        # 127.0.0.1:8089, applies migrations
cd framecheck_FE && pnpm install && pnpm dev # 127.0.0.1:5181, proxies /v1 to the API
```

Checks: `go test ./...` and `go vet ./...` in `framecheck_BE`;
`pnpm typecheck`, `pnpm test`, `pnpm build` in `framecheck_FE`.

The API reads `../.env` (or `.env`) without overriding variables already set.
`FRAMECHECK_PROVIDER` picks the provider: `mock` (default; a
pixel-difference heuristic labelled `origin: mock`), `fixture` (replays
`demo/fixtures` ground truth for those exact images, `origin: fixture`, and
fails a run with `NO_FIXTURE` for any other pair), or `meta` (Muse Spark through
the Meta Model API, `origin: meta_api`; see below). `framecheck_FE/pnpm-workspace.yaml` allows
only esbuild's install script, which pnpm 11 blocks by default.

## Enabling Muse Spark

The adapter is tested against a local fake server and made its first real
request on 28 October 2026 (see `PROGRESS.md`). The Meta account needs a payment
method on file even while free credits are used. In `.env` only:

```sh
FRAMECHECK_PROVIDER=meta
MODEL_API_ENABLED=true
MODEL_API_KEY=...            # server side only; never in the frontend or logs
MODEL_API_MAX_REQUESTS=3     # start tiny: e.g. control + one missing element + one spare
```

The server logs a warning at startup, refuses to start without the key, and
fails runs with `REQUEST_CAP_REACHED` once the cap is used (the count resets on
restart). Each run is one request with both images. The first real run (two 2874×1294
images) used 5,225 input and 1,745 output tokens and took 13 s. Pricing has not
been checked.

## Resetting local demo data

`scripts/clean-demo-data.sh` lists every project except the sample project
with the most runs (dry run). `--yes` writes a `pg_dump` backup to
`framecheck_BE/storage/backups/` (ignored by Git), deletes the other projects
with their runs, findings and reviews, and removes image files no remaining
asset references. `--keep <project-id>` (repeatable) keeps other projects.
