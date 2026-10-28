import { useEffect, useState, type ReactNode } from "react";
import { isRunActive, isRunFinished, tallyReviews, type Category, type ComparisonDetail, type Finding, type RunDetail } from "../../../domain/types";
import { ApiError, type ExportFormat } from "../../../application/ports";
import { useUseCases } from "../context";
import { href, navigate } from "../routes";
import { EvidenceImage } from "../components/EvidenceImage";
import { FindingCard } from "../components/FindingCard";
import { Stack3D, type Layer } from "../components/Stack3D";
import { SwipeCompare } from "../components/SwipeCompare";
import { Button, CATEGORY_INFO, CategoryGlyph, ErrorBanner, OriginBadge, StatusBadge, Steps, cx, formatTime, useLoad, type Step } from "../components/ui";

type View = "side" | "swipe" | "layers";

const VIEWS: { id: View; label: string; help: string }[] = [
  { id: "side", label: "Side by side", help: "Design on the left, build on the right. Click a box to open its finding." },
  { id: "swipe", label: "Swipe", help: "Drag the divider to wipe between design and build at the same spot." },
  { id: "layers", label: "Layers 3D", help: "The build floats above the design. Drag to turn the stack; slide to flatten it into an overlay." },
];

export function ComparisonPage({ comparisonId, runId }: { comparisonId: string; runId?: string }) {
  const uc = useUseCases();
  const { data: cmp, error, reload } = useLoad(() => uc.getComparison(comparisonId), [uc, comparisonId]);
  const [run, setRun] = useState<RunDetail | null>(null);
  const [runError, setRunError] = useState<unknown>(null);
  const [starting, setStarting] = useState(false);
  // Set when these exact inputs were analysed before: the earlier run's id.
  const [duplicateOf, setDuplicateOf] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);
  const [view, setView] = useState<View>("side");

  const activeRunId = runId ?? cmp?.runs[0]?.id;

  // Load the selected run and poll while it is queued or running.
  useEffect(() => {
    if (!activeRunId) {
      setRun(null);
      return;
    }
    const ctrl = new AbortController();
    uc.waitForRun(activeRunId, setRun, ctrl.signal)
      .then(() => !ctrl.signal.aborted && reload())
      .catch((err) => !ctrl.signal.aborted && setRunError(err));
    return () => ctrl.abort();
    // reload is recreated each render; polling must restart only when the run changes
  }, [uc, activeRunId, refreshKey]);

  async function start(allowDuplicate: boolean) {
    setStarting(true);
    setRunError(null);
    setDuplicateOf(null);
    try {
      const r = await uc.startRun(comparisonId, allowDuplicate);
      setSelectedId(null);
      navigate(href.comparison(comparisonId, r.id));
      reload();
    } catch (err) {
      if (err instanceof ApiError && err.code === "DUPLICATE_RUN") setDuplicateOf(err.message.match(/[0-9a-f-]{36}/)?.[0] ?? "");
      else setRunError(err);
    } finally {
      setStarting(false);
    }
  }

  async function openEarlier() {
    if (!duplicateOf) return;
    try {
      const earlier = await uc.getRun(duplicateOf);
      setDuplicateOf(null);
      navigate(href.comparison(earlier.comparisonId, earlier.id));
    } catch (err) {
      setRunError(err);
    }
  }

  async function download(format: ExportFormat) {
    if (!run) return;
    try {
      const file = await uc.exportRun(run.id, format);
      const url = URL.createObjectURL(file.blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = file.filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setRunError(err);
    }
  }

  if (error) return <ErrorBanner error={error} />;
  if (!cmp) return <p className="text-sm text-ink-3">Loading…</p>;

  const busy = cmp.runs.some((r) => isRunActive(r.status));
  const findings = run?.findings ?? [];
  const active = run ? isRunActive(run.status) : false;
  const finished = run ? isRunFinished(run.status) : false;
  const tally = tallyReviews(findings);
  const cap = cmp.reference.capture;
  const selected = findings.find((f) => f.id === selectedId) ?? null;

  const step: Step = !finished ? "Analyse" : tally.unreviewed > 0 ? "Review" : "Export";
  const done: Step[] = ["Upload", "Pair", ...(finished ? (["Analyse"] as Step[]) : []), ...(finished && tally.unreviewed === 0 ? (["Review"] as Step[]) : [])];

  const runButton = (
    <Button onClick={() => start(false)} disabled={starting || busy}>
      {busy ? "Analysing…" : starting ? "Starting…" : run ? "Run again" : "Run analysis →"}
    </Button>
  );

  return (
    <div className="space-y-5">
      <div className="space-y-3">
        <a href={href.project(cmp.projectId)} className="font-mono text-[11px] text-ink-3 hover:text-ink">
          ← Project
        </a>
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h1 className="font-display text-4xl leading-none font-semibold">
              Comparison <span className="font-mono text-lg text-ink-3">{cmp.id.slice(0, 8)}</span>
            </h1>
            <p className="mt-1 font-mono text-[11px] text-ink-3">
              {cap.viewportWidthCss}×{cap.viewportHeightCss} CSS @{cap.devicePixelRatio}x · state “{cap.contentStateKey}” · notes v{cmp.requirement.version}
            </p>
          </div>
          <Steps current={step} done={done} />
        </div>
      </div>

      <NextStep
        step={step}
        run={run}
        tally={tally}
        duplicate={duplicateOf !== null}
        action={
          duplicateOf !== null ? (
            <div className="flex flex-wrap gap-2">
              {duplicateOf && <Button onClick={openEarlier}>Open earlier result</Button>}
              <Button variant="secondary" onClick={() => start(true)} disabled={starting || busy}>
                Run again anyway
              </Button>
            </div>
          ) : step === "Analyse" ? (
            runButton
          ) : step === "Export" ? (
            <div className="flex flex-wrap gap-2">
              <Button onClick={() => download("markdown")}>Download Markdown</Button>
              <Button variant="secondary" onClick={() => download("json")}>
                JSON
              </Button>
            </div>
          ) : null
        }
      />

      <ErrorBanner error={runError} onDismiss={() => setRunError(null)} />

      {run && <RunTicket run={run} runs={cmp.runs} onPick={(id) => (setSelectedId(null), navigate(href.comparison(comparisonId, id)))} rerun={step !== "Analyse" ? runButton : null} />}

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_400px]">
        {/* Sticky on wide screens so the images stay in view while reviewing a long findings list. */}
        <section className="sheet-shadow min-w-0 space-y-4 self-start rounded-lg border border-line bg-sheet p-4 xl:sticky xl:top-16">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div role="tablist" aria-label="View" className="inline-flex rounded-md bg-paper p-0.5 ring-1 ring-line">
              {VIEWS.map((v) => (
                <button
                  key={v.id}
                  role="tab"
                  type="button"
                  aria-selected={view === v.id}
                  onClick={() => setView(v.id)}
                  className={cx("rounded px-3 py-1 text-xs font-medium transition-colors", view === v.id ? "bg-neon text-paper shadow-[0_0_14px_-3px_var(--color-neon)]" : "text-ink-2 hover:text-neon")}
                >
                  {v.label}
                </button>
              ))}
            </div>
            <Legend findings={findings} />
          </div>
          <p className="text-[11px] text-ink-3">{VIEWS.find((v) => v.id === view)!.help}</p>

          {cmp.requirement.text && (
            <p className="rounded-sm border-l-2 border-ink-3 bg-paper px-3 py-1.5 text-xs text-ink-2">
              <span className="mr-2 font-mono text-[10px] tracking-wider text-ink-3 uppercase">Notes</span>
              {cmp.requirement.text}
            </p>
          )}

          {view === "side" && (
            <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
              <EvidenceImage asset={cmp.reference} role="reference" findings={findings} selectedId={selectedId} onSelect={setSelectedId} scanning={active} />
              <EvidenceImage asset={cmp.actual} role="actual" findings={findings} selectedId={selectedId} onSelect={setSelectedId} scanning={active} />
            </div>
          )}
          {view === "swipe" && <SwipeCompare reference={cmp.reference} actual={cmp.actual} />}
          {view === "layers" && <LayersView cmp={cmp} findings={findings} selected={selected} onSelect={setSelectedId} />}
        </section>

        <aside className="space-y-3">
          <section className="sheet-shadow rounded-lg border border-line bg-sheet">
            <header className="space-y-2 border-b border-line/70 px-4 py-3">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-semibold">{run ? `Findings (${findings.length})` : "Findings"}</h2>
                {finished && (
                  <div className="flex gap-1">
                    <Button variant="ghost" className="px-2 py-1 font-mono text-[11px]" onClick={() => download("markdown")}>
                      ⤓ MD
                    </Button>
                    <Button variant="ghost" className="px-2 py-1 font-mono text-[11px]" onClick={() => download("json")}>
                      ⤓ JSON
                    </Button>
                  </div>
                )}
              </div>
              {findings.length > 0 && <ReviewProgress tally={tally} total={findings.length} />}
            </header>
            <div className="space-y-2 p-3">
              {!run && <Empty title="Not analysed yet">Press “Run analysis”. Findings appear here, and their boxes appear on the images.</Empty>}
              {active && <Empty title="Analysing…">Both images are being compared. This page updates by itself.</Empty>}
              {run?.status === "failed" && (
                <Empty title="The analysis failed" tone="bad">
                  {run.failureMessage} <span className="font-mono">({run.failureCode})</span>. Nothing was recorded, so this is not a pass. You can run it
                  again.
                </Empty>
              )}
              {finished && findings.length === 0 && (
                <Empty title="No findings" tone="good">
                  Nothing was found in the three checked categories. That is not a sign-off: colour and wording are not checked.
                </Empty>
              )}
              {findings.map((f, i) => (
                <FindingCard
                  key={f.id}
                  finding={f}
                  index={i}
                  reference={cmp.reference}
                  actual={cmp.actual}
                  selected={f.id === selectedId}
                  reviewable={finished}
                  onSelect={() => setSelectedId(f.id)}
                  onReviewed={() => setRefreshKey((k) => k + 1)}
                />
              ))}
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}

