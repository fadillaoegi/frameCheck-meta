// Shared 3D scene for the FrameCheck intro and outro.
// Everything is a pure function of time t (seconds): the renderer calls
// window.render(t) once per frame, so output is deterministic.

export const W = 1920;
export const H = 1080;

// Neon palette, matching the app's dark theme.
export const C = {
  bg: "#06060c",
  paper: "#ecebff",
  neon: "#00f0ff",
  magenta: "#ff2bd6",
  blueTop: "#00d4f0",
  blueA: "#0098b3",
  blueB: "#006b80",
  inkTop: "#1a1a30",
  inkA: "#13132a",
  inkB: "#0b0b18",
  red: "#ff2e63",
  teal: "#14143a",
  clipped: "#ffb627",
  layout: "#b46bff",
};

// ---------- easing ----------
export const clamp = (x, a = 0, b = 1) => Math.min(b, Math.max(a, x));
export const prog = (t, a, b) => clamp((t - a) / (b - a));
export const easeOut = (x) => 1 - Math.pow(1 - x, 3);
export const easeInOut = (x) => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2);
export const easeOutBack = (x) => {
  const c1 = 1.4, c3 = c1 + 1;
  return 1 + c3 * Math.pow(x - 1, 3) + c1 * Math.pow(x - 1, 2);
};
export const lerp = (a, b, x) => a + (b - a) * x;

// ---------- DOM helpers ----------
export function el(tag, style = {}, parent, cls) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  Object.assign(e.style, style);
  parent?.appendChild(e);
  return e;
}

const P3 = { position: "absolute", left: "0px", top: "0px", width: "0px", height: "0px", transformStyle: "preserve-3d" };

/** A group node: a zero-size 3D origin you can move and rotate. */
export function group(parent) {
  return el("div", { ...P3 }, parent);
}

function face(parent, w, h, color, transform, extra = {}) {
  return el("div", { position: "absolute", left: `${-w / 2}px`, top: `${-h / 2}px`, width: `${w}px`, height: `${h}px`, background: color, transform, ...extra }, parent);
}

/**
 * A box w (x) × d (y) × h (z), sitting on z=0. Shading is baked per face:
 * top lightest, then the two faces that point toward the camera.
 */
export function cuboid(parent, { w, d, h, top, sideA, sideB, topExtra = {} }) {
  const g = group(parent);
  face(g, w, d, sideB, `translateZ(0px)`);
  face(g, w, h, sideA, `translateY(${d / 2}px) translateZ(${h / 2}px) rotateX(90deg)`);
  face(g, w, h, sideB, `translateY(${-d / 2}px) translateZ(${h / 2}px) rotateX(90deg)`);
  face(g, h, d, sideB, `translateX(${w / 2}px) translateZ(${h / 2}px) rotateY(90deg)`);
  face(g, h, d, sideA, `translateX(${-w / 2}px) translateZ(${h / 2}px) rotateY(90deg)`);
  const topFace = face(g, w, d, top, `translateZ(${h}px)`, topExtra);
  return { g, top: topFace };
}

// ---------- the FrameCheck mark in 3D ----------
export const SIZE = 380;
export const THICK = 30;
export const BAR = 92;
export const GAP = 120;

/** Mini web page printed on the design plate: the "design" being checked. */
function designPrint(top) {
  Object.assign(top.style, {
    backgroundImage: `linear-gradient(135deg, ${C.blueTop}, #2fe6ff 55%, #00b6d6)`,
    overflow: "hidden",
    boxShadow: "inset 0 0 0 2px rgba(255,255,255,0.35), 0 0 60px rgba(0,225,255,0.45)",
  });
  const s = SIZE;
  const ink = "rgba(7,7,14,0.8)";
  const faint = "rgba(7,7,14,0.3)";
  el("div", { position: "absolute", left: "26px", top: "24px", width: `${s - 52}px`, height: "14px", borderRadius: "3px", background: faint }, top);
  el("div", { position: "absolute", left: "26px", top: "62px", width: "200px", height: "16px", borderRadius: "3px", background: ink }, top);
  el("div", { position: "absolute", left: "26px", top: "86px", width: "150px", height: "16px", borderRadius: "3px", background: ink }, top);
  // The call-to-action, centred so it reads through the build's window.
  el("div", { position: "absolute", left: `${s / 2 - 50}px`, top: `${s / 2 - 16}px`, width: "100px", height: "32px", borderRadius: "6px", background: C.teal, boxShadow: "0 0 0 2px rgba(7,7,14,0.6)" }, top);
  for (let i = 0; i < 3; i++) {
    el("div", { position: "absolute", left: `${26 + i * 112}px`, top: `${s - 96}px`, width: "96px", height: "70px", borderRadius: "6px", background: "rgba(7,7,14,0.1)", boxShadow: "inset 0 0 0 1.5px rgba(7,7,14,0.35)" }, top);
  }
}

