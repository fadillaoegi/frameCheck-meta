"""Synthesise the FrameCheck intro and outro soundtracks (stdlib only).

Every sound is generated here, so there are no licensing questions.
Cue times match the T timelines in intro.html and outro.html.

    python3 sound.py OUTDIR              # writes intro.wav and outro.wav
    python3 sound.py OUTDIR bed SECONDS    # writes screen-bed.wav, a quiet bed for the recording
    python3 sound.py OUTDIR music SECONDS  # writes music.wav, the synthwave bed for the final demo
"""

import math
import random
import struct
import sys
import wave

SR = 48000
TAU = 2 * math.pi


class Track:
    def __init__(self, seconds):
        self.n = int(seconds * SR)
        self.l = [0.0] * self.n
        self.r = [0.0] * self.n

    def add(self, t0, samples, pan=0.0, gain=1.0):
        """Mix mono samples at t0 seconds; pan -1 (left) .. 1 (right)."""
        i0 = int(t0 * SR)
        gl = gain * math.cos((pan + 1) * math.pi / 4)
        gr = gain * math.sin((pan + 1) * math.pi / 4)
        for k, s in enumerate(samples):
            i = i0 + k
            if 0 <= i < self.n:
                self.l[i] += s * gl
                self.r[i] += s * gr

    def reverb(self, mix=0.25):
        """A few feedback delays: enough air to feel like a room, cheap to run."""
        for ch, taps in ((self.l, (0.113, 0.171, 0.237)), (self.r, (0.127, 0.193, 0.251))):
            wet = [0.0] * self.n
            for d in taps:
                k = int(d * SR)
                buf = [0.0] * self.n
                for i in range(k, self.n):
                    buf[i] = ch[i - k] + 0.55 * buf[i - k]
                for i in range(self.n):
                    wet[i] += buf[i]
            for i in range(self.n):
                ch[i] += mix * wet[i] / len(taps)

    def fade(self, t0, t1):
        i0, i1 = int(t0 * SR), min(self.n, int(t1 * SR))
        for i in range(i0, self.n):
            g = 0.0 if i >= i1 else 0.5 + 0.5 * math.cos(math.pi * (i - i0) / (i1 - i0))
            self.l[i] *= g
            self.r[i] *= g

    def write(self, path, peak_db=-1.0):
        peak = max(1e-9, max(max(abs(x) for x in self.l), max(abs(x) for x in self.r)))
        g = (10 ** (peak_db / 20)) / peak
        with wave.open(path, "wb") as w:
            w.setnchannels(2)
            w.setsampwidth(2)
            w.setframerate(SR)
            frames = bytearray()
            for a, b in zip(self.l, self.r):
                frames += struct.pack("<hh", int(max(-1, min(1, a * g)) * 32767), int(max(-1, min(1, b * g)) * 32767))
            w.writeframes(bytes(frames))


rng = random.Random(11)


def env_adsr(n, a, r):
    """Attack then release, in seconds; n samples total."""
    na, nr = max(1, int(a * SR)), max(1, int(r * SR))
    out = []
    for i in range(n):
        g = min(1.0, i / na)
        if i > n - nr:
            g *= max(0.0, (n - i) / nr)
        out.append(g)
    return out


def pad(freqs, dur, attack, release, amp=0.12, shimmer=0.0):
    """Soft chord: detuned sine pairs with slow tremolo."""
    n = int(dur * SR)
    e = env_adsr(n, attack, release)
    out = [0.0] * n
    for j, f in enumerate(freqs):
        for det in (-0.0018, 0.0018):
            ff = f * (1 + det)
            ph = rng.random() * TAU
            for i in range(n):
                t = i / SR
                trem = 1 + 0.12 * math.sin(TAU * (0.21 + 0.07 * j) * t + j)
                out[i] += math.sin(TAU * ff * t + ph) * trem
    if shimmer:
        for i in range(n):
            t = i / SR
            out[i] += shimmer * math.sin(TAU * freqs[-1] * 2 * t) * (0.5 + 0.5 * math.sin(TAU * 3 * t))
    k = amp / (2 * len(freqs))
    return [out[i] * e[i] * k for i in range(n)]


def whoosh(dur, amp=0.5, up=True):
    """Filtered noise whose brightness and level swell, like air moving past."""
    n = int(dur * SR)
    out, y = [], 0.0
    for i in range(n):
        x = i / n
        shape = math.sin(math.pi * (x if up else 1 - x) ** 0.8)
        cutoff = 200 + 5000 * shape ** 2
        a = 1 - math.exp(-TAU * cutoff / SR)
        y += a * (rng.uniform(-1, 1) - y)
        out.append(y * shape * amp)
    return out


def thud(amp=0.9, f0=150, f1=38, decay=0.55):
    """Low impact: a falling sine plus a short noise click."""
    n = int(1.2 * SR)
    out, ph = [], 0.0
    for i in range(n):
        t = i / SR
        f = f1 + (f0 - f1) * math.exp(-t * 18)
        ph += TAU * f / SR
        s = math.sin(ph) * math.exp(-t / decay)
        if i < 0.012 * SR:
            s += rng.uniform(-1, 1) * 0.6 * (1 - i / (0.012 * SR))
        out.append(s * amp)
    return out


