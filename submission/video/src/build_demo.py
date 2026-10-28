"""Assemble the final FrameCheck demo video.

Timeline: 3D intro -> screen recording cut into scenes (each scene stretched just
enough to fit its voice-over clip) with 3D chapter cards between the steps -> 3D
outro. On top: burned-in captions from the voice-over script and informative
callouts. Audio: intro/outro sound, a synthwave bed that ducks under the voice,
and the voice-over clips. Also writes the captions to an .srt (not muxed, so players don't show them twice).

    VIDEOFRAMES=/path/videoframes OVERLAYS=/path/overlays python3 src/build_demo.py

Inputs (relative to submission/video): framecheck-intro.mp4, framecheck-outro.mp4,
framecheck-screen-recording.mp4, build/screen-markers.json, ../voiceover/*.mp3.
"""

import html
import json
import os
import subprocess
import sys
from pathlib import Path
from urllib.parse import urlencode

HERE = Path(__file__).resolve().parent
VID = HERE.parent
SUB = VID.parent
OUT = VID / "build" / "final"
FPS = 30

# One voice-over line per screen scene, in order (text = submission/VOICEOVER.md).
SCENES = [
    ("home", "how-it-works", "Every front-end developer knows this question: does the build really match the design?"),
    ("how-it-works", "categories", "FrameCheck answers it in five steps."),
    ("categories", "new-project", "Three kinds of defect: missing, clipped, misplaced."),
    ("new-project", "upload", "Let's try it on a checkout page I designed for a coffee subscription."),
    ("upload", "capture", "I drop in the design export, and a screenshot of the build."),
    ("capture", "pair", "Both must be captured the same way. FrameCheck reads the image size and suggests the pixel ratio, so the two line up pixel for pixel."),
    ("pair", "analyse", "I pair them, and add a note about what matters on this page. Notes give the model context, but they can't change its rules or its output format."),
    ("analyse", "wait-start", "Now, one request to Meta's Muse Spark, with both images."),
    ("wait-start", "results", "It answers in strict, structured JSON: a category, a summary, and a box on each image."),
    ("results", "finding-1", "Three findings, with the model and token cost."),
    ("finding-1", "finding-2", "The promo code row is missing. Both images zoom to the same spot, with a dashed box where it should be. Accept."),
    ("finding-2", "finding-3", "The pay button is narrower than designed, boxed on both images, so the difference is easy to check."),
    ("finding-3", "swipe", "And the first product title is cut off. Nothing counts until a person decides, so each one gets a human verdict."),
    ("swipe", "layers", "Swipe across the two images to see each change at the same position: the promo row vanishes, the button shrinks, the title is cut."),
    ("layers", "export", "Or pull them apart in 3D. The build floats above the design. Flatten it into an overlay, or turn the stack and follow a finding from the design straight up to the build. Every box is anchored to the original pixels, so it lines up at any zoom."),
    ("export", "end", "Finally, export the reviewed findings as Markdown for the pull request, with the model, the tokens, and every human decision."),
]
OUTRO_LINE = "FrameCheck. Visual QA, with evidence."
OUTRO_VO_AT = 3.4  # seconds into the outro, as the wordmark lands

# Chapter cards, inserted before the scene with this index.
CHAPTERS = {
    3: ("01", "Step 1 of 5", "Upload & pair", "A design export and a build screenshot, captured the same way", "#00f0ff"),
    7: ("02", "Step 2 of 5", "Analyse with Muse Spark", "One request to Meta's model, both images, structured output", "#ff2bd6"),
    10: ("03", "Step 3 of 5", "Review every finding", "Zoom both images, then accept or reject with a reason", "#3dff9e"),
    13: ("04", "Step 4 of 5", "Compare in Swipe & 3D", "See every change at the same position", "#b46bff"),
    15: ("05", "Step 5 of 5", "Export the report", "Markdown for the pull request, JSON for your tools", "#ffb627"),
}
CARD_LEN = 2.2

