import { useEffect, useMemo, useState, type ChangeEvent, type FormEvent } from "react";
import { capturesMatch, pairProblem, type Asset, type Capture, type ImageRole } from "../../../domain/types";
import { captureProblem } from "../../../application/usecases";
import { useUseCases } from "../context";
import { href, navigate } from "../routes";
import { Button, Callout, Card, CropMarks, ErrorBanner, Field, Steps, cx, formatBytes, formatTime, inputClass, useLoad } from "../components/ui";

const defaultCapture: Capture = { viewportWidthCss: 1280, viewportHeightCss: 800, devicePixelRatio: 2, contentStateKey: "default" };

const REQUIREMENT_EXAMPLES = ["The primary CTA is visible in the hero.", "Card titles fit on one line.", "The logo sits at the top left."];

const ROLE_UI: Record<ImageRole, { tag: string; tone: string; title: string; help: string }> = {
  reference: { tag: "Design", tone: "bg-design", title: "Design (reference)", help: "Export from Figma or a mockup — what it should look like." },
  actual: { tag: "Build", tone: "bg-build", title: "Build (screenshot)", help: "A screenshot of the running page — what was implemented." },
};

export function ProjectPage({ projectId }: { projectId: string }) {
  const uc = useUseCases();
  const { data: project, error, reload } = useLoad(() => uc.getProject(projectId), [uc, projectId]);

  const [refId, setRefId] = useState("");
  const [actId, setActId] = useState("");
  const [requirements, setRequirements] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<unknown>(null);

  const assets = project?.assets ?? [];
  const references = assets.filter((a) => a.role === "reference");
  const actuals = assets.filter((a) => a.role === "actual");
  const reference = references.find((a) => a.id === refId) ?? references.at(-1);
  // Default to the newest actual image captured under the same conditions.
  const actual =
    actuals.find((a) => a.id === actId) ??
    actuals.findLast((a) => reference !== undefined && capturesMatch(a.capture, reference.capture)) ??
    actuals.at(-1);
  const problem = pairProblem(reference, actual);
  const byId = new Map(assets.map((a) => [a.id, a]));

  async function createComparison(e: FormEvent) {
    e.preventDefault();
    setCreating(true);
    setCreateError(null);
    try {
      const c = await uc.createComparison(projectId, reference, actual, requirements);
      navigate(href.comparison(c.id));
    } catch (err) {
      setCreateError(err);
    } finally {
      setCreating(false);
    }
  }

  if (error) return <ErrorBanner error={error} />;
  if (!project) return <p className="text-sm text-ink-3">Loading…</p>;

  const hasBoth = references.length > 0 && actuals.length > 0;

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <a href={href.projects()} className="font-mono text-[11px] text-ink-3 hover:text-ink">
          ← All projects
        </a>
        <div className="flex flex-wrap items-end justify-between gap-3">
          <h1 className="font-display text-4xl leading-none font-semibold">{project.name}</h1>
          <Steps current={hasBoth ? "Pair" : "Upload"} done={hasBoth ? ["Upload"] : []} />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <UploadCard projectId={projectId} onUploaded={reload} />

        <Card kicker="Step 2" title="Pair a design with a build">
          <form onSubmit={createComparison} className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-2">
              <PairSlot role="reference" assets={references} selected={reference} onChange={setRefId} />
              <PairSlot role="actual" assets={actuals} selected={actual} onChange={setActId} />
            </div>
            <Field
              label="What matters on this page? (optional)"
              hint="Short notes in English. They guide the analysis but cannot change its rules or output."
            >
              <textarea className={cx(inputClass, "min-h-20")} maxLength={4000} value={requirements} onChange={(e) => setRequirements(e.target.value)} />
            </Field>
            <div className="flex flex-wrap gap-1.5">
              {REQUIREMENT_EXAMPLES.map((ex) => (
                <button
                  key={ex}
                  type="button"
                  onClick={() => setRequirements((r) => (r.trim() ? `${r.trim()}\n${ex}` : ex))}
                  className="rounded-full border border-dashed border-line-2 px-2.5 py-0.5 text-[11px] text-ink-2 hover:border-neon/60 hover:text-neon"
                >
                  + {ex}
                </button>
              ))}
            </div>
            {problem && assets.length > 0 && <Callout tone="warn">{problem}</Callout>}
            {!hasBoth && <Callout>Upload at least one design and one build first. Both slots fill in automatically.</Callout>}
            <ErrorBanner error={createError} />
            <Button type="submit" disabled={creating || problem !== null}>
              {creating ? "Creating…" : "Create comparison →"}
            </Button>
          </form>
        </Card>
      </div>

      <Card kicker="Step 3 onward" title={`Comparisons (${project.comparisons.length})`}>
        {project.comparisons.length === 0 ? (
          <p className="text-sm text-ink-3">No comparisons yet. Pair a design with a build above; the analysis runs on the next screen.</p>
        ) : (
          <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {project.comparisons
              .slice()
              .reverse()
              .map((c) => {
                const r = byId.get(c.referenceAssetId);
                const a = byId.get(c.actualAssetId);
                return (
                  <li key={c.id}>
                    <a href={href.comparison(c.id)} className="group block rounded-md border border-line bg-paper-2 p-2 transition-all hover:-translate-y-0.5 hover:border-neon/50 hover:shadow-[0_0_24px_-10px_var(--color-neon)]">
                      <div className="grid grid-cols-2 gap-1.5">
                        {[r, a].map((img, i) => (
                          <div key={i} className="relative overflow-hidden rounded-sm ring-1 ring-ink/10">
                            {img && <img src={img.contentUrl} alt="" className="aspect-[16/10] w-full object-cover object-top" loading="lazy" />}
                            <span className={cx("absolute bottom-1 left-1 rounded-sm px-1 font-mono text-[9px] text-paper uppercase", i === 0 ? "bg-design" : "bg-build")}>
                              {i === 0 ? "Design" : "Build"}
                            </span>
                          </div>
                        ))}
                      </div>
                      <div className="mt-2 flex items-center justify-between font-mono text-[11px] text-ink-3">
                        <span className="group-hover:text-ink">Open {c.id.slice(0, 8)} →</span>
                        <span>{formatTime(c.createdAt)}</span>
                      </div>
                    </a>
                  </li>
                );
              })}
          </ul>
        )}
      </Card>

      <Card kicker="Library" title={`Uploaded images (${assets.length})`}>
        {assets.length === 0 ? (
          <p className="text-sm text-ink-3">Nothing uploaded yet.</p>
        ) : (
          <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {assets.map((a) => (
              <li key={a.id} className="overflow-hidden rounded-md border border-line bg-paper-2">
                <img src={a.contentUrl} alt={`${a.role} image`} className="checker aspect-video w-full object-contain" loading="lazy" />
                <div className="space-y-0.5 p-2 text-xs">
                  <div className="flex items-center justify-between">
                    <span className={cx("rounded-sm px-1 font-mono text-[10px] text-paper uppercase", ROLE_UI[a.role].tone)}>{ROLE_UI[a.role].tag}</span>
                    <span className="font-mono text-[11px] text-ink-3">{formatBytes(a.bytes)}</span>
                  </div>
                  <div className="font-mono text-[11px] text-ink-2">
                    {a.widthPx}×{a.heightPx}px · {a.capture.viewportWidthCss}×{a.capture.viewportHeightCss} @{a.capture.devicePixelRatio}x
                  </div>
                  <div className="font-mono text-[11px] text-ink-3">state: {a.capture.contentStateKey}</div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}

function PairSlot({
  role,
  assets,
  selected,
  onChange,
}: {
  role: ImageRole;
  assets: Asset[];
  selected: Asset | undefined;
  onChange: (id: string) => void;
}) {
  const ui = ROLE_UI[role];
  return (
    <div className="min-w-0 space-y-1.5">
      <span className="text-xs font-medium text-ink-2">{ui.title}</span>
      <div className="relative overflow-hidden rounded-sm bg-paper ring-1 ring-line">
        {selected ? (
          <img src={selected.contentUrl} alt="" className="aspect-[16/10] w-full object-cover object-top" />
        ) : (
          <div className="grid aspect-[16/10] place-items-center text-[11px] text-ink-3">none uploaded</div>
        )}
        <span className={cx("absolute top-1 left-1 rounded-sm px-1 font-mono text-[9px] text-paper uppercase", ui.tone)}>{ui.tag}</span>
      </div>
      <select className={cx(inputClass, "text-xs")} value={selected?.id ?? ""} onChange={(e) => onChange(e.target.value)} disabled={assets.length === 0}>
        {assets.length === 0 && <option value="">—</option>}
        {assets.map((a) => (
          <option key={a.id} value={a.id}>
            {a.sha256.slice(0, 8)} · {a.widthPx}×{a.heightPx} @{a.capture.devicePixelRatio}x · {a.capture.contentStateKey}
          </option>
        ))}
      </select>
    </div>
  );
}

/** Reads the pixel size of a chosen file so we can explain and suggest capture settings. */
function useImageSize(file: File | null) {
  const [size, setSize] = useState<{ w: number; h: number; url: string } | null>(null);
  useEffect(() => {
    if (!file) {
      setSize(null);
      return;
    }
    const url = URL.createObjectURL(file);
    const img = new Image();
    img.onload = () => setSize({ w: img.naturalWidth, h: img.naturalHeight, url });
    img.src = url;
    return () => URL.revokeObjectURL(url);
  }, [file]);
  return size;
}

function UploadCard({ projectId, onUploaded }: { projectId: string; onUploaded: () => void }) {
  const uc = useUseCases();
  const [capture, setCapture] = useState(defaultCapture);
  const [files, setFiles] = useState<Record<ImageRole, File | null>>({ reference: null, actual: null });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const problem = useMemo(() => captureProblem(capture), [capture]);
  const refSize = useImageSize(files.reference);
  const actSize = useImageSize(files.actual);

  const anySize = refSize ?? actSize;
  // Image width ÷ viewport width is the pixel ratio the screenshot was taken at.
  const suggestedDpr = anySize && capture.viewportWidthCss > 0 ? Math.round((anySize.w / capture.viewportWidthCss) * 4) / 4 : null;
  const sizesDiffer = refSize && actSize && (refSize.w !== actSize.w || refSize.h !== actSize.h);

  const num = (key: keyof Capture) => (e: ChangeEvent<HTMLInputElement>) =>
    setCapture({ ...capture, [key]: e.target.value === "" ? NaN : Number(e.target.value) });

  async function upload(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      for (const role of ["reference", "actual"] as const) {
        const file = files[role];
        if (file) await uc.uploadAsset(projectId, file, role, capture);
      }
      setFiles({ reference: null, actual: null });
      (e.target as HTMLFormElement).reset();
    } catch (err) {
      setError(err);
    } finally {
      setBusy(false);
      // Refresh even after a failure: the first file may already be stored.
      onUploaded();
    }
  }

  return (
    <Card kicker="Step 1" title="Upload a design and a build">
      <form onSubmit={upload} className="space-y-4">
        <div className="grid gap-3 sm:grid-cols-2">
          {(["reference", "actual"] as const).map((role) => {
            const size = role === "reference" ? refSize : actSize;
            const ui = ROLE_UI[role];
            return (
              <label key={role} className="group relative block cursor-pointer">
                <CropMarks />
                <div
                  className={cx(
                    "relative grid aspect-[16/10] place-items-center overflow-hidden rounded-sm border-2 border-dashed bg-paper-2/60 transition-colors",
                    size ? "border-transparent" : "border-line-2 group-hover:border-neon/60",
                  )}
                >
                  {size ? (
                    <img src={size.url} alt="" className="absolute inset-0 h-full w-full object-cover object-top" />
                  ) : (
                    <div className="px-3 text-center">
                      <p className="text-sm font-medium">{ui.title}</p>
                      <p className="mt-1 text-[11px] leading-snug text-ink-3">{ui.help}</p>
                      <p className="mt-2 font-mono text-[10px] text-ink-3">click to choose · PNG/JPEG ≤ 10 MB</p>
                    </div>
                  )}
                  <span className={cx("absolute top-1.5 left-1.5 rounded-sm px-1 font-mono text-[9px] text-paper uppercase", ui.tone)}>{ui.tag}</span>
                  {size && <span className="absolute right-1.5 bottom-1.5 rounded-sm bg-paper/80 px-1 font-mono text-[10px] text-ink">{size.w}×{size.h}px</span>}
                </div>
                <input
                  type="file"
                  accept="image/png,image/jpeg"
                  className="sr-only"
                  onChange={(e) => setFiles({ ...files, [role]: e.target.files?.[0] ?? null })}
                />
              </label>
            );
          })}
        </div>

        <fieldset className="space-y-3 rounded-md border border-line bg-paper/60 p-3">
          <legend className="px-1 text-xs font-semibold text-ink">How were they captured?</legend>
          <p className="text-[11px] leading-snug text-ink-3">
            Both images must come from the same browser size, screen density and page state — otherwise everything looks different.
          </p>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Field label="Viewport width" hint="CSS px">
              <input type="number" className={inputClass} value={capture.viewportWidthCss} onChange={num("viewportWidthCss")} />
            </Field>
            <Field label="Viewport height" hint="CSS px">
              <input type="number" className={inputClass} value={capture.viewportHeightCss} onChange={num("viewportHeightCss")} />
            </Field>
            <Field label="Pixel ratio" hint="2 on Retina">
              <input type="number" step="0.25" className={inputClass} value={capture.devicePixelRatio} onChange={num("devicePixelRatio")} />
            </Field>
            <Field label="Page state" hint="e.g. logged-in">
              <input className={inputClass} value={capture.contentStateKey} onChange={(e) => setCapture({ ...capture, contentStateKey: e.target.value })} />
            </Field>
          </div>
          {suggestedDpr !== null && suggestedDpr !== capture.devicePixelRatio && suggestedDpr >= 0.5 && suggestedDpr <= 5 && (
            <Callout>
              The image is {anySize!.w}px wide. At a {capture.viewportWidthCss}px viewport that means pixel ratio <b>{suggestedDpr}</b>.{" "}
              <button type="button" className="font-semibold text-design underline" onClick={() => setCapture({ ...capture, devicePixelRatio: suggestedDpr })}>
                Use {suggestedDpr}
              </button>
            </Callout>
          )}
          {sizesDiffer && (
            <Callout tone="warn">
              The two images have different sizes ({refSize!.w}×{refSize!.h} vs {actSize!.w}×{actSize!.h}). They can still be uploaded, but they cannot be
              paired until both are captured the same way.
            </Callout>
          )}
        </fieldset>

        {problem && <Callout tone="warn">{problem}</Callout>}
        <ErrorBanner error={error} />
        <Button type="submit" disabled={busy || problem !== null || (!files.reference && !files.actual)}>
          {busy ? "Uploading…" : "Upload"}
        </Button>
      </form>
    </Card>
  );
}
