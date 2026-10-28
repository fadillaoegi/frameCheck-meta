import { useEffect, useRef, useState, type PointerEvent } from "react";
import { boxToPercent, type Box } from "../../../domain/types";
import { cx } from "./ui";

export interface LayerBox {
  id: string;
  box: Box;
  color: string;
  label?: string;
  /** Dashed outline: marks where something should be but is not. */
  ghost?: boolean;
  selected?: boolean;
}

export interface Layer {
  key: string;
  title: string;
  tone: string;
  src: string;
  widthPx: number;
  heightPx: number;
  boxes: LayerBox[];
}

/**
 * Two screenshots as stacked sheets in 3D. spread 0 is a flat onion-skin
 * overlay (top layer at 50%); spread 1 pulls the layers apart so each can be
 * read on its own while staying registered to the other. A post links the
 * selected region across layers.
 */
export function Stack3D({
  layers,
  spread,
  tilt = "drag",
  link,
  onSelect,
  className,
}: {
  layers: [Layer, ...Layer[]];
  spread: number;
  tilt?: "drag" | "hover" | "none";
  link?: { box: Box; color: string } | null;
  onSelect?: (id: string) => void;
  className?: string;
}) {
  const [yaw, setYaw] = useState(0);
  const [pitch, setPitch] = useState(0);
  const [dragging, setDragging] = useState(false);
  const start = useRef<{ x: number; y: number; yaw: number; pitch: number } | null>(null);
  const root = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(640);

  // Layer separation scales with the rendered width so phones and desktops look alike.
  useEffect(() => {
    const el = root.current;
    if (!el) return;
    const ro = new ResizeObserver(([e]) => e && setWidth(e.contentRect.width));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const base = layers[0];
  const gap = 0.3 * width * spread;
  const rx = Math.max(0, 56 * spread + pitch);
  const rz = -30 * spread + yaw;
  const scale = 1 - 0.2 * spread;

  function onDown(e: PointerEvent<HTMLDivElement>) {
    if (tilt !== "drag" || (e.target as HTMLElement).closest("button")) return;
    e.currentTarget.setPointerCapture(e.pointerId);
    start.current = { x: e.clientX, y: e.clientY, yaw, pitch };
    setDragging(true);
  }
  function onMove(e: PointerEvent<HTMLDivElement>) {
    if (tilt === "hover") {
      const r = e.currentTarget.getBoundingClientRect();
      setYaw(((e.clientX - r.left) / r.width - 0.5) * 14);
      setPitch(((e.clientY - r.top) / r.height - 0.5) * -10);
      return;
    }
    if (!start.current) return;
    setYaw(clamp(start.current.yaw + (e.clientX - start.current.x) * 0.25, -50, 50));
    setPitch(clamp(start.current.pitch + (e.clientY - start.current.y) * -0.2, -30, 25));
  }
  function onUp() {
    start.current = null;
    setDragging(false);
  }

  const linkPct = link ? boxToPercent(link.box, base.widthPx, base.heightPx) : null;

  return (
    <div
      ref={root}
      className={cx("relative touch-none select-none", tilt === "drag" && (dragging ? "cursor-grabbing" : "cursor-grab"), className)}
      style={{ perspective: "2000px", paddingTop: `${gap * 0.7}px` }}
      onPointerDown={onDown}
      onPointerMove={onMove}
      onPointerUp={onUp}
      onPointerCancel={onUp}
      onPointerLeave={() => tilt === "hover" && (setYaw(0), setPitch(0))}
      onDoubleClick={() => (setYaw(0), setPitch(0))}
    >
      <div
        className={cx("relative w-full", !dragging && "transition-transform duration-700 ease-[cubic-bezier(0.2,0.8,0.2,1)]")}
        style={{
          aspectRatio: `${base.widthPx} / ${base.heightPx}`,
          transformStyle: "preserve-3d",
          transform: `scale(${scale}) rotateX(${rx}deg) rotateZ(${rz}deg)`,
        }}
      >
        {layers.map((layer, i) => (
          <div
            key={layer.key}
            className="absolute inset-0 transition-[transform,opacity] duration-700 ease-[cubic-bezier(0.2,0.8,0.2,1)]"
            style={{
              transform: `translateZ(${i * gap}px)`,
              // Upper layers stay slightly see-through so the one below still reads.
              opacity: i === 0 ? 1 : 0.5 + 0.42 * spread,
              transformStyle: "preserve-3d",
            }}
          >
            <div
              className="absolute inset-0 overflow-hidden rounded-[3px] bg-white ring-1 ring-ink/25"
              style={{ boxShadow: `0 ${40 * spread}px ${70 * spread}px -${24 * spread}px rgb(0 0 0 / ${0.7 * spread})` }}
            >
              <img src={layer.src} alt={`${layer.title} layer`} draggable={false} className="block h-full w-full" />
              {layer.boxes.map((b) => {
                const p = boxToPercent(b.box, layer.widthPx, layer.heightPx);
                return (
                  <button
                    key={b.id}
                    type="button"
                    onClick={() => onSelect?.(b.id)}
                    aria-label={b.label ?? "Finding"}
                    className={cx("absolute rounded-[2px]", b.ghost ? "border-2 border-dashed" : "border-2", b.selected && "z-10")}
                    style={{
                      left: `${p.left}%`,
                      top: `${p.top}%`,
                      width: `${p.width}%`,
                      height: `${p.height}%`,
                      borderColor: b.color,
                      background: b.selected ? `color-mix(in srgb, ${b.color} 22%, transparent)` : `color-mix(in srgb, ${b.color} 8%, transparent)`,
                      outline: b.selected ? `3px solid color-mix(in srgb, ${b.color} 30%, transparent)` : undefined,
                      boxShadow: `0 0 16px ${b.color}`,
                    }}
                  >
                    {b.label && (
                      <span className="absolute -top-px left-full ml-1 rounded-sm px-1 font-mono text-[10px] leading-4 whitespace-nowrap text-paper" style={{ background: b.color }}>
                        {b.label}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
            <span
              className="absolute -top-6 left-0 inline-flex items-center gap-1.5 rounded-t-md px-2 py-1 font-mono text-[10px] font-semibold tracking-widest text-paper uppercase transition-opacity duration-500"
              style={{ background: layer.tone, opacity: spread > 0.15 ? 1 : 0 }}
            >
              {layer.title}
            </span>
          </div>
        ))}

        {linkPct && layers.length > 1 && (
          <span
            aria-hidden
            className="pointer-events-none absolute w-0.5 origin-top transition-[height,opacity] duration-700"
            style={{
              left: `${linkPct.left + linkPct.width / 2}%`,
              top: `${linkPct.top + linkPct.height / 2}%`,
              height: `${gap * (layers.length - 1)}px`,
              // Stand the line up out of the design plane toward the build layer.
              transform: "rotateX(90deg)",
              background: `repeating-linear-gradient(to bottom, ${link!.color} 0 6px, transparent 6px 10px)`,
              opacity: spread > 0.15 ? 1 : 0,
            }}
          />
        )}
      </div>
    </div>
  );
}

function clamp(n: number, lo: number, hi: number) {
  return Math.min(hi, Math.max(lo, n));
}