# Informative callouts: (scene index, seconds after the scene starts, duration, kicker, text, colour).
CALLOUTS = [
    (0, 0.8, 4.2, "Meta Global AI Developer Hackathon", "Indie track · Build for Your World", "#00f0ff"),
    (5, 4.0, 5.5, "Capture check", "1440 px image ÷ 1440 px viewport = pixel ratio 1", "#00f0ff"),
    (6, 4.5, 6.0, "Prompt safety", "Notes go in a delimited block; they cannot change the output schema", "#00f0ff"),
    (7, 0.6, 3.8, "One request", "muse-spark-1.3 · 2 images · strict json_schema", "#ff2bd6"),
    (9, 0.3, 3.4, "Live result", "3 of 3 injected defects found · 0 false positives", "#3dff9e"),
    (10, 1.0, 6.5, "missing_element", "Boxed on the design; a dashed ghost marks the spot on the build", "#ff2e63"),
    (11, 0.8, 5.8, "layout_mismatch", "Same element, different size: boxed on both images", "#b46bff"),
    (12, 0.8, 5.0, "clipped_text", "Full title on the design, cut text on the build", "#ffb627"),
    (14, 8.0, 7.0, "Coordinates", "Model answers on a 0–1000 grid, mapped back to original pixels", "#00f0ff"),
    (15, 0.8, 6.0, "Report", "Origin, model, tokens, input fingerprint and every human decision", "#00f0ff"),
]

# The recording shows the model wait at this speed-up; slowing the scene changes it.
RECORDED_WAIT_SPEED = 2.5
BADGE_HTML = """<!doctype html><html><head><meta charset="utf-8">
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@600&display=block" rel="stylesheet">
<style>html,body{{margin:0;width:1920px;height:1080px;background:transparent}}
.b{{position:absolute;left:50%;bottom:30px;transform:translateX(-50%);padding:15px 28px;border-radius:999px;white-space:nowrap;
background:#00f0ff;color:#07070e;font:600 19px/1 "IBM Plex Mono",monospace;box-shadow:0 0 30px rgba(0,240,255,.6)}}</style></head>
<body><div class="b">{text}</div></body></html>"""

LEAD, TAIL = 0.35, 0.4  # silence before / after each voice line inside its scene
MAX_SLOW = 1.5          # slow a scene at most this much; hold the last frame beyond


def run(*args, **kw):
    print("+", " ".join(str(a) for a in args)[:200], flush=True)
    subprocess.run([str(a) for a in args], check=True, **kw)


