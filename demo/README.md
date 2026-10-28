# FrameCheck demo and fixtures

Everything here is synthetic and self-owned. Nothing in this folder is model output.

| Path | What it is |
| --- | --- |
| `site/index.html` | Deterministic demo page ("Lembar", a fictional notes product). `?variant=<id>` applies one defect. No external fonts, images, or scripts. |
| `cases.json` | Dev case list: 1 unchanged control + 12 defects (4 × missing element, clipped text, layout mismatch), split `dev`. |
| `fixtures/` | Dev PNGs (1280×800 CSS px @1x, state `default`) and `manifest.json` with ground-truth boxes, `origin: fixture`. |
| `site/holdout.html`, `cases-holdout.json` | **Holdout** page (pricing layout, serif type, dark header, 1440×900, state `plans`): 2 controls (identical, and JPEG q85 compression noise) + 12 defects. |
| `fixtures-holdout/` | Holdout captures and manifest. |
| `results/` | Evaluation reports written by `tools/cmd/evaluate`, one JSON + Markdown pair per run, named by time and origin. |
| `tools/` | Separate Go module: `cmd/capture` and `cmd/evaluate`. |

## Regenerate fixtures

```sh
cd demo/tools && go run ./cmd/capture
cd demo/tools && go run ./cmd/capture -cases cases-holdout.json -out fixtures-holdout
```

Ground-truth boxes are measured from the DOM (`getBoundingClientRect`), not
drawn by hand. Capture fails if a defect variant renders identically to the
reference, if the reference is not byte-identical across two captures, or if
any pixel changes **outside** the labelled component (so one case cannot leak
into another category). Do not edit `manifest.json` by hand.

## Evaluate

Start the API with the provider to measure, then:

```sh
cd demo/tools && go run ./cmd/evaluate                                         # dev
cd demo/tools && go run ./cmd/evaluate -split holdout -fixtures fixtures-holdout
```

A finding matches a labelled defect when the category agrees and its box has
IoU ≥ 0.5 with the labelled region (reference box for a missing element,
actual box otherwise). Unmatched findings are false positives; any finding on
the control is a control false positive. Failed runs count every labelled
defect as missed.

## Limits

- **Holdout discipline:** tune prompts only on `dev`. Run a model on the
  holdout only with a frozen prompt version, record that version, and report
  the holdout separately. If the prompt changes after a holdout run, the old
  holdout number no longer describes the new prompt.
- Both sets are synthetic pages rendered by one browser on one machine; they
  say nothing about real product screenshots.
- `fixture` runs replay ground truth: their perfect score only shows the
  pipeline and scorer agree.
