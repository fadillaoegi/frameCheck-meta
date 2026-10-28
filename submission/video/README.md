# Intro and closing animations

| File | Length | Use |
| --- | --- | --- |
| `framecheck-intro.mp4` | 7.0 s | First shot of the demo video, ends on a fade to black |
| `framecheck-outro.mp4` | 7.0 s | Last shot of the demo video, fades in from black and out to black |
| `framecheck-demo-final.mp4` | 2:32 | **Final demo**: everything below plus voice-over, captions, callouts, chapter cards and music. Built by `src/build_demo.py` |
| `framecheck-demo-final.srt` | | Captions as a subtitle file (for upload platforms; the MP4 has captions burned in only) |
| `framecheck-demo-full.mp4` | 2:12 | Intro + screen recording (with a quiet ambient bed) + outro: the demo video, ready for a voice-over |
| `framecheck-screen-recording.mp4` | 1:58 | The screen recording alone, no audio |
| `build/intro.wav`, `build/outro.wav`, `build/screen-bed.wav` | | The soundtracks on their own, for re-editing |
| `build/framecheck-demo-guide-tts.mp4` | 2:12 | Timing guide with a synthetic voice reading `../VOICEOVER.md`. Not for submission. |
| `build/screen-markers.json` | | Scene times in the screen recording |

Both are 1920×1080, 30 fps, H.264 + AAC 256 kbps, audio peaking at −1 dBFS. Place the intro at the start of the timeline, cut straight to the screen recording after its fade, and put the outro last.

## What happens

**Intro.** A floor grid and a registration crosshair draw in the dark. The design plate (blue, with a small web page printed on it) rises out of the depth and lands with a low thud. The build frame (black) drops onto it from above and seats with a click. Through the frame's window a red registration point pops with a bell, and the dotted evidence post climbs up through the frame. The camera pulls back and the wordmark arrives on a soft A-major chord: **FrameCheck — Does the build match the design?**

**Outro.** The camera sweeps around the assembled mark. Three cards flip up, one per defect category (missing element, clipped text, layout mismatch), followed by the line "Evidence on both images · A person decides every finding · Export to your PR". The cards fold away, the wordmark returns with "Visual QA with evidence." and the hackathon credit. The frame lifts for a beat, the point pings once, and it fades out.

The look matches the app's dark neon theme: cyan design plate, dark build frame with a magenta edge, hot-pink finding, cyan "Check" in Chakra Petch. The motion follows the logo's meaning (see `docs/BRAND.md`): design underneath, build on top, the difference found at one point with evidence running through both.

## Sound

Every sound is synthesised by `src/sound.py` from sine waves and filtered noise, so there is nothing to license or credit:

- a low A drone under everything;
- filtered-noise whooshes for movement;
- a falling-sine thud and a three-partial click when the plates land;
- a bell ping for the registration point, and a gliding tone for the evidence post;
- an A major 9 pad for the wordmark, with a sputtering neon-tube hum as "Check" lights up;
- a short feedback-delay reverb for space.

Cue times in `sound.py` match the `T` timelines in `src/intro.html` and `src/outro.html`.

## Rebuild

The scenes are plain HTML and CSS 3D, written so that every frame is a pure function of time. Rebuilding gives the same frames every time.

```bash
cd demo/tools
go build -o /tmp/videoframes ./cmd/videoframes
/tmp/videoframes -page ../../submission/video/src/intro.html -out /tmp/fr-intro
/tmp/videoframes -page ../../submission/video/src/outro.html -out /tmp/fr-outro
# stills only, for checking:  -at 1.0,3.3,6.2

cd ../../submission/video
python3 src/sound.py build
for n in intro outro; do
  ffmpeg -y -framerate 30 -i /tmp/fr-$n/frame_%05d.png -i build/$n.wav \
    -c:v libx264 -preset slow -crf 16 -pix_fmt yuv420p -movflags +faststart \
    -c:a aac -b:a 256k -shortest framecheck-$n.mp4
done
```

The renderer needs Chrome or Chromium installed. The pages load fonts from Google Fonts, so rendering needs internet access.

## Screen recording

`demo/tools/cmd/screenrec` drives the running app in headless Chrome at a 1.25 device scale (1920×1080 frames): a visible cursor moves on eased, slightly curved paths, text is typed key by key, and the DevTools screencast captures every frame with its timestamp. Upload clicks are shown but the files are set directly, since a native file dialog cannot be recorded. The wait for Muse Spark is played 2.5× faster with an on-screen label. The run in the recording is a real Muse Spark request.

```bash
scripts/clean-demo-data.sh --yes          # fresh project list, no duplicate run
cd demo/tools && go build -o /tmp/screenrec ./cmd/screenrec
cd .. && /tmp/screenrec -out /tmp/rec -design showcase/kopiku-design.png -build showcase/kopiku-build.png
ffmpeg -f concat -safe 0 -i /tmp/rec/frames.txt -vf "fps=30,format=yuv420p" -c:v libx264 -crf 17 screen.mp4
python3 ../submission/video/src/sound.py ../submission/video/build bed <seconds>
```

## Final edit

`src/build_demo.py` assembles `framecheck-demo-final.mp4`:

- cuts the screen recording at the scene markers and slows a scene (at most 1.5×) only as much as its voice-over clip needs;
- inserts five 3D chapter cards (`src/chapter.html`, rendered with `cmd/videoframes -query`);
- burns in captions and callouts rendered as transparent PNGs by `demo/tools/cmd/overlays` (this ffmpeg has no text filters), and writes the same captions to an `.srt` (not muxed into the MP4, so players do not show them twice);
- corrects the on-screen speed-up label for the model wait after retiming;
- mixes the intro/outro sound, a generated synthwave bed (`sound.py ... music`) ducked under the voice with a sidechain compressor, and the 17 voice clips, normalised to −15 LUFS.

```bash
cd demo/tools && go build -o /tmp/videoframes ./cmd/videoframes && go build -o /tmp/overlays ./cmd/overlays
cd ../../submission/video && VIDEOFRAMES=/tmp/videoframes OVERLAYS=/tmp/overlays python3 src/build_demo.py
```

The caption text comes from `SCENES` in `build_demo.py` (the script in `../VOICEOVER.md`). If a recorded clip says something different, edit the text there and rebuild.
