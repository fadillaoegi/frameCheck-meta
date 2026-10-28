import { useState } from "react";
import { cropWindow, reviewProblem, unionBox, type Asset, type Decision, type Evidence, type Finding } from "../../../domain/types";
import { EvidenceCrop } from "./EvidenceCrop";
import { useUseCases } from "../context";
import { Button, CATEGORY_INFO, CategoryTag, ErrorBanner, EvidenceBadge, cx, formatTime, inputClass } from "./ui";

export function FindingCard({
  finding,
  index,
  reference,
  actual,
  selected,
  reviewable,
  onSelect,
  onReviewed,
}: {
  finding: Finding;
  index: number;
  reference: Asset;
  actual: Asset;
  selected: boolean;
  reviewable: boolean;
  onSelect: () => void;
  onReviewed: () => void;
}) {
  const uc = useUseCases();
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState<Decision | null>(null);
  const [error, setError] = useState<unknown>(null);
  const info = CATEGORY_INFO[finding.category];
  const current = finding.currentReview;
  const refBox = finding.referenceEvidence?.box ?? null;
  const actBox = finding.actualEvidence?.box ?? null;
  const missing = finding.category === "missing_element";
  // Assumes matched captures (same raster size), which comparisons enforce.
  const union = unionBox([refBox, actBox].filter((b) => b !== null));
  const crop = union && cropWindow(union, Math.min(reference.widthPx, actual.widthPx), Math.min(reference.heightPx, actual.heightPx));

  async function review(decision: Decision) {
    const problem = reviewProblem(decision, reason);
    if (problem) {
      setError(new Error(problem));
      return;
    }
    setBusy(decision);
    setError(null);
    try {
      await uc.reviewFinding(finding.id, decision, reason);
      setReason("");
      onReviewed();
    } catch (err) {
      setError(err);
    } finally {
      setBusy(null);
    }
  }

  return (
    <article
      onClick={onSelect}
      className={cx(
        "relative cursor-pointer overflow-hidden rounded-md border bg-paper-2 pl-4 transition-all",
        selected ? "border-neon/50 shadow-[0_0_28px_-10px_var(--color-neon)]" : "border-line hover:-translate-y-px hover:border-line-2",
      )}
    >
      <span aria-hidden className="absolute inset-y-0 left-0 w-1.5" style={{ background: info.color, boxShadow: `0 0 14px ${info.color}` }} />
      <div className="p-3 pl-0">
        <header className="flex flex-wrap items-center gap-2">
          <span className="grid size-5 place-items-center rounded-sm font-mono text-[11px] font-semibold text-paper" style={{ background: info.color }}>
            {index + 1}
          </span>
          <CategoryTag category={finding.category} />
          <EvidenceBadge status={finding.evidenceStatus} />
          {current && (
            <span
              className={cx(
                "ml-auto rounded-sm px-1.5 py-0.5 font-mono text-[10px] font-semibold tracking-wider text-paper uppercase",
                current.decision === "accepted" ? "bg-accept" : "bg-ink-2",
              )}
            >
              {current.decision}
            </span>
          )}
        </header>

        <p className="mt-2 text-sm leading-snug font-medium text-ink">{finding.summary}</p>
        {finding.checkStep && (
          <p className="mt-1.5 text-xs leading-snug text-ink-2">
            <span className="font-mono text-[10px] tracking-wider text-ink-3 uppercase">How to verify · </span>
            {finding.checkStep}
          </p>
        )}

        {selected && (
          <>
            <p className="mt-2 text-[11px] leading-snug text-ink-3">
              <b className={info.text}>{info.label}:</b> {info.meaning} {info.boxes}
            </p>

            <dl className="mt-2 grid gap-1.5 text-xs sm:grid-cols-2">
              <EvidenceLine label="Design" ev={finding.referenceEvidence} absentText="no box on the design" />
              <EvidenceLine label="Build" ev={finding.actualEvidence} absentText={missing ? "absent — this is the defect" : "no box on the build"} />
            </dl>

            {finding.issues.length > 0 && (
              <ul className="mt-2 list-inside list-disc text-xs text-clipped">
                {finding.issues.map((issue) => (
                  <li key={issue}>{issue}</li>
                ))}
              </ul>
            )}

            {crop && (
              // Wide regions stack so each crop keeps a readable height.
              <div className={cx("mt-3 grid gap-2", crop.width / crop.height > 2.2 ? "grid-cols-1" : "grid-cols-2")}>
                <EvidenceCrop asset={reference} box={refBox} win={crop} label="Design · zoom" color={info.color} />
                <EvidenceCrop
                  asset={actual}
                  box={actBox ?? (missing ? refBox : null)}
                  ghost={!actBox && missing}
                  win={crop}
                  label={actBox ? "Build · zoom" : missing ? "Build · should be here" : "Build · no box"}
                  color={info.color}
                />
              </div>
            )}
          </>
        )}

        {selected && reviewable && (
          <div className="mt-3 space-y-2 border-t border-dashed border-line-2 pt-3" onClick={(e) => e.stopPropagation()}>
            <p className="text-[11px] text-ink-3">
              Is this a real defect? <b className="text-ink-2">Accept</b> keeps it in the report; <b className="text-ink-2">Reject</b> needs a short reason so
              others know why.
            </p>
            <textarea
              className={cx(inputClass, "min-h-14 text-xs")}
              placeholder="Reason (required to reject, optional to accept)"
              maxLength={1000}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
            <ErrorBanner error={error} onDismiss={() => setError(null)} />
            <div className="flex gap-2">
              <Button onClick={() => review("accepted")} disabled={busy !== null}>
                {busy === "accepted" ? "Saving…" : current ? "Change to accept" : "Accept"}
              </Button>
              <Button variant="danger" onClick={() => review("rejected")} disabled={busy !== null}>
                {busy === "rejected" ? "Saving…" : current ? "Change to reject" : "Reject"}
              </Button>
            </div>
            {finding.reviewHistory.length > 0 && (
              <ol className="space-y-0.5 font-mono text-[10px] text-ink-3">
                {finding.reviewHistory
                  .slice()
                  .reverse()
                  .map((r) => (
                    <li key={r.id}>
                      {formatTime(r.reviewedAt)} — <b>{r.decision}</b>
                      {r.reason && `: ${r.reason}`}
                    </li>
                  ))}
              </ol>
            )}
          </div>
        )}
      </div>
    </article>
  );
}

function EvidenceLine({ label, ev, absentText }: { label: string; ev: Evidence | null; absentText: string }) {
  return (
    <div className="rounded-sm bg-paper px-2 py-1 ring-1 ring-line">
      <dt className="font-mono text-[10px] tracking-wider text-ink-3 uppercase">{label}</dt>
      <dd className="text-ink-2">
        {ev?.box ? (
          <>
            <span className="font-mono text-[11px]">
              x{ev.box.x} y{ev.box.y} · {ev.box.width}×{ev.box.height}
            </span>
            {ev.description && <span className="block">{ev.description}</span>}
          </>
        ) : (
          <span className="italic">{absentText}</span>
        )}
      </dd>
    </div>
  );
}