function NextStep({
  step,
  run,
  tally,
  duplicate,
  action,
}: {
  step: Step;
  run: RunDetail | null;
  tally: ReturnType<typeof tallyReviews>;
  duplicate: boolean;
  action: ReactNode;
}) {
  let title: string;
  let text: string;
  if (duplicate) {
    title = "These exact images were already analysed";
    text = "Open the earlier result to avoid another model request, or run again on purpose (for example after changing the model settings).";
  } else if (step === "Analyse") {
    if (run && isRunActive(run.status)) {
      title = "Analysing the pair";
      text = "The model is comparing the design and the build. It usually takes a few seconds to a minute.";
    } else if (run?.status === "failed") {
      title = "The last analysis failed";
      text = "See the reason on the right. Fix it if you can, then run the analysis again.";
    } else {
      title = "Next: run the analysis";
      text = "FrameCheck sends both images to the model and asks for missing elements, clipped text and layout mismatches, each with boxes.";
    }
  } else if (step === "Review") {
    title = `Next: review ${tally.unreviewed} finding${tally.unreviewed === 1 ? "" : "s"}`;
    text = "Click a finding to zoom into both images, then Accept it if it is a real defect or Reject it with a reason.";
  } else {
    title = "All findings reviewed";
    text = "Download the report: Markdown to paste into a PR or ticket, JSON for your own tooling. Only your decisions are marked as accepted.";
  }
  return (
    <div className="glow flex flex-wrap items-center justify-between gap-3 rounded-lg bg-sheet px-4 py-3">
      <div className="min-w-0">
        <p className="font-mono text-[10px] tracking-widest text-neon uppercase">▸ Step · {step}</p>
        <p className="text-sm font-semibold">{title}</p>
        <p className="text-xs text-ink-2">{text}</p>
      </div>
      {action && <div>{action}</div>}
    </div>
  );
}

