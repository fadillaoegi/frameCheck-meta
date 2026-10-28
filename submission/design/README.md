# Design materials

Posters, thumbnail and the submission brief, in the app's dark neon theme
(`theme.css`: Chakra Petch, IBM Plex Sans/Mono, cyan/magenta neon on a grid).

| File | Size | Use |
| --- | --- | --- |
| `FrameCheck-Submission-Brief.pdf` | A4, 7 pages | Project write-up organised by the Official Rules' judging criteria (Impact, Use of Meta AI Tooling, Technical Depth, Demo) plus originality and roadmap |
| `framecheck-poster.png` | 2480×3508 | Poster (A3 portrait at 212 dpi / A4 at 300 dpi) for social posts and the submission gallery |
| `framecheck-poster-A3.pdf` | A3 | Print version of the poster |
| `framecheck-youtube-thumbnail.png` | 1280×720 | YouTube thumbnail for the demo video |

Screens in `assets/` are frames of the demo recording (a real Muse Spark run on the
self-made "Kopiku" page); `kopiku-*.png` are the design and build images.

Rebuild after editing the HTML:

```bash
cd demo/tools && go build -o /tmp/htmlrender ./cmd/htmlrender
cd ../../submission/design
/tmp/htmlrender -page thumbnail.html -out framecheck-youtube-thumbnail.png -w 1280 -h 720
/tmp/htmlrender -page poster.html -out framecheck-poster.png -w 1240 -h 1754 -scale 2
/tmp/htmlrender -page poster.html -out framecheck-poster-A3.pdf -pdf A3
/tmp/htmlrender -page brief.html -out FrameCheck-Submission-Brief.pdf -pdf A4
```
