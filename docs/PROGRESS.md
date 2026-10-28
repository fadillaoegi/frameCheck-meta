# FrameCheck progress log

## 22 October 2026 — pre-build preparation

### Changed

- Read `README.md`, `../research/FRAMECHECK-HANDOFF.md`, and
  `../research/META-GLOBAL-AI-HACKATHON-ANALISA-2026-10-16.md`.
- Checked parent/project `AGENTS.md`; none exists for FrameCheck. Other
  projects' `AGENTS.md` files were not applied.
- Rewrote the README with the product background and present status.
- Added product flow, FE/BE Clean Architecture, draft data/HTTP contracts,
  evaluation plan, rules/access log, development plan, `.env.example`,
  `.gitignore`, and isolated PostgreSQL Compose configuration. No app feature
  code was created.
- Created an ignored `storage/` directory for later local development.

### Verification

- Public Meta and Cerebral Valley event pages checked on 22 October: kickoff
  26 October, submission 4 November 21:00 ET, Official Rules still promised
  before kickoff. No published public rule text found in this check.
- Public Meta docs show hosted `muse-spark-1.3` image input and `sam-3.1`
  segmentation over `/v1/responses`. Account access was not tested.
- Audited listeners without changing them. Ports 5181, 8088, and 5440 had no
  listener; 5173, 5432, 5438, 8080, and 8087 were occupied.
- Audited tools: Go 1.26.3, Node 24.15.0, pnpm 11.1.3, Docker CLI 29.8.2;
  `psql` not installed. Docker Engine socket access denied in this sandbox,
  so container startup and Compose runtime validation remain unverified.
- `docker compose -f compose.yaml config --quiet` passed with a temporary
  placeholder password. This checks Compose syntax only, not database startup.
- Rechecked 5181, 8088, and 5440 after writing configuration; no listeners.
- The folder is not a Git repository. `.gitignore` is ready; no commit exists.

### Actual gaps / gates

1. Obtain and accept Official Rules, then resolve prior-code, model, and
   submission requirements before competition implementation.
2. Verify Meta credit activation, account model entitlement, card requirement,
   and pricing behavior. No paid request or payment action is authorized.
3. Verify Docker Engine access in the development session, recheck ports, and
   start only the FrameCheck Compose project.
4. Create the self-owned demo/fixtures and implement the first PostgreSQL-backed
   upload → real analysis → evidence → review slice after gates 1–2 permit it.
5. Measure model reliability and review value; no accuracy or savings claim yet.

### Pre-build declaration record

All files above were authored on 22 October 2026 as planning and environment
configuration before the announced 26 October kickoff. Declare these files as
prior work if the final rules require disclosure; assess their eligibility under
the actual rules before using them in a competition submission.

## 27 October 2026 — credit claim review

- Reviewed three user-provided screenshots without copying account identifiers
  or personal details into the repository. The claim form's Team ID matches the
  Meta dashboard URL shown. The application email and actual form submission
  remain unverified from the screenshots.
- Updated `RULES-AND-ACCESS.md`: claim form says payment method required for
  the first API request; credits arrive in batches after a submitted claim and
  must be confirmed in Billing. Credit balance is still unverified.
- Verified against Meta billing help that “Spend Limit” is an alert, not a hard
  cap. No card was added, no form submitted, and no API call made by this work.
- Current public event page still says Official Rules will be published before
  kickoff; participant-specific rule text was not available in these screenshots.
- User confirmed the credit claim was submitted and showed the Google Form
  `formResponse` page. Updated the access log: claim submitted; credit balance
  and API entitlement still unverified. No billing or API action was taken.

## 27 October 2026 — first vertical slice (mock provider)

### Changed