export function buildMark(parent) {
  const root = group(parent);

  const designG = group(root);
  const plate = cuboid(designG, { w: SIZE, d: SIZE, h: THICK, top: C.blueTop, sideA: C.blueA, sideB: C.blueB });
  designPrint(plate.top);

  // Registration crosshair + finding dot, printed on top of the design.
  const markG = group(designG);
  markG.style.transform = `translateZ(${THICK + 0.5}px)`;
  const cross = [];
  cross.push(el("div", { position: "absolute", left: "-120px", top: "-1px", width: "240px", height: "2px", background: "rgba(7,7,14,0.85)" }, markG));
  cross.push(el("div", { position: "absolute", left: "-1px", top: "-120px", width: "2px", height: "240px", background: "rgba(7,7,14,0.85)" }, markG));
  const circle = el("div", { position: "absolute", left: "-46px", top: "-46px", width: "92px", height: "92px", borderRadius: "50%", border: "2px solid rgba(7,7,14,0.85)", boxSizing: "border-box" }, markG);
  const dot = el("div", { position: "absolute", left: "-15px", top: "-15px", width: "30px", height: "30px", borderRadius: "50%", background: C.red, boxShadow: `0 0 30px ${C.red}` }, markG);
  const ring = el("div", { position: "absolute", left: "-15px", top: "-15px", width: "30px", height: "30px", borderRadius: "50%", border: `3px solid ${C.red}`, boxSizing: "border-box" }, markG);

  // The build: a frame of four bars, open in the middle.
  const frameG = group(root);
  const inner = SIZE - 2 * BAR;
  const bar = (w, d, x, y) => {
    const c = cuboid(frameG, {
      w, d, h: THICK, top: C.inkTop, sideA: C.inkA, sideB: C.inkB,
      topExtra: { boxShadow: "inset 0 0 0 2px rgba(255,43,214,0.85), 0 0 24px rgba(255,43,214,0.35)" },
    });
    c.g.style.transform = `translate3d(${x}px, ${y}px, 0px)`;
  };
  bar(SIZE, BAR, 0, -(inner / 2 + BAR / 2));
  bar(SIZE, BAR, 0, inner / 2 + BAR / 2);
  bar(BAR, inner, -(inner / 2 + BAR / 2), 0);
  bar(BAR, inner, inner / 2 + BAR / 2, 0);

  // Evidence post: a dotted red line standing up from the dot. Built from
  // small discs stacked in z; thin vertical planes get culled at some angles.
  const postG = group(root);
  const beads = [];
  for (let i = 0; i < 40; i++) {
    beads.push(el("div", { position: "absolute", left: "-5px", top: "-5px", width: "10px", height: "10px", borderRadius: "50%", background: C.red, boxShadow: `0 0 10px ${C.red}`, transform: `translateZ(${THICK + 1 + i * 9}px)` }, postG));
  }
  const headG = group(postG);
  const head = el("div", { position: "absolute", left: "-16px", top: "-16px", width: "32px", height: "32px", borderRadius: "50%", background: C.red, boxShadow: `0 0 40px ${C.red}` }, headG);

  // Soft contact shadow under the design plate.
  const shadow = el("div", {
    position: "absolute", left: `${-SIZE * 0.65}px`, top: `${-SIZE * 0.65}px`, width: `${SIZE * 1.3}px`, height: `${SIZE * 1.3}px`,
    background: "radial-gradient(closest-side, rgba(0,0,0,0.75), rgba(0,0,0,0))", transform: "translateZ(-2px)",
  }, root);

  return {
    root, designG, frameG, markG, cross, circle, dot, ring, headG, head, shadow,
    /** Height of the post in px; it stands from the design's top face. */
    setPost(h) {
      beads.forEach((b, i) => (b.style.opacity = i * 9 <= h ? "1" : "0"));
      headG.style.transform = `translateZ(${THICK + 1 + h}px)`;
    },
  };
}

