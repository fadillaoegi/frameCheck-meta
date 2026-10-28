# Draft data and HTTP contracts (v0, preparation only)

These contracts are proposed for the first slice. They are not implemented.
JSON field names and enum values are English for the UI and demo.

## Main records

| Record | Required fields | Invariant |
| --- | --- | --- |
| Project | `id`, `name`, `created_at` | One user's project in MVP |
| Asset | `id`, `project_id`, `role`, `sha256`, `mime`, `bytes`, `width_px`, `height_px`, `viewport_width_css`, `viewport_height_css`, `device_pixel_ratio`, `content_state_key`, `storage_key` | Immutable bytes/hash; role `reference` or `actual` |
| Requirement version | `id`, `project_id`, `version`, `text`, `sha256`, `created_at` | Edits create a new version |
| Comparison | `id`, `project_id`, `reference_asset_id`, `actual_asset_id`, `requirement_version_id`, `created_at` | Capture metadata must match |
| Run | `id`, `comparison_id`, `status`, `origin`, `input_fingerprint`, `model_id`, `config_version`, `prompt_version`, `transform`, `started_at`, `finished_at`, `provider_request_id` | Immutable input/provenance; origin `meta_api`, `fixture`, or `mock` |
| Finding | `id`, `run_id`, `category`, `summary`, `check_step`, `reference_evidence`, `actual_evidence`, `evidence_status` | Only three MVP categories; evidence bound to run assets |
| Review | `id`, `finding_id`, `decision`, `reason`, `reviewed_at` | Append-only history; latest is current decision |

`Run.status`: `queued`, `running`, `completed`, `needs_review`, `failed`.
`Finding.evidence_status`: `validated` or `needs_review`.
`Review.decision`: `accepted` or `rejected`. A model response cannot create a
review. A completed run with zero findings needs valid response and image
coverage; invalid or absent response becomes `needs_review` or `failed`.

## Evidence object

```json
{
  "asset_id": "asset_reference_uuid",
  "image_role": "reference",
  "box_px": {"x": 120, "y": 88, "width": 156, "height": 48},
  "description": "Primary action is visible here"
}
```

The companion actual evidence can be `null` for a truly missing element, but
the finding must state that absence is the claim. A layout mismatch needs both
boxes. Clipped text needs an actual box plus reference text/region evidence.
Every non-null box must lie inside its asset's original raster boundaries. A
crop URI is derived by the server from the asset and box; provider-supplied
URLs are never trusted as evidence.

## HTTP routes

| Method and path | Request | Success | Key errors |
| --- | --- | --- | --- |
| `POST /v1/projects` | `{ "name": "Demo page" }` | `201` project | `400` invalid name |
| `POST /v1/projects/{id}/assets` | multipart `file`, `role`, viewport CSS width/height, DPR, content state key | `201` asset metadata | `400` invalid image, `413` too large |
| `POST /v1/projects/{id}/comparisons` | `reference_asset_id`, `actual_asset_id`, `requirements` | `201` comparison + requirement version | `409` mismatch or wrong project |
| `POST /v1/comparisons/{id}/runs` | optional `Idempotency-Key` | `202` run | `409` active/duplicate conflict, `503` provider unavailable |
| `GET /v1/runs/{id}` | — | `200` run + findings | `404` missing |
| `POST /v1/findings/{id}/reviews` | `decision`, `reason` | `201` review | `400` invalid decision, `409` stale run |
| `GET /v1/runs/{id}/export?format=json\|markdown` | — | `200` attachment | `400` format, `409` unfinished run |
| `GET /healthz` | — | `200` process status | — |

Errors use `{ "error": { "code": "CAPTURE_MISMATCH", "message": "..." } }`.
Server responses never include filesystem paths or provider keys. Exports
include origin, input hashes, requirement version, model/config, run status,
findings, reviews, and any unresolved evidence caveat.

## Provider boundary

Implemented in `framecheck_BE/internal/adapters/meta` (prompt version
`muse-v1`): one `POST /v1/responses` per run with the prompt, the reference
image, then the actual image as `input_image` data URLs (downscaled when the
longer side exceeds `MODEL_API_MAX_IMAGE_SIDE`, with the transform stored on the
run); `text.format` is a strict `json_schema` (`findings[]` with category,
summary, check_step, nullable reference/actual evidence with an
`x_min..y_max` box on the 0–1000 grid, plus `inspected_both_images`);
`temperature` 0 and `store` false. Response `id`, `usage`, and latency are
stored on the run. Boxes within 10 units outside 0..1000 are clamped, as the
image docs advise; anything further is rejected by the domain. Non-200 HTTP
statuses, `incomplete`/`failed` responses, empty text, and off-schema JSON fail
the run with a safe code; response bodies are never stored.

The adapter will send both images and requirements to Meta Model API only after
account, rules, and spending gates are cleared. Official docs currently show
`POST https://api.meta.ai/v1/responses`, `muse-spark-1.3` for image input, and
`sam-3.1` for media segmentation. Provider JSON is untrusted even if structured
output is requested. Parsing must reject unknown categories, out-of-range or
non-finite boxes, cross-asset IDs, empty explanations, and missing evidence.
Provider request IDs and usage/latency are stored when returned; a provider
failure never becomes a clean pass.