function ReviewProgress({ tally, total }: { tally: ReturnType<typeof tallyReviews>; total: number }) {
  const pct = (n: number) => `${(n / total) * 100}%`;
  return (
    <div className="space-y-1">
      <div className="flex h-1.5 overflow-hidden rounded-full bg-paper-2">
        <div className="bg-accept transition-all" style={{ width: pct(tally.accepted) }} />
        <div className="bg-ink-3 transition-all" style={{ width: pct(tally.rejected) }} />
      </div>
      <p className="font-mono text-[10px] text-ink-3">
        <span className="text-accept">{tally.accepted} accepted</span> · {tally.rejected} rejected ·{" "}
        <b className={tally.unreviewed ? "text-ink" : ""}>{tally.unreviewed} to review</b>
      </p>
    </div>
  );
}

function Empty({ title, children, tone }: { title: string; children: ReactNode; tone?: "good" | "bad" }) {
  return (
    <div className={cx("rounded-md border border-dashed px-3 py-4 text-center", tone === "bad" ? "border-missing/50" : "border-line-2")}>
      <p className={cx("text-sm font-semibold", tone === "bad" ? "text-missing" : tone === "good" ? "text-accept" : "text-ink")}>{title}</p>
      <p className="mt-1 text-xs leading-relaxed text-ink-2">{children}</p>
    </div>
  );
}

function Legend({ findings }: { findings: Finding[] }) {
  const counts = new Map<Category, number>();
  for (const f of findings) counts.set(f.category, (counts.get(f.category) ?? 0) + 1);
  return (
    <ul className="flex flex-wrap gap-3 text-[11px]">
      {(Object.keys(CATEGORY_INFO) as Category[]).map((c) => (
        <li key={c} className={cx("inline-flex items-center gap-1", CATEGORY_INFO[c].text, !counts.get(c) && findings.length > 0 && "opacity-40")} title={CATEGORY_INFO[c].meaning}>
          <CategoryGlyph category={c} className="size-3.5" />
          {CATEGORY_INFO[c].label}
          {findings.length > 0 && <span className="font-mono">· {counts.get(c) ?? 0}</span>}
        </li>
      ))}
    </ul>
  );
}