- Backend (`framecheck_BE`, Go, Clean Architecture): domain rules (capture
  matching, requirement versions, coordinate mapping 0..1000 → original pixels
  with floor/ceil and bounds checks, per-category evidence rules, run
  fingerprint), use cases (project, upload with byte-level PNG/JPEG checks,
  comparison, queued run with sequential worker, review, export), PostgreSQL
  adapter with embedded migration, local hash-addressed storage, and HTTP API
  matching `CONTRACTS.md`. Additions to the contract: `GET /v1/projects`,
  `GET /v1/projects/{id}`, `GET /v1/comparisons/{id}`,
  `GET /v1/assets/{id}/content`, and `allow_duplicate` on run start
  (`409 DUPLICATE_RUN` otherwise). Reviews need a reason only to reject.
- Offline mock provider: pixel-difference regions reported as
  `layout_mismatch` with `origin: mock`. It has no UI semantics.
- Frontend (`framecheck_FE`, React 19 + TS + Vite 7 + Tailwind 4): projects,
  upload/capture form, comparison builder, side-by-side evidence overlays,
  per-finding accept/reject with history, run provenance, Markdown/JSON export.
- API port moved to 8089. Created local `.env` with a random DB password and
  started only the `framecheck` Compose project (Postgres on 5440).
- No Git command was run (user handles Git). No Meta API call was made.

### Verification

- `go vet ./...` clean; `go test ./...` passes (domain, use cases with an
  in-memory repository, mock provider).
- `pnpm typecheck`, `pnpm test` (5 tests), and `pnpm build` pass.
- Manual API run against Postgres: upload, `CAPTURE_MISMATCH` 409, run to
  `completed` with box 240,288 160×64 px, `DUPLICATE_RUN` 409, Idempotency-Key
  replay 200, review 201, Markdown/JSON export with mock caveat.
- Headless Chrome: comparison page overlays align on both images; no
  horizontal scroll at 390 px on the three pages; the reject-without-reason
  guard and a reject with reason worked through the UI, and the history kept
  both decisions.

### Still open

1. Official Rules have still not been checked after the 26 Oct kickoff;
   prior-work and model requirements remain unverified.
2. Meta adapter (`muse-spark-1.3`) is not implemented; credits and entitlement
   are unverified, so no paid request is authorised.
3. No demo page or labelled fixtures under `demo/` yet; no evaluation results.
4. Server-derived evidence crops and the frontend use-case tests are not
   written yet.

## 27 October 2026 — demo page, fixtures, evaluation harness

### Changed

- `demo/site/index.html`: deterministic self-owned page with 12 defect
  variants; `demo/cases.json` lists them plus one unchanged control.
- `demo/tools` (own Go module): `cmd/capture` renders fixtures with headless
  Chrome and measures ground-truth boxes from the DOM; `cmd/evaluate` runs every
  case through the API and scores per-category TP/FP/FN, precision, recall,
  mean IoU, control false positives, and latency.
- Backend: `fixture` provider (`FRAMECHECK_PROVIDER=fixture`) and
  `ports.ProviderError` so a provider can store a safe failure code/message.
- Frontend: paired evidence crops for the selected finding (same window on both
  images), stacked when the region is wide.

### Verification

- Capture checks passed: reference byte-identical across two captures; every
  defect differs from the reference; no changed pixel outside the labelled
  component. Two leaks were found and fixed this way (hero title height,
  CTA sibling shift).
- `go test ./...` passes in `framecheck_BE` (including fixture provider) and in
  `demo/tools` (IoU and matching); `pnpm typecheck` and `pnpm test` (6) pass.
- Evaluation against the running API: `fixture` 12/12 matched, IoU 1.00,
  0 control FP; `mock` 2/12 recall, 11 FP, 0 control FP.
- Headless Chrome screenshots of fixture runs show aligned overlays and crops.

### Still open

Official Rules check; Meta adapter and credits; holdout set; a second
unchanged control with a harmless rendering difference.

## 28 October 2026 — Muse Spark adapter (offline-tested only)

### Changed

- `internal/adapters/meta`: Responses API client, strict JSON-schema prompt
  (`muse-v1`), data-URL images with area-averaged downscale, response parsing
  with edge clamping, safe HTTP/status error mapping, and a per-process request
  cap (`MODEL_API_MAX_REQUESTS`, default 20).
- `domain.FitTransform` and `ports.ImageSizeLimiter`: providers can ask for a
  maximum image side; the exact per-axis scale is stored on the run.
