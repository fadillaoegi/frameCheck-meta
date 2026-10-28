import type { Asset, Box } from "../../../domain/types";
import { cx } from "./ui";

/**
 * Zoomed view of one evidence region, cut client-side from the stored image
 * (never from a provider-supplied URL). Both sides of a finding use the same
 * window so their scale and position line up; each outlines its own box.
 */
export function EvidenceCrop({
  asset,
  box,
  win,
  label,
  color,
  ghost = false,
}: {
  asset: Asset;
  box: Box | null;
  win: Box;
  label: string;
  color: string;
  ghost?: boolean;
}) {
  return (
    <figure className="min-w-0 space-y-1">
      <figcaption className="font-mono text-[10px] tracking-wider text-ink-3 uppercase">{label}</figcaption>
      <div className="relative w-full overflow-hidden rounded-[3px] bg-white ring-1 ring-ink/15" style={{ aspectRatio: `${win.width} / ${win.height}` }}>
        <img
          src={asset.contentUrl}
          alt=""
          draggable={false}
          className="absolute max-w-none"
          style={{
            width: `${(asset.widthPx / win.width) * 100}%`,
            left: `${(-win.x / win.width) * 100}%`,
            top: `${(-win.y / win.height) * 100}%`,
          }}
        />
        {box && (
          <div
            className={cx("absolute rounded-[2px] border-2", ghost && "border-dashed")}
            style={{
              borderColor: color,
              left: `${((box.x - win.x) / win.width) * 100}%`,
              top: `${((box.y - win.y) / win.height) * 100}%`,
              width: `${(box.width / win.width) * 100}%`,
              height: `${(box.height / win.height) * 100}%`,
            }}
          />
        )}
      </div>
    </figure>
  );
}
