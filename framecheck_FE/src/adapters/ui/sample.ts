import type { Capture } from "../../domain/types";
import type { UseCases } from "../../application/usecases";

// Sample pairs from the self-owned demo page (demo/site). They let a first-time
// user see the whole flow without preparing screenshots.

export const SAMPLE_CAPTURE: Capture = { viewportWidthCss: 1280, viewportHeightCss: 800, devicePixelRatio: 1, contentStateKey: "default" };

export const SAMPLE_REFERENCE = "/samples/reference.png";

const SAMPLE_PAIRS = [
  {
    file: "actual-missing-hero-cta.png",
    requirements: 'The hero shows a primary "Start free trial" button under the headline.',
  },
  {
    file: "actual-clipped-hero-title.png",
    requirements: "The hero headline is fully readable on two lines.",
  },
  {
    file: "actual-layout-card-drop.png",
    requirements: "The three feature cards sit on one row with equal tops.",
  },
];

/** Where the sample's missing CTA sits in the 1280×800 reference, for the landing illustration. */
export const SAMPLE_MISSING_BOX = { x: 64, y: 376, width: 196, height: 52 };

async function fetchFile(path: string): Promise<File> {
  const res = await fetch(path);
  if (!res.ok) throw new Error(`Could not load sample image ${path} (${res.status}).`);
  const blob = await res.blob();
  return new File([blob], path.split("/").pop()!, { type: "image/png" });
}

export const SAMPLE_PROJECT_NAME = "Sample · Lembar landing page";

/** Opens the existing sample project, or creates one with one design and three builds, each paired into a comparison. */
export async function openOrCreateSampleProject(uc: UseCases): Promise<{ projectId: string; comparisonId: string }> {
  const existing = (await uc.listProjects()).find((p) => p.name === SAMPLE_PROJECT_NAME);
  if (existing) {
    const detail = await uc.getProject(existing.id);
    // Comparisons come newest first; the oldest is the missing-CTA pair.
    const first = detail.comparisons.at(-1);
    if (first) return { projectId: existing.id, comparisonId: first.id };
  }
  const project = await uc.createProject(SAMPLE_PROJECT_NAME);
  const reference = await uc.uploadAsset(project.id, await fetchFile(SAMPLE_REFERENCE), "reference", SAMPLE_CAPTURE);
  let first = "";
  for (const pair of SAMPLE_PAIRS) {
    const actual = await uc.uploadAsset(project.id, await fetchFile(`/samples/${pair.file}`), "actual", SAMPLE_CAPTURE);
    const cmp = await uc.createComparison(project.id, reference, actual, pair.requirements);
    first ||= cmp.id;
  }
  return { projectId: project.id, comparisonId: first };
}