def duration(path):
    return float(subprocess.check_output(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", str(path)]))


def ts(t):
    h, rem = divmod(t, 3600)
    m, s = divmod(rem, 60)
    return f"{int(h):02d}:{int(m):02d}:{int(s):02d},{int(round((s - int(s)) * 1000)):03d}"


def chunks(text, dur):
    """Split a voice line into caption chunks of at most ~84 characters, timed by length."""
    parts, cur = [], ""
    for word in text.split():
        if cur and (len(cur) + 1 + len(word) > 84 or (cur.endswith((".", "?", ":")) and len(cur) > 30)):
            parts.append(cur)
            cur = word
        else:
            cur = f"{cur} {word}".strip()
    parts.append(cur)
    total = sum(len(p) for p in parts)
    out, t = [], 0.0
    for p in parts:
        d = dur * len(p) / total
        out.append((t, t + d, p))
        t += d
    return out


CAPTION_HTML = """<!doctype html><html><head><meta charset="utf-8">
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@500;600&display=block" rel="stylesheet">
<style>html,body{{margin:0;width:1920px;height:1080px;background:transparent}}
.c{{position:absolute;left:50%;bottom:{bottom}px;transform:translateX(-50%);max-width:1500px;padding:14px 26px;border-radius:14px;
background:rgba(7,7,14,.82);box-shadow:0 0 0 1px rgba(0,240,255,.35);color:#fff;text-align:center;
font:600 40px/1.3 "IBM Plex Sans",sans-serif;letter-spacing:.2px}}</style></head>
<body><div class="c">{text}</div></body></html>"""

CALLOUT_HTML = """<!doctype html><html><head><meta charset="utf-8">
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@500;600&family=IBM+Plex+Sans:wght@500;600&display=block" rel="stylesheet">
<style>html,body{{margin:0;width:1920px;height:1080px;background:transparent}}
.b{{position:absolute;right:44px;top:112px;max-width:620px;padding:16px 22px 18px;border-radius:16px;background:rgba(13,13,26,.92);
box-shadow:inset 0 0 0 2px {color},0 0 34px -6px {color};border-left:8px solid {color}}}
.k{{font:600 18px/1 "IBM Plex Mono",monospace;letter-spacing:3px;text-transform:uppercase;color:{color}}}
.t{{margin-top:10px;font:600 28px/1.3 "IBM Plex Sans",sans-serif;color:#ecebff}}</style></head>
<body><div class="b"><div class="k">ⓘ {kicker}</div><div class="t">{text}</div></div></body></html>"""


def main():
    vf, ov = os.environ.get("VIDEOFRAMES"), os.environ.get("OVERLAYS")
    if not vf or not ov:
        sys.exit("set VIDEOFRAMES and OVERLAYS to the built demo/tools binaries")
    OUT.mkdir(parents=True, exist_ok=True)
    screen = VID / "framecheck-screen-recording.mp4"
    markers = {m["name"]: m["at"] for m in json.loads((VID / "build" / "screen-markers.json").read_text())["markers"]}
    markers["end"] = duration(screen)
    vo = [SUB / "voiceover" / f"framecheck-voiceover-{i}.mp3" for i in range(1, 18)]
    for p in vo:
        if not p.exists():
            sys.exit(f"missing voice-over clip {p}")
    vo_len = [duration(p) for p in vo]
    intro_len, outro_len = duration(VID / "framecheck-intro.mp4"), duration(VID / "framecheck-outro.mp4")

    # ---- timeline -------------------------------------------------------------
    pieces = [("intro", None)]
    t = intro_len
    scene_start, vo_at, srt, overlays, plan = {}, {}, [], [], []
    for i, (a, b, text) in enumerate(SCENES):
        if i in CHAPTERS:
            pieces.append(("card", i))
            t += CARD_LEN
        src_len = markers[b] - markers[a]
        need = LEAD + vo_len[i] + TAIL
        slow = min(MAX_SLOW, max(1.0, need / src_len))
        hold = max(0.0, need - src_len * slow)
        out_len = src_len * slow + hold
        pieces.append(("scene", (markers[a], markers[b], slow, hold)))
        scene_start[i] = t
        vo_at[i] = t + LEAD
        plan.append(f"scene {i + 1:2d} {a:>12}: src {src_len:5.2f}s x{slow:.2f} +hold {hold:4.2f}s -> {out_len:5.2f}s  voice {vo_len[i]:5.2f}s")
        t += out_len
    outro_at = t
    pieces.append(("outro", None))
    total = t + outro_len
    vo_at[16] = outro_at + OUTRO_VO_AT
    print("\n".join(plan))
    print(f"total {total:.1f}s")

    # ---- chapter cards -------------------------------------------------------
    cards = {}
    for idx, (n, kicker, title, sub, color) in CHAPTERS.items():
        frames = OUT / f"card{n}"
        mp4 = OUT / f"card{n}.mp4"
        if not mp4.exists():
            q = urlencode({"n": n, "kicker": kicker, "title": title, "sub": sub, "color": color})
            run(vf, "-page", HERE / "chapter.html", "-out", frames, "-query", q)
            run("ffmpeg", "-loglevel", "error", "-y", "-framerate", FPS, "-i", frames / "frame_%05d.png",
                "-c:v", "libx264", "-crf", "16", "-pix_fmt", "yuv420p", mp4)
        cards[idx] = mp4

    # ---- captions + callouts as transparent PNGs -------------------------------
    # The closing line is spoken over the wordmark and left uncaptioned.
    lines = [(vo_at[i], vo_len[i], SCENES[i][2]) for i in range(16)]
    spec, k = [], 0
    for li, (start, dur, text) in enumerate(lines):
        # The model-wait scene has its own badge at the bottom; lift the caption above it.
        bottom = 120 if SCENES[li][0] == "wait-start" else 58
        for a, b, part in chunks(text, dur):
            png = OUT / f"cap{k:03d}.png"
            spec.append({"out": str(png), "html": CAPTION_HTML.format(text=html.escape(part), bottom=bottom)})
            overlays.append((png, start + a, start + b + (0.15 if b >= dur - 1e-6 else 0.0)))
            srt.append((start + a, start + b, part))
            k += 1
    for j, (scene, off, dur, kicker, text, color) in enumerate(CALLOUTS):
        png = OUT / f"call{j:02d}.png"
        spec.append({"out": str(png), "html": CALLOUT_HTML.format(kicker=html.escape(kicker), text=html.escape(text), color=color)})
        overlays.append((png, scene_start[scene] + off, scene_start[scene] + off + dur))
    # Correct the on-screen speed-up label for the wait scene's actual playback speed.
    wait = next(i for i, sc in enumerate(SCENES) if sc[0] == "wait-start")
    wait_piece = [p for p in pieces if p[0] == "scene"][wait][1]
    speed = RECORDED_WAIT_SPEED / wait_piece[2]
    png = OUT / "badge.png"
    spec.append({"out": str(png), "html": BADGE_HTML.format(text=f"⏩ Waiting for Muse Spark — sped up {speed:.1f}×")})
    overlays.insert(0, (png, scene_start[wait], scene_start[wait] + (wait_piece[1] - wait_piece[0]) * wait_piece[2] + wait_piece[3]))
    (OUT / "overlays.json").write_text(json.dumps(spec))
    run(ov, "-spec", OUT / "overlays.json")

    srt_path = VID / "framecheck-demo-final.srt"
    srt_path.write_text("".join(f"{n}\n{ts(a)} --> {ts(b)}\n{txt}\n\n" for n, (a, b, txt) in enumerate(srt, 1)))

    # ---- base video: intro + scenes + cards + outro ----------------------------
    norm = f"fps={FPS},scale=1920:1080,setsar=1,format=yuv420p"
    inputs = ["-i", VID / "framecheck-intro.mp4", "-i", screen]
    card_input = {}
    for idx in sorted(cards):
        card_input[idx] = len(inputs) // 2
        inputs += ["-i", cards[idx]]
    outro_input = len(inputs) // 2
    inputs += ["-i", VID / "framecheck-outro.mp4"]
    scenes = [p for p in pieces if p[0] == "scene"]
    fc = [f"[1:v]split={len(scenes)}" + "".join(f"[s{i}]" for i in range(len(scenes)))]
    labels, si = [], 0
    for kind, val in pieces:
        if kind == "intro":
            fc.append(f"[0:v]{norm}[vin]")
            labels.append("[vin]")
        elif kind == "outro":
            fc.append(f"[{outro_input}:v]{norm}[vout]")
            labels.append("[vout]")
        elif kind == "card":
            fc.append(f"[{card_input[val]}:v]{norm}[c{val}]")
            labels.append(f"[c{val}]")
        else:
            a, b, slow, hold = val
            chain = f"[s{si}]trim=start={a:.3f}:end={b:.3f},setpts=(PTS-STARTPTS)*{slow:.4f},{norm}"
            if hold > 0:
                chain += f",tpad=stop_mode=clone:stop_duration={hold:.3f}"
            fc.append(chain + f"[p{si}]")
            labels.append(f"[p{si}]")
            si += 1
    fc.append("".join(labels) + f"concat=n={len(labels)}:v=1:a=0[base]")
    base = OUT / "base.mp4"
    run("ffmpeg", "-loglevel", "error", "-y", *inputs, "-filter_complex", ";".join(fc), "-map", "[base]",
        "-c:v", "libx264", "-preset", "medium", "-crf", "16", "-r", FPS, base)

    # ---- overlays on top -----------------------------------------------------
    inputs, fc, prev = ["-i", base], [], "[0:v]"
    for n, (png, a, b) in enumerate(overlays, 1):
        d = b - a
        inputs += ["-loop", "1", "-t", f"{d:.3f}", "-i", png]
        fc.append(f"[{n}:v]format=rgba,fade=t=in:st=0:d=0.18:alpha=1,fade=t=out:st={max(0, d - 0.18):.3f}:d=0.18:alpha=1,"
                  f"setpts=PTS-STARTPTS+{a:.3f}/TB[o{n}]")
        fc.append(f"{prev}[o{n}]overlay=eof_action=pass:format=auto[v{n}]")
        prev = f"[v{n}]"
    fc.append(f"{prev}format=yuv420p[vout]")
    video = OUT / "video.mp4"
    run("ffmpeg", "-loglevel", "error", "-y", *inputs, "-filter_complex", ";".join(fc), "-map", "[vout]",
        "-c:v", "libx264", "-preset", "slow", "-crf", "17", "-r", FPS, "-t", f"{total:.3f}", video)

    # ---- audio -----------------------------------------------------------------
    music_len = outro_at - intro_len + 1.2
    run(sys.executable, HERE / "sound.py", VID / "build", "music", f"{music_len:.2f}")
    a_in = ["-i", VID / "build" / "intro.wav", "-i", VID / "build" / "outro.wav", "-i", VID / "build" / "music.wav"]
    for p in vo:
        a_in += ["-i", p]
    fc = [
        "[0:a]aresample=48000,apad[ain]",
        f"[1:a]aresample=48000,adelay={int(outro_at * 1000)}|{int(outro_at * 1000)},apad[aout]",
        f"[2:a]aresample=48000,volume=0.55,adelay={int((intro_len - 0.6) * 1000)}|{int((intro_len - 0.6) * 1000)},apad[mus]",
    ]
    vlabels = []
    for i in range(17):
        ms = int(vo_at[i] * 1000)
        fc.append(f"[{3 + i}:a]aresample=48000,aformat=channel_layouts=stereo,adelay={ms}|{ms}[vo{i}]")
        vlabels.append(f"[vo{i}]")
    fc.append("".join(vlabels) + f"amix=inputs=17:normalize=0:duration=longest,apad,atrim=0:{total:.3f},"
              "loudnorm=I=-16:TP=-2:LRA=7,asplit=2[voice][key]")
    fc.append("[mus][key]sidechaincompress=threshold=0.02:ratio=8:attack=15:release=450:makeup=1[ducked]")
    fc.append(f"[ain][aout][ducked][voice]amix=inputs=4:normalize=0:duration=longest,atrim=0:{total:.3f},"
              "loudnorm=I=-15:TP=-1.5:LRA=9[mix]")
    audio = OUT / "audio.m4a"
    run("ffmpeg", "-loglevel", "error", "-y", *a_in, "-filter_complex", ";".join(fc), "-map", "[mix]",
        "-c:a", "aac", "-b:a", "256k", "-ar", "48000", audio)

    final = VID / "framecheck-demo-final.mp4"
    # Captions are burned in; no soft subtitle track, or players would show them twice.
    run("ffmpeg", "-loglevel", "error", "-y", "-i", video, "-i", audio,
        "-map", "0:v", "-map", "1:a", "-c:v", "copy", "-c:a", "copy", "-shortest", "-movflags", "+faststart", final)
    (OUT / "timeline.txt").write_text("\n".join(plan) + f"\noutro at {outro_at:.2f}s, total {total:.2f}s\n")
    print(f"wrote {final} ({total:.1f}s) and {srt_path}")


if __name__ == "__main__":
    main()
