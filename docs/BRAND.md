# FrameCheck brand mark

Files: `framecheck_FE/public/brand/`

| File | Use |
| --- | --- |
| `framecheck-mark.svg` | The mark on its own, transparent background (app header, README, slides) |
| `framecheck-app-icon.svg` | The mark on a paper-coloured rounded tile (favicon, app icon, avatars) |
| `framecheck-mark-neon.svg`, `framecheck-app-icon-neon.svg` | Variants for the dark neon app theme: cyan design plate, dark frame with a magenta edge, hot-pink finding |
| `generate_logo.py` | Generates both SVGs from isometric geometry; edit and rerun instead of editing paths by hand |

## What it shows

An isometric view of two plates stacked one above the other.

1. **The blue plate underneath is the design.** It is the source of truth: what the page is meant to look like. It sits at the bottom because everything is checked against it.
2. **The black frame above is the build.** It is a frame, not a solid sheet, because FrameCheck looks *through* the implementation at the design. The name comes from this frame.
3. **The crosshair seen through the window is a registration mark.** Printers use it to check that colour layers line up. Here it marks the one point where the design and the build are compared.
4. **The red dot and dashed post are a finding with its evidence.** The red dot is where something does not line up. The post runs from that point on the design up through the build, the same line the "Layers 3D" view draws for a selected finding. FrameCheck never reports a problem without showing where it is on both images.

In short: **look through what was built, at what was intended, and point at the difference.**

## Colours

| Role | Hex | Meaning |
| --- | --- | --- |
| Design blue | `#2b4acb` | The reference design, also used for "Design" tags in the app |
| Ink | `#1b1915` | The build, and the app's primary text and buttons |
| Vermilion | `#d4412a` | A finding; also the "Missing element" colour |
| Paper | `#f3efe6` | Background, like a proofing table |

## Usage

- Keep the mark at 24px or larger. At 16px the window detail is lost, so use `framecheck-app-icon.svg`, whose tile keeps the silhouette readable.
- Give it clear space equal to the height of the red dot's post on every side.
- Do not recolour the two plates. The blue-under-black order is the meaning: design underneath, build on top.
- The wordmark is "FrameCheck" set in Instrument Serif, regular weight, to the right of the mark.
