import { useState } from "react";
import type { Asset } from "../../../domain/types";

/** Design over build with a draggable divider. Comparisons guarantee equal raster sizes. */
export function SwipeCompare({ reference, actual }: { reference: Asset; actual: Asset }) {
  const [pos, setPos] = useState(50);
  return (
    <div className="space-y-2">
      <div className="relative overflow-hidden rounded-[3px] bg-white ring-1 ring-ink/20" style={{ aspectRatio: `${actual.widthPx} / ${actual.heightPx}` }}>
        <img src={actual.contentUrl} alt="Build" draggable={false} className="absolute inset-0 h-full w-full" />
        <img
          src={reference.contentUrl}
          alt="Design"
          draggable={false}
          className="absolute inset-0 h-full w-full"
          style={{ clipPath: `inset(0 ${100 - pos}% 0 0)` }}
        />
        <div className="pointer-events-none absolute inset-y-0 w-0.5 bg-neon shadow-[0_0_12px_var(--color-neon)]" style={{ left: `${pos}%` }}>
          <span className="absolute top-1/2 left-1/2 grid size-8 -translate-1/2 place-items-center rounded-full bg-neon font-mono text-xs text-paper shadow-[0_0_18px_var(--color-neon)]">
            ⇆
          </span>
        </div>
        <span className="absolute top-2 left-2 rounded-sm bg-design px-1.5 py-0.5 font-mono text-[10px] font-semibold tracking-widest text-paper uppercase">Design</span>
        <span className="absolute top-2 right-2 rounded-sm bg-build px-1.5 py-0.5 font-mono text-[10px] font-semibold tracking-widest text-paper uppercase">Build</span>
        <input
          type="range"
          min={0}
          max={100}
          step={0.5}
          value={pos}
          onChange={(e) => setPos(Number(e.target.value))}
          aria-label="Divider position: left shows the design, right shows the build"
          className="absolute inset-0 h-full w-full cursor-ew-resize opacity-0"
        />
      </div>
      <p className="text-[11px] text-ink-3">Drag across the image. Anything that jumps or vanishes as the divider passes is a difference.</p>
    </div>
  );
}
