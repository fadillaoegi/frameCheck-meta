# Evaluation plan

## Fixture design

Use one self-owned demo page with deterministic content and captures at matched
viewport, device pixel ratio, browser zoom, and content state. Keep fixture and
mock assets under `demo/fixtures/` with manifests identifying `origin: fixture`.
Real provider runs must be labeled `origin: meta_api` and stored separately.

Development set proposal: 12 deliberately defective pairs, four per category,
plus at least one unchanged control. Build a separate holdout with changed
wording, positions, and visual styling, including unchanged controls, before
prompt tuning. Record expected category, component name, reference/actual boxes,
and whether an issue is visually decidable. Ambiguous examples cannot be counted
as automatic model success.

## Measurements

- Match a finding to a ground-truth issue only when category and component agree
  and the validated box overlaps the labeled region above a declared threshold
  (initial candidate IoU ≥ 0.5). For missing elements, match the reference box.
- Report per-category true positives, false positives, false negatives,
  precision, recall, and location overlap. Report false positives separately
  on unchanged controls.
- Time a developer's review for comparable manual and FrameCheck tasks, recording
  sample size and task order. Do not claim time savings before measurement.
- Record provider model/version, request count, returned usage if available,
  latency, estimated cost if pricing is known, and failure/needs-review rate.
- Probe SAM with a concrete UI component prompt and inspect region precision
  before adding it to the main path. A useful mask is not proof of UI semantics.

## First proof gate

After Official Rules and account access permit it, run one known missing-element
pair, one unchanged pair, and one SAM region probe. Save request metadata and
responses without secrets. Continue to a PostgreSQL-backed review slice only
if findings have valid evidence and coordinates; otherwise document the failure
and adjust the plan.

## Status (27 October 2026)

The development set exists: `demo/cases.json`, generated fixtures in
`demo/fixtures/`, and the scorer `demo/tools/cmd/evaluate` (see
`demo/README.md`). It has 12 defects (4 per category) and **one** unchanged
control. No holdout set exists yet.

Pipeline checks only, **no model result**:

| Origin | Recall | Precision | Control FP | Note |
| --- | --- | --- | --- | --- |
| `fixture` | 12/12 | 12/12 | 0 | Replays ground truth; proves scorer and pipeline agree |
| `mock` | 2/12 | 2/13 | 0 | Pixel-difference baseline; only ever reports `layout_mismatch` |

Holdout set (28 Oct): `demo/cases-holdout.json`, a different pricing page at
1440×900 with 12 defects and **two** controls, one of them the identical render
saved as JPEG q85.

| Origin | Recall | Precision | Control FP | Note |
| --- | --- | --- | --- | --- |
| `fixture` | 12/12 | 12/12 | 0 | Pipeline check, including JPEG upload |
| `mock` | 1/12 | 1/18 | 5 | All 5 control FPs come from JPEG compression noise |

Reports are in `demo/results/`. No Muse Spark run has been made on either set.
