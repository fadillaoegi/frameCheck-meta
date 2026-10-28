# Product brief and screen flow

## Why FrameCheck exists

Developers often compare UI screenshots against designs manually. A missing
component, clipped label, or shifted layout can be missed before release.
FrameCheck turns a matched design and implementation capture into reviewable
visual findings. It provides evidence and a developer decision trail; it does
not infer code defects or change source files.

## User and scope

The first user is an individual web or mobile developer checking one page at a
time. The proposed hackathon entry is Indie Developer / Build for Your World.
The MVP accepts PNG/JPEG images and short English requirements. Its only finding
categories are `missing_element`, `clipped_text`, and `layout_mismatch`.

## Four screens

| Screen | Main action | Required feedback |
| --- | --- | --- |
| Project & upload | Create project; add reference, actual, requirements, capture metadata | Image dimensions, viewport/scale/content match, validation errors |
| Comparison | Start a run; view synchronized images and annotations | Run provenance, progress, uncertainty, reference and actual evidence |
| Findings & review | Inspect a finding; accept or reject with reason | Category, explanation, paired crops, check step, review state |
| Export | Choose JSON or Markdown for a specific run | Reviewed status, input hashes, model/config, explicit gaps |

`project → comparison draft → ready → queued → running → completed / needs_review / failed → reviewed → exported`

- `ready` requires two valid images with matching viewport CSS size, device
  pixel ratio, and content state. The first slice rejects mismatches.
- A changed image or requirement creates a new comparison/run version. Existing
  findings and reviews remain bound to their original inputs.
- `needs_review` covers missing/invalid evidence or incomplete model output.
  Zero findings is a result only when the run completed with sufficient evidence.
- The dashboard marks `fixture`, `mock`, and `meta_api` origins visibly.

## First demo

Create one original page under `demo/` after the build gate. Capture a reference
and matching implementation at the same viewport and content state. Introduce
one known missing element for the first real vertical slice, plus an unchanged
control. Add clipped text and layout mismatch only after that slice works.

No automatic release verdict, DOM parsing, code repair, or generated design
proposal is in the MVP. Muse Image is optional after detection reliability is
measured. SAM 3.1 is a region candidate, not a UI parser.
