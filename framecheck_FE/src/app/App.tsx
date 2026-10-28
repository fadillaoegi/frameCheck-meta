import { useEffect, useState } from "react";
import { createHttpApi } from "../adapters/api/httpApi";
import { createUseCases } from "../application/usecases";
import { UseCasesContext } from "../adapters/ui/context";
import { href, parseHash } from "../adapters/ui/routes";
import { ProjectsPage } from "../adapters/ui/pages/ProjectsPage";
import { ProjectPage } from "../adapters/ui/pages/ProjectPage";
import { ComparisonPage } from "../adapters/ui/pages/ComparisonPage";

// Composition root: the only place that chooses concrete adapters.
const useCases = createUseCases(createHttpApi());

export function App() {
  const [route, setRoute] = useState(() => parseHash(window.location.hash));
  useEffect(() => {
    const onChange = () => {
      setRoute(parseHash(window.location.hash));
      window.scrollTo({ top: 0 });
    };
    window.addEventListener("hashchange", onChange);
    return () => window.removeEventListener("hashchange", onChange);
  }, []);

  return (
    <UseCasesContext.Provider value={useCases}>
      <div className="mat min-h-screen text-ink">
        <header className="sticky top-0 z-30 border-b border-neon/20 bg-paper/85 shadow-[0_1px_24px_-12px_var(--color-neon)] backdrop-blur">
          <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-2.5">
            <a href={href.projects()} className="flex items-center gap-2">
              <Logo />
              <span className="font-display text-2xl leading-none font-semibold tracking-wide">
                Frame<span className="text-neon text-glow">Check</span>
              </span>
            </a>
            <p className="hidden font-mono text-[11px] text-ink-3 sm:block">design ↔ build · every finding needs a human decision</p>
          </div>
        </header>
        <main className="mx-auto max-w-7xl px-4 py-6 sm:py-8">
          {route.page === "projects" && <ProjectsPage />}
          {route.page === "project" && <ProjectPage key={route.projectId} projectId={route.projectId} />}
          {route.page === "comparison" && <ComparisonPage key={route.comparisonId} comparisonId={route.comparisonId} runId={route.runId} />}
        </main>
        <footer className="mx-auto max-w-7xl px-4 pb-8 font-mono text-[11px] text-ink-3">
          FrameCheck · every run shows where its findings came from: Meta Muse Spark, or a labelled fixture/mock used for development.
        </footer>
      </div>
    </UseCasesContext.Provider>
  );
}

/** The FrameCheck mark: the build as a frame over the design plate, checked at one registration point. See docs/BRAND.md. */
function Logo() {
  return <img src="/brand/framecheck-mark-neon.svg" alt="" className="size-9" />;
}
