import { useState, type FormEvent } from "react";
import type { Category } from "../../../domain/types";
import { useUseCases } from "../context";
import { href, navigate } from "../routes";
import { SAMPLE_MISSING_BOX, SAMPLE_REFERENCE, openOrCreateSampleProject } from "../sample";
import { Stack3D } from "../components/Stack3D";
import { Button, CATEGORY_INFO, CategoryGlyph, Card, ErrorBanner, formatTime, inputClass, useLoad } from "../components/ui";

const HOW = [
  {
    title: "Capture both at the same size",
    text: "Take the design export and a screenshot of the build at the same viewport and pixel ratio, in the same content state. Otherwise every pixel would differ.",
  },
  { title: "Upload and pair", text: "Upload the design and the build into a project, then pair them. Add a few notes on what matters, like “the CTA is visible”." },
  {
    title: "Analyse",
    text: "Meta Muse Spark compares the pair and returns findings in three categories, each with a box on the design, the build, or both.",
  },
  { title: "Review every finding", text: "Nothing counts until a person decides. Accept real defects; reject false alarms with a reason." },
  { title: "Export", text: "Download the reviewed findings as Markdown for a PR or ticket, or JSON for your own tools." },
];

export function ProjectsPage() {
  const uc = useUseCases();
  const { data: projects, error } = useLoad(() => uc.listProjects(), [uc]);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [sampleBusy, setSampleBusy] = useState(false);
  const [actionError, setActionError] = useState<unknown>(null);

  async function create(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setActionError(null);
    try {
      const p = await uc.createProject(name);
      navigate(href.project(p.id));
    } catch (err) {
      setActionError(err);
    } finally {
      setBusy(false);
    }
  }

  async function trySample() {
    setSampleBusy(true);
    setActionError(null);
    try {
      const s = await openOrCreateSampleProject(uc);
      navigate(href.comparison(s.comparisonId));
    } catch (err) {
      setActionError(err);
    } finally {
      setSampleBusy(false);
    }
  }

  return (
    <div className="space-y-14">
      {/* Hero */}
      <section className="grid items-center gap-10 lg:grid-cols-[minmax(0,5fr)_minmax(0,6fr)]">
        <div className="space-y-5">
          <p className="font-mono text-[11px] tracking-widest text-ink-3 uppercase">Visual QA for front-end builds</p>
          <h1 className="font-display text-5xl leading-[0.95] font-semibold sm:text-6xl">
            Does the build <em className="flicker text-neon-2 text-glow">match</em> the design?
          </h1>
          <p className="max-w-md text-base leading-relaxed text-ink-2">
            FrameCheck lays your implementation screenshot over the design and points at what went wrong: elements that went missing, text that got cut
            off, and things that moved. Every claim comes with a box you can check, and you decide what is real.
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <Button onClick={trySample} disabled={sampleBusy}>
              {sampleBusy ? "Preparing sample…" : "Try it with a sample →"}
            </Button>
            <a href="#new-project" className="text-sm font-medium text-ink-2 underline decoration-line-2 underline-offset-4 hover:text-ink">
              or start your own project
            </a>
          </div>
          <p className="font-mono text-[11px] text-ink-3">The sample loads one design and three builds of a demo page. Nothing is analysed until you press Run.</p>
          <ErrorBanner error={actionError} onDismiss={() => setActionError(null)} />
        </div>

        <div className="relative px-2 pt-2 pb-6 sm:px-6">
          <Stack3D
            spread={1}
            tilt="hover"
            link={{ box: SAMPLE_MISSING_BOX, color: CATEGORY_INFO.missing_element.color }}
            layers={[
              {
                key: "design",
                title: "Design",
                tone: "var(--color-design)",
                src: SAMPLE_REFERENCE,
                widthPx: 1280,
                heightPx: 800,
                boxes: [{ id: "cta", box: SAMPLE_MISSING_BOX, color: CATEGORY_INFO.missing_element.color, selected: true }],
              },
              {
                key: "build",
                title: "Build",
                tone: "var(--color-build)",
                src: "/samples/actual-missing-hero-cta.png",
                widthPx: 1280,
                heightPx: 800,
                boxes: [{ id: "cta", box: SAMPLE_MISSING_BOX, color: CATEGORY_INFO.missing_element.color, ghost: true, label: "button missing" }],
              },
            ]}
          />
          <p className="mt-2 text-center font-mono text-[11px] text-ink-3">
            The “Start free trial” button is in the design layer but not in the build. The post links the two places.
          </p>
        </div>
      </section>

      {/* How it works */}
      <section className="space-y-4">
        <SectionTitle kicker="How it works" title="Five steps, one screen each" />
        <ol className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          {HOW.map((s, i) => (
            <li key={s.title} className="sheet-shadow relative rounded-lg border border-line bg-sheet p-4">
              <span className="font-display text-4xl leading-none font-semibold text-neon/40">{String(i + 1).padStart(2, "0")}</span>
              <h3 className="mt-2 text-sm font-semibold">{s.title}</h3>
              <p className="mt-1 text-xs leading-relaxed text-ink-2">{s.text}</p>
            </li>
          ))}
        </ol>
      </section>

      {/* Categories */}
      <section className="space-y-4">
        <SectionTitle kicker="What it looks for" title="Three kinds of defect, each with a different colour" />
        <div className="grid gap-3 md:grid-cols-3">
          {(Object.keys(CATEGORY_INFO) as Category[]).map((c) => {
            const info = CATEGORY_INFO[c];
            return (
              <div key={c} className="sheet-shadow overflow-hidden rounded-lg border border-line bg-sheet">
                <div className="h-1" style={{ background: info.color, boxShadow: `0 0 16px ${info.color}` }} />
                <div className="space-y-2 p-4">
                  <h3 className={`flex items-center gap-2 text-sm font-semibold ${info.text}`}>
                    <CategoryGlyph category={c} />
                    {info.label}
                  </h3>
                  <p className="text-sm text-ink">{info.meaning}</p>
                  <p className="text-xs text-ink-3">{info.boxes}</p>
                </div>
              </div>
            );
          })}
        </div>
        <p className="max-w-3xl text-xs leading-relaxed text-ink-3">
          Not checked: colour, font rendering, or wording on their own. An empty result means nothing was found in these three categories. It is not a
          sign-off.
        </p>
      </section>

      {/* Projects */}
      <section id="new-project" className="grid scroll-mt-20 grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
        <Card kicker="Your work" title="Projects">
          <ErrorBanner error={error} />
          {projects === null && !error && <p className="text-sm text-ink-3">Loading…</p>}
          {projects?.length === 0 && <p className="text-sm text-ink-3">No projects yet. Try the sample above, or create one on the right.</p>}
          <ul className="divide-y divide-line/70">
            {projects?.map((p) => (
              <li key={p.id}>
                <a href={href.project(p.id)} className="group flex items-center justify-between gap-3 py-2.5">
                  <span className="font-medium group-hover:underline group-hover:underline-offset-4">{p.name}</span>
                  <span className="shrink-0 font-mono text-[11px] text-ink-3">{formatTime(p.createdAt)}</span>
                </a>
              </li>
            ))}
          </ul>
        </Card>

        <Card kicker="Step 1" title="New project">
          <form onSubmit={create} className="space-y-3">
            <input className={inputClass} placeholder="e.g. Checkout page v2" value={name} maxLength={120} onChange={(e) => setName(e.target.value)} />
            <Button type="submit" disabled={busy || !name.trim()} className="w-full">
              {busy ? "Creating…" : "Create project"}
            </Button>
          </form>
          <p className="mt-3 text-xs leading-relaxed text-ink-3">A project holds the designs and builds of one page or screen. Next you upload images.</p>
        </Card>
      </section>
    </div>
  );
}

function SectionTitle({ kicker, title }: { kicker: string; title: string }) {
  return (
    <div>
      <p className="font-mono text-[11px] tracking-widest text-ink-3 uppercase">{kicker}</p>
      <h2 className="font-display text-3xl leading-tight font-semibold">{title}</h2>
    </div>
  );
}