function LayersView({ cmp, findings, selected, onSelect }: { cmp: ComparisonDetail; findings: Finding[]; selected: Finding | null; onSelect: (id: string) => void }) {
  const [spread, setSpread] = useState(1);
  const layers: [Layer, Layer] = [
    {
      key: "design",
      title: "Design",
      tone: "var(--color-design)",
      src: cmp.reference.contentUrl,
      widthPx: cmp.reference.widthPx,
      heightPx: cmp.reference.heightPx,
      boxes: findings.flatMap((f, i) =>
        f.referenceEvidence?.box
          ? [{ id: f.id, box: f.referenceEvidence.box, color: CATEGORY_INFO[f.category].color, selected: f.id === selected?.id, label: String(i + 1) }]
          : [],
      ),
    },
    {
      key: "build",
      title: "Build",
      tone: "var(--color-build)",
      src: cmp.actual.contentUrl,
      widthPx: cmp.actual.widthPx,
      heightPx: cmp.actual.heightPx,
      boxes: findings.flatMap((f, i) => {
        const own = f.actualEvidence?.box;
        const ghost = !own && f.category === "missing_element" ? f.referenceEvidence?.box : null;
        const box = own ?? ghost;
        return box
          ? [{ id: f.id, box, color: CATEGORY_INFO[f.category].color, ghost: !!ghost, selected: f.id === selected?.id, label: ghost ? `${i + 1} missing` : String(i + 1) }]
          : [];
      }),
    },
  ];
  const linkBox = selected?.referenceEvidence?.box ?? selected?.actualEvidence?.box ?? null;
  return (
    <div className="space-y-3">
      <div className="overflow-hidden px-4 pt-4 pb-2 sm:px-10">
        <Stack3D layers={layers} spread={spread} link={linkBox && selected ? { box: linkBox, color: CATEGORY_INFO[selected.category].color } : null} onSelect={onSelect} />
      </div>
      <label className="flex items-center gap-3 font-mono text-[11px] text-ink-3">
        <span>Overlay</span>
        <input type="range" min={0} max={1} step={0.01} value={spread} onChange={(e) => setSpread(Number(e.target.value))} className="flex-1 accent-neon" />
        <span>Exploded</span>
      </label>
      <p className="text-[11px] text-ink-3">
        {spread < 0.15
          ? "Overlay: the build is drawn at 50% over the design. Anything doubled or ghosted moved or changed."
          : "Drag the stack to turn it, double-click to reset. Select a finding to draw a post between its two positions."}
      </p>
    </div>
  );
}

function RunTicket({ run, runs, onPick, rerun }: { run: RunDetail; runs: ComparisonDetail["runs"]; onPick: (id: string) => void; rerun: ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-line-2 bg-sheet/80 px-4 py-2.5 text-xs text-ink-2">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <StatusBadge status={run.status} />
        <OriginBadge origin={run.origin} />
        {runs.length > 1 && (
          <select className="rounded-sm border border-line-2 bg-paper-2 px-1.5 py-0.5 font-mono text-[11px]" value={run.id} onChange={(e) => onPick(e.target.value)}>
            {runs.map((r, i) => (
              <option key={r.id} value={r.id}>
                run {runs.length - i} · {r.status} · {formatTime(r.createdAt)}
              </option>
            ))}
          </select>
        )}
        <span className="font-mono text-[11px]">
          {run.modelId} · prompt {run.promptVersion}
        </span>
        {run.finishedAt && <span className="font-mono text-[11px]">{(run.latencyMs / 1000).toFixed(1)} s</span>}
        {(run.inputTokens > 0 || run.outputTokens > 0) && (
          <span className="font-mono text-[11px]">
            {run.inputTokens.toLocaleString()} in / {run.outputTokens.toLocaleString()} out tokens
          </span>
        )}
        <span className="font-mono text-[11px] text-ink-3" title={`Input fingerprint ${run.inputFingerprint}`}>
          #{run.inputFingerprint.slice(0, 10)}
        </span>
        {rerun && <span className="ml-auto [&_button]:px-2.5 [&_button]:py-1 [&_button]:text-xs">{rerun}</span>}
      </div>
      {run.origin !== "meta_api" && (
        <p className="mt-1.5 font-medium text-missing">
          {run.origin === "mock"
            ? "Mock run: boxes come from a pixel-difference heuristic, not from a model. Do not present them as AI findings."
            : "Fixture run: canned answers from the demo's ground truth, for development. Not model output."}
        </p>
      )}
      {run.issues.length > 0 && (
        <ul className="mt-1.5 list-inside list-disc text-clipped">
          {run.issues.map((i) => (
            <li key={i}>{i}</li>
          ))}
        </ul>
      )}
    </div>
  );
}