def click(amp=0.5):
    """Precise mechanical click: frame seating into place."""
    n = int(0.25 * SR)
    out = []
    for i in range(n):
        t = i / SR
        s = sum(math.sin(TAU * f * t) * math.exp(-t * d) for f, d in ((2150, 60), (3320, 80), (5230, 110)))
        if i < 0.004 * SR:
            s += rng.uniform(-1, 1)
        out.append(s * amp / 3)
    return out


def ping(f=1318.5, amp=0.45, decay=0.9):
    """Bright bell for the registration point."""
    n = int(2.0 * SR)
    out = []
    for i in range(n):
        t = i / SR
        s = math.sin(TAU * f * t) + 0.35 * math.sin(TAU * f * 2.01 * t) * math.exp(-t * 4) + 0.2 * math.sin(TAU * f * 0.5 * t)
        out.append(s * math.exp(-t / decay) * min(1, t * 400) * amp)
    return out


def rise(dur, f0=330, f1=990, amp=0.18):
    """Gliding tone for the evidence post climbing upward."""
    n = int(dur * SR)
    out, ph = [], 0.0
    for i in range(n):
        x = i / n
        f = f0 * (f1 / f0) ** (x * x)
        ph += TAU * f / SR
        g = math.sin(math.pi * x) ** 0.6
        out.append((math.sin(ph) + 0.3 * math.sin(2 * ph)) * g * amp)
    return out


def buzz(dur, amp=0.12, f=120.0):
    """Neon-tube hum: a filtered saw at mains-harmonic pitch that sputters, then holds."""
    n = int(dur * SR)
    out, y, ph = [], 0.0, 0.0
    for i in range(n):
        t = i / SR
        ph = (ph + f / SR) % 1.0
        y += 0.08 * ((2 * ph - 1) - y)
        flick = 1.0 if t > 0.35 or rng.random() > 0.35 * (1 - t / 0.35) else 0.2
        out.append(y * flick * min(1, t * 30) * min(1, (dur - t) * 3) * amp)
    return out


# Notes: A major 9 colour over an A drone.
A1, E2, A2, E3, A3, Cs4, E4, Gs4, B4 = 55.0, 82.41, 110.0, 164.81, 220.0, 277.18, 329.63, 415.30, 493.88


def intro():
    t = Track(7.0)
    t.add(0.0, pad([A1, E2, A2], 7.0, 1.6, 0.6, amp=0.5))           # low drone
    t.add(0.55, whoosh(1.5, 0.35), pan=-0.3)                         # design rises
    t.add(2.0, thud(0.9))                                            # design lands
    t.add(1.95, whoosh(1.05, 0.4, up=False), pan=0.3)                # build drops
    t.add(3.0, click(0.7))
    t.add(3.0, thud(0.45, f0=220, f1=70, decay=0.25))                # build seats
    t.add(3.2, ping(1318.5, 0.5))                                    # point found
    t.add(3.4, rise(0.9))                                            # evidence post
    t.add(4.55, pad([A2, E3, A3, Cs4, Gs4, B4], 2.45, 0.35, 0.4, amp=0.55, shimmer=0.05))  # wordmark chord
    t.add(4.6, thud(0.6, f0=90, f1=36, decay=0.9))
    t.add(4.6, buzz(2.0, 0.5))                                       # "Check" lights up
    t.add(5.2, ping(1760.0, 0.18, decay=0.6), pan=0.4)               # tagline sparkle
    t.reverb(0.3)
    t.fade(6.45, 7.0)
    return t


def outro():
    t = Track(7.0)
    t.add(0.0, pad([A1, E2, A2], 7.0, 0.8, 0.6, amp=0.5))
    for i, (c, pan) in enumerate(zip((0.5, 0.8, 1.1), (-0.5, 0.0, 0.5))):
        t.add(c - 0.1, whoosh(0.55, 0.3), pan=pan)                    # cards flip in
        t.add(c + 0.35, click(0.25 + 0.05 * i), pan=pan)
    t.add(1.9, ping(2637.0, 0.12, decay=0.3))                        # caption tick
    t.add(2.95, whoosh(0.7, 0.35, up=False))                         # cards fold away
    t.add(3.55, pad([A2, E3, A3, Cs4, E4, Gs4, B4], 3.45, 0.3, 0.5, amp=0.6, shimmer=0.05))
    t.add(3.6, thud(0.75, f0=100, f1=34, decay=1.0))
    t.add(3.6, buzz(2.2, 0.5))
    t.add(4.2, ping(1760.0, 0.16, decay=0.5), pan=-0.3)
    t.add(5.3, rise(0.5, 440, 660, 0.1))                             # frame lifts
    t.add(5.6, ping(1318.5, 0.4))                                    # dot pings
    t.add(5.85, click(0.35))                                         # frame seats again
    t.reverb(0.32)
    t.fade(6.3, 7.0)
    return t