// ---------- environment ----------
export function buildStage() {
  document.body.style.cssText = `margin:0;width:${W}px;height:${H}px;overflow:hidden;background:${C.bg};`;
  const view = el("div", { position: "absolute", inset: "0", perspective: "1700px", perspectiveOrigin: "50% 45%", overflow: "hidden" }, document.body);
  const cam = el("div", { position: "absolute", left: "50%", top: "50%", width: "0", height: "0", transformStyle: "preserve-3d" }, view);
  const world = group(cam);

  // Infinite-looking floor grid, fading with distance.
  const floor = el("div", {
    position: "absolute", left: "-3000px", top: "-3000px", width: "6000px", height: "6000px",
    transform: "translateZ(-160px)",
    backgroundImage:
      "linear-gradient(rgba(0,240,255,0.16) 2px, transparent 2px), linear-gradient(90deg, rgba(0,240,255,0.16) 2px, transparent 2px)," +
      "linear-gradient(rgba(0,240,255,0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(0,240,255,0.05) 1px, transparent 1px)",
    backgroundSize: "200px 200px, 200px 200px, 40px 40px, 40px 40px",
    backgroundPosition: "center",
    WebkitMaskImage: "radial-gradient(closest-side, #000 15%, transparent 75%)",
    maskImage: "radial-gradient(closest-side, #000 15%, transparent 75%)",
  }, world);

  // Floor crosshair: the registration target the whole scene is aligned to.
  const floorMarks = group(world);
  floorMarks.style.transform = "translateZ(-159px)";
  const fx = el("div", { position: "absolute", left: "-900px", top: "-1px", width: "1800px", height: "2px", background: "rgba(255,43,214,0.7)", boxShadow: "0 0 12px rgba(255,43,214,0.8)", transformOrigin: "50% 50%" }, floorMarks);
  const fy = el("div", { position: "absolute", left: "-1px", top: "-900px", width: "2px", height: "1800px", background: "rgba(255,43,214,0.7)", boxShadow: "0 0 12px rgba(255,43,214,0.8)", transformOrigin: "50% 50%" }, floorMarks);
  const fc = el("div", { position: "absolute", left: "-260px", top: "-260px", width: "520px", height: "520px", borderRadius: "50%", border: "2px solid rgba(255,43,214,0.55)", boxShadow: "0 0 18px rgba(255,43,214,0.5)", boxSizing: "border-box" }, floorMarks);

  // Dust motes for depth, from a fixed seed.
  let seed = 7;
  const rnd = () => ((seed = (seed * 16807) % 2147483647) / 2147483647);
  const motes = [];
  for (let i = 0; i < 70; i++) {
    const m = el("div", {
      position: "absolute", width: "4px", height: "4px", left: "-2px", top: "-2px", borderRadius: "50%",
      background: "rgba(0,240,255,0.8)", boxShadow: "0 0 6px rgba(0,240,255,0.9)",
    }, world);
    motes.push({ m, x: (rnd() - 0.5) * 2600, y: (rnd() - 0.5) * 2600, z: rnd() * 900 - 100, s: 0.5 + rnd() * 1.6, p: rnd() * 6.28 });
  }

  const vignette = el("div", { position: "absolute", inset: "0", pointerEvents: "none", background: "radial-gradient(ellipse at 50% 45%, transparent 35%, rgba(0,0,0,0.75) 100%), radial-gradient(50rem 30rem at 0% 0%, rgba(255,43,214,0.12), transparent 70%), radial-gradient(50rem 30rem at 100% 100%, rgba(0,240,255,0.10), transparent 70%)" }, document.body);
  // Text sits above the 3D view; the fade overlay sits above everything.
  const hud = el("div", { position: "absolute", inset: "0", pointerEvents: "none" }, document.body);
  const overlay = el("div", { position: "absolute", inset: "0", pointerEvents: "none", background: "#000", opacity: "0" }, document.body);

  return {
    view, cam, world, floor, floorMarks, fx, fy, fc, motes, overlay, hud,
    setCamera({ rx, rz, z = 0, y = 0, x = 0 }) {
      cam.style.transform = `translate3d(${x}px, ${y}px, ${z}px) rotateX(${rx}deg) rotateZ(${rz}deg)`;
    },
    animateMotes(t, alpha) {
      for (const d of motes) {
        const z = d.z + Math.sin(t * 0.6 + d.p) * 30 + t * 12;
        d.m.style.transform = `translate3d(${d.x}px, ${d.y}px, ${z}px) scale(${d.s})`;
        d.m.style.opacity = String(alpha * (0.25 + 0.35 * Math.sin(t * 1.3 + d.p) ** 2));
      }
    },
  };
}

/** Big serif wordmark and lines of text, drawn flat over the 3D view. */
export function text(parent, content, style) {
  return el("div", {
    position: "absolute", left: "0", right: "0", textAlign: "center", color: C.paper, whiteSpace: "nowrap", ...style,
  }, parent).appendChild(document.createTextNode(content)).parentNode;
}

export async function ready() {
  await document.fonts.ready;
  await Promise.all([...document.fonts].map((f) => f.load().catch(() => null)));
}
