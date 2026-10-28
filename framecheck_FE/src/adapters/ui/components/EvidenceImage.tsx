import { boxToPercent, type Asset, type Finding, type ImageRole } from "../../../domain/types";
import { CATEGORY_INFO, CropMarks, cx } from "./ui";

/**
 * Shows one image with every finding's evidence box for that role. Boxes are
 * placed in percentages of the original raster, so they stay aligned at any
 * display size. A missing element is drawn as a dashed ghost on the build, at
 * the place the design puts it.
 */
export function EvidenceImage({
  asset,
  role,
  findings,
  selectedId,
  onSelect,
  scanning = false,
}: {
  asset: Asset;
  role: ImageRole;
  findings: Finding[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  scanning?: boolean;
}) {
  const design = role === "reference";
  return (
    <figure className="space-y-2.5">
      <figcaption className="flex items-baseline justify-between gap-2 text-xs">
        <span className="inline-flex items-center gap-1.5">
          <span className={cx("rounded-sm px-1.5 py-0.5 font-mono text-[10px] font-semibold tracking-widest text-paper uppercase", design ? "bg-design" : "bg-build")}>
            {design ? "Design" : "Build"}
          </span>
          <span className="text-ink-2">{design ? "what it should look like" : "what was implemented"}</span>
        </span>
        <span className="font-mono text-[11px] text-ink-3">
          {asset.widthPx}×{asset.heightPx}
        </span>
      </figcaption>
      <div className="relative">
        <CropMarks />
        <div className="checker relative overflow-hidden rounded-[3px] ring-1 ring-ink/20">
          <img src={asset.contentUrl} alt={`${design ? "Design" : "Build"} image`} className="block h-auto w-full" draggable={false} />
          {findings.map((f, i) => {
            const own = design ? f.referenceEvidence : f.actualEvidence;
            // A missing element has no box on the build; show where it should have been.
            const ghost = !design && !own?.box && f.category === "missing_element" ? f.referenceEvidence?.box : null;
            const box = own?.box ?? ghost;
            if (!box) return null;
            const p = boxToPercent(box, asset.widthPx, asset.heightPx);
            const selected = f.id === selectedId;
            const color = CATEGORY_INFO[f.category].color;
            return (
              <button
                key={f.id}
                type="button"
                onClick={() => onSelect(f.id)}
                aria-label={`Finding ${i + 1}: ${f.summary}`}
                title={ghost ? "Expected here, missing in the build" : own?.description || f.summary}
                className={cx("mark-in absolute rounded-[2px] border-2 transition-[background-color,box-shadow]", ghost && "border-dashed", selected && "z-10")}
                style={{
                  left: `${p.left}%`,
                  top: `${p.top}%`,
                  width: `${p.width}%`,
                  height: `${p.height}%`,
                  borderColor: color,
                  background: `color-mix(in srgb, ${color} ${selected ? 20 : 6}%, transparent)`,
                  boxShadow: selected ? `0 0 0 3px color-mix(in srgb, ${color} 30%, transparent), 0 0 22px ${color}` : `0 0 10px color-mix(in srgb, ${color} 60%, transparent)`,
                  opacity: selectedId && !selected ? 0.55 : 1,
                }}
              >
                <span className="absolute -top-px -left-px rounded-br-sm px-1 font-mono text-[10px] leading-4 font-semibold text-paper" style={{ background: color }}>
                  {i + 1}
                  {ghost && " missing"}
                </span>
              </button>
            );
          })}
          {scanning && (
            <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
              <div className="scanline h-[10%] w-full bg-gradient-to-b from-transparent via-design/25 to-design/0" />
            </div>
          )}
        </div>
      </div>
    </figure>
  );
}