def synthwave_loop(bpm=96.0):
    """Eight bars of soft synthwave in A minor (Am F C G): kick, off-beat hats,
    a filtered-saw bass arpeggio, a slow pad and a quiet sine arp. Returns
    (left, right) sample lists for one loop."""
    beat = 60.0 / bpm
    bar = 4 * beat
    n = int(8 * bar * SR)
    L, R = [0.0] * n, [0.0] * n
    chords = [(220.0, 261.63, 329.63), (174.61, 220.0, 261.63), (130.81, 164.81, 196.0), (196.0, 246.94, 293.66)]

    def put(t0, samples, pan=0.0, gain=1.0):
        i0 = int(t0 * SR)
        gl, gr = gain * (1 - max(0, pan)), gain * (1 + min(0, pan))
        for k, v in enumerate(samples):
            i = (i0 + k) % n  # wrap so the loop tiles seamlessly
            L[i] += v * gl
            R[i] += v * gr

    kick = [math.sin(TAU * (48 + 90 * math.exp(-i / SR * 30)) * i / SR) * math.exp(-i / SR * 7) for i in range(int(0.35 * SR))]
    hat, y = [], 0.0
    for i in range(int(0.06 * SR)):
        x = rng.uniform(-1, 1)
        y = 0.6 * y + 0.4 * x
        hat.append((x - y) * math.exp(-i / SR * 70))

    def saw_note(f, dur, cutoff, amp):
        out, ph, y = [], 0.0, 0.0
        a = 1 - math.exp(-TAU * cutoff / SR)
        m = int(dur * SR)
        for i in range(m):
            ph = (ph + f / SR) % 1.0
            y += a * ((2 * ph - 1) - y)
            out.append(y * amp * min(1, i / 200) * min(1, (m - i) / 600))
        return out

    for b8 in range(8):
        t_bar = b8 * bar
        root, third, fifth = chords[b8 // 2]
        for q in range(4):
            put(t_bar + q * beat, kick, gain=0.55)
            put(t_bar + q * beat + beat / 2, hat, pan=0.3 if q % 2 else -0.3, gain=0.12)
        # Bass: root / fifth / octave eighths, one octave down.
        pattern = [root / 2, root / 2, fifth / 2, root, root / 2, fifth / 2, root, fifth / 2]
        for k, f in enumerate(pattern):
            put(t_bar + k * beat / 2, saw_note(f, beat / 2 * 0.9, 600 + 300 * (k % 2), 0.22))
        # Pad: the chord, swelling over two bars.
        if b8 % 2 == 0:
            m = int(2 * bar * SR)
            for f, pan in ((root, -0.4), (third, 0.0), (fifth, 0.4), (root * 2, 0.0)):
                ph1, ph2 = rng.random() * TAU, rng.random() * TAU
                pad_s = [(math.sin(TAU * f * 1.002 * i / SR + ph1) + math.sin(TAU * f * 0.998 * i / SR + ph2))
                         * min(1, i / (0.6 * SR)) * min(1, (m - i) / (0.5 * SR)) * 0.035 for i in range(m)]
                put(t_bar, pad_s, pan=pan)
        # Arp: sixteenths over the chord, only in the second half for movement.
        if b8 >= 4:
            notes = [root * 2, third * 2, fifth * 2, third * 2]
            for k in range(16):
                f = notes[k % 4]
                m = int(beat / 4 * SR)
                put(t_bar + k * beat / 4, [math.sin(TAU * f * i / SR) * math.exp(-i / SR * 14) * 0.05 for i in range(m)],
                    pan=0.5 if k % 2 else -0.5)
    return L, R


def music(seconds):
    """Synthwave bed for the demo: the loop tiled to length, faded in and out."""
    L, R = synthwave_loop()
    t = Track(seconds)
    n = len(L)
    for i in range(t.n):
        t.l[i] = L[i % n]
        t.r[i] = R[i % n]
    t.reverb(0.18)
    fade_in = int(1.5 * SR)
    for i in range(min(fade_in, t.n)):
        t.l[i] *= i / fade_in
        t.r[i] *= i / fade_in
    t.fade(seconds - 2.5, seconds)
    return t


def bed(seconds):
    """Quiet A-drone bed under the screen recording, leaving room for a voice-over."""
    t = Track(seconds)
    t.add(0.0, pad([A1, E2, A2, E3], seconds, 2.0, 2.0, amp=0.5))
    t.add(0.0, pad([Cs4, Gs4], seconds, 4.0, 3.0, amp=0.12))
    t.fade(seconds - 1.5, seconds)
    return t


if __name__ == "__main__":
    out = sys.argv[1] if len(sys.argv) > 1 else "."
    if len(sys.argv) > 3 and sys.argv[2] == "music":
        music(float(sys.argv[3])).write(f"{out}/music.wav", peak_db=-3.0)
        print("wrote music.wav")
    elif len(sys.argv) > 3 and sys.argv[2] == "bed":
        # Peak well below the intro/outro so a voice-over sits on top.
        bed(float(sys.argv[3])).write(f"{out}/screen-bed.wav", peak_db=-20.0)
        print("wrote screen-bed.wav")
    else:
        intro().write(f"{out}/intro.wav")
        outro().write(f"{out}/outro.wav")
        print("wrote intro.wav and outro.wav")