- Migration `0002_run_usage.sql`: `input_tokens` and `output_tokens` on runs,
  shown in the UI, Markdown export, and evaluation reports.
- Startup: `FRAMECHECK_PROVIDER=meta` requires `MODEL_API_ENABLED=true` and a
  non-empty key, else the server refuses to start.

### Verification

- Adapter tests against `httptest` (no network): request shape, strict
  schema, two PNG data URLs, requirements cannot close the prompt delimiter,
  token/ID parsing, 1001→1000 clamp, off-schema/incomplete outputs fail, 401/
  402/429/400/503 map to safe codes without echoing the key or body, the 6th
  call is refused under a cap of 5, and downscaled images match the transform.
- Startup guards: refused without `MODEL_API_ENABLED`, refused with an empty key.
- End to end with a local fake `/v1/responses` server and a temporary database
  (dropped afterwards, so no fake `meta_api` rows remain): two runs stored
  `origin meta_api`, tokens 2950/180, request ID, and a validated
  missing-element box at 62,375 200×54 px (ground truth 64,376 196×52); the
  third run failed with `REQUEST_CAP_REACHED` and the fake server saw 2 calls.
- **No request has been sent to Meta.** Rules, credits, and payment remain open.

## 28 October 2026 — holdout set

- Added `demo/site/holdout.html` and `demo/cases-holdout.json`: different
  layout, wording, fonts, colours, and viewport (1440×900), 12 defects, an
  identical control, and a JPEG-q85 control for compression noise.
- `cmd/capture` takes `-cases`/`-out` and a `postprocess` option for controls;
  `cmd/evaluate` takes `-fixtures`; result files now include the split name.
  The fixture provider loads several manifests (`FRAMECHECK_FIXTURES_DIR` is a
  comma-separated list, default dev + holdout).
- Verified: all holdout captures pass determinism and containment checks; the
  dev fixtures re-captured byte-identical with the new tool; `fixture` scores
  12/12 on both sets; `mock` on holdout scores 1/12 with 5 control false
  positives, all on the JPEG control.
- Still no Meta request. Waiting on Official Rules, credits, and explicit
  authorization for a paid call.

## 28 October 2026 — first real Muse Spark request

- Authorized by the user: 1 request, cap set with `MODEL_API_MAX_REQUESTS=1`.
- Input: the user's own dashboard screenshot (2874×1294, viewport 1437×647 @2x)
  as the reference, and a copy with three injected defects as the actual. Personal
  data (name, company, location, check-in time) was covered with identical grey
  boxes in both images before sending. Images kept outside the repository.
- Result: `completed`, `muse-spark-1.3`, 13.0 s, 5,225 input / 1,745 output tokens.
  All three defects found with the right category and no false positives (the
  grey boxes and the ticking clock were ignored):
  - missing "Absen Pulang" button: box IoU 0.92 with the injected region;
  - "Pengumuman Divisi" clipped: full title boxed on the reference, the
    remaining "Pengumu" on the actual;
  - month/week/day switcher shifted 160 px: IoU about 0.96 on both images.
- One real pair is an anecdote, not an evaluation. The dev/holdout fixture
  evaluation with Muse Spark still needs its own authorization.

## 28 October 2026 — demo video recording

- New self-made showcase design `demo/site/kopiku.html` (fictional coffee-subscription
  checkout, 1440×900) with three defects in one build; truth in `demo/showcase/truth.json`.
- Authorized: up to 4 Muse Spark requests for the recording. Used 3: one pre-check,
  one recording discarded because frames were captured at the wrong scale, and the
  final recording. Every run found 3/3 defects with the right category and no false
  positives (e.g. 3,659 input / 2,481 output tokens, 13.8 s).
- `demo/tools/cmd/screenrec` records the walkthrough; `demo/tools/cmd/render` renders
  page variants. The image panel on the comparison page is now sticky on wide
  screens so the images stay in view while reviewing.
- Output: `submission/video/framecheck-demo-full.mp4` (2:12) and `submission/VOICEOVER.md`.
