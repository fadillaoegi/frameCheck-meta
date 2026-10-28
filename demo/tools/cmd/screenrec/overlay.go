package main

// overlayJS is injected into every document: a visible cursor that follows
// the dispatched mouse events, click ripples, eased scrolling, a status badge,
// and a preview of the exported Markdown. It only adds elements; it never
// changes the app's behaviour.
const overlayJS = `(() => {
  const fc = (window.__fc = {});
  const ready = (fn) => (document.body ? fn() : document.addEventListener('DOMContentLoaded', fn));

  ready(() => {
    const css = document.createElement('style');
    css.textContent = ` + "`" + `
      #fc-cursor { position: fixed; left: 0; top: 0; z-index: 2147483647; pointer-events: none; width: 26px; height: 26px;
        transform: translate(-100px, -100px); transition: none; filter: drop-shadow(0 2px 3px rgba(0,0,0,.55)); }
      #fc-cursor.down svg { transform: scale(.86); transform-origin: 3px 3px; }
      .fc-ripple { position: fixed; z-index: 2147483646; pointer-events: none; width: 44px; height: 44px; margin: -22px 0 0 -22px;
        border-radius: 50%; border: 2px solid #00f0ff; box-shadow: 0 0 18px #00f0ff; animation: fc-r .55s ease-out forwards; }
      @keyframes fc-r { from { transform: scale(.2); opacity: 1 } to { transform: scale(1.25); opacity: 0 } }
      #fc-badge { position: fixed; left: 50%; bottom: 28px; transform: translateX(-50%); z-index: 2147483645; pointer-events: none;
        font: 600 15px/1 "IBM Plex Mono", monospace; color: #07070e; background: #00f0ff; padding: 10px 16px; border-radius: 999px;
        box-shadow: 0 0 24px rgba(0,240,255,.6); }
      #fc-preview { position: fixed; inset: 0; z-index: 2147483640; display: grid; place-items: center; background: rgba(4,4,10,.72);
        backdrop-filter: blur(3px); animation: fc-in .35s ease-out; }
      @keyframes fc-in { from { opacity: 0; transform: translateY(12px) } to { opacity: 1; transform: none } }
      #fc-preview .win { width: min(980px, 86vw); max-height: 78vh; display: flex; flex-direction: column; border-radius: 14px; overflow: hidden;
        background: #0d0d1a; box-shadow: 0 0 0 1px rgba(0,240,255,.5), 0 0 50px -10px rgba(0,240,255,.6); }
      #fc-preview .bar { padding: 12px 16px; font: 500 13px "IBM Plex Mono", monospace; color: #b6b4dc; border-bottom: 1px solid #24243f; }
      #fc-preview pre { margin: 0; padding: 18px 22px; overflow: auto; font: 13px/1.6 "IBM Plex Mono", monospace; color: #ecebff; white-space: pre-wrap; }
    ` + "`" + `;
    document.head.appendChild(css);
    const c = document.createElement('div');
    c.id = 'fc-cursor';
    c.innerHTML = '<svg width="26" height="26" viewBox="0 0 26 26"><path d="M3 2 L3 21 L8.2 16.3 L11.6 24 L15 22.5 L11.7 15 L18.5 15 Z" fill="#fff" stroke="#000" stroke-width="1.4" stroke-linejoin="round"/></svg>';
    document.body.appendChild(c);
    window.addEventListener('mousemove', (e) => fc.cursor(e.clientX, e.clientY), true);
    window.addEventListener('mousedown', (e) => { c.classList.add('down'); fc.ripple(e.clientX, e.clientY); }, true);
    window.addEventListener('mouseup', () => c.classList.remove('down'), true);
  });

  fc.cursor = (x, y) => {
    const c = document.getElementById('fc-cursor');
    if (c) c.style.transform = 'translate(' + (x - 3) + 'px,' + (y - 2) + 'px)';
  };
  fc.ripple = (x, y) => {
    const r = document.createElement('div');
    r.className = 'fc-ripple';
    r.style.left = x + 'px';
    r.style.top = y + 'px';
    document.body.appendChild(r);
    setTimeout(() => r.remove(), 600);
  };

  // Element finders used by the recorder.
  fc.q = (sel, nth = 0) => document.querySelectorAll(sel)[nth] || null;
  fc.text = (tag, s, root = document) => [...root.querySelectorAll(tag)].find((e) => e.textContent.includes(s)) || null;
  fc.exact = (tag, s) => [...document.querySelectorAll(tag)].find((e) => e.textContent.trim() === s) || null;

  const ease = (t) => (t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2);
  const animate = (from, to, ms, set) => new Promise((done) => {
    const t0 = performance.now();
    const step = (now) => {
      const t = Math.min(1, (now - t0) / ms);
      set(from + (to - from) * ease(t));
      t < 1 ? requestAnimationFrame(step) : done(true);
    };
    requestAnimationFrame(step);
  });
  fc.scroll = (y, ms) => {
    const max = document.documentElement.scrollHeight - innerHeight;
    return animate(scrollY, Math.min(Math.max(0, y), max), ms, (v) => window.scrollTo(0, v));
  };
  fc.scrollIn = (sel, by, ms) => {
    const e = document.querySelector(sel);
    return e ? animate(e.scrollTop, e.scrollTop + by, ms, (v) => (e.scrollTop = v)) : true;
  };

  fc.badge = (text) => {
    let b = document.getElementById('fc-badge');
    if (!text) { b && b.remove(); return true; }
    if (!b) { b = document.createElement('div'); b.id = 'fc-badge'; document.body.appendChild(b); }
    b.textContent = text;
    return true;
  };

  // Shows the exported Markdown, as the downloaded file would read.
  fc.preview = async (show = true) => {
    const old = document.getElementById('fc-preview');
    if (!show) { old && old.remove(); return true; }
    const run = location.hash.split('/runs/')[1];
    const res = await fetch('/v1/runs/' + run + '/export?format=markdown');
    const name = (res.headers.get('content-disposition') || '').split('filename=')[1]?.replaceAll('"', '') || 'report.md';
    const md = await res.text();
    const p = document.createElement('div');
    p.id = 'fc-preview';
    p.innerHTML = '<div class="win"><div class="bar">⤓ Downloads / ' + name + '</div><pre></pre></div>';
    p.querySelector('pre').textContent = md;
    document.body.appendChild(p);
    return true;
  };
})();`
