// Pure FrameCheck entities and rules. No React, fetch, or browser storage.

export type ImageRole = "reference" | "actual";
export type Category = "missing_element" | "clipped_text" | "layout_mismatch";
export type RunStatus = "queued" | "running" | "completed" | "needs_review" | "failed";
export type RunOrigin = "meta_api" | "fixture" | "mock";
export type EvidenceStatus = "validated" | "needs_review";
export type Decision = "accepted" | "rejected";

export interface Project {
  id: string;
  name: string;
  createdAt: string;
}

export interface Capture {
  viewportWidthCss: number;
  viewportHeightCss: number;
  devicePixelRatio: number;
  contentStateKey: string;
}

export interface Asset {
  id: string;
  projectId: string;
  role: ImageRole;
  sha256: string;
  mime: string;
  bytes: number;
  widthPx: number;
  heightPx: number;
  capture: Capture;
  contentUrl: string;
  createdAt: string;
}

export interface RequirementVersion {
  id: string;
  version: number;
  text: string;
  sha256: string;
}

export interface Comparison {
  id: string;
  projectId: string;
  referenceAssetId: string;
  actualAssetId: string;
  requirementVersionId: string;
  createdAt: string;
}

export interface ProjectDetail extends Project {
  assets: Asset[];
  comparisons: Comparison[];
}

export interface Run {
  id: string;
  comparisonId: string;
  status: RunStatus;
  origin: RunOrigin;
  inputFingerprint: string;
  modelId: string;
  configVersion: string;
  promptVersion: string;
  providerRequestId?: string;
  latencyMs: number;
  inputTokens: number;
  outputTokens: number;
  failureCode?: string;
  failureMessage?: string;
  issues: string[];
  createdAt: string;
  startedAt: string | null;
  finishedAt: string | null;
}

export interface ComparisonDetail extends Comparison {
  reference: Asset;
  actual: Asset;
  requirement: RequirementVersion;
  runs: Run[];
}

/** Rectangle in original raster pixels, origin top left. */
export interface Box {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Evidence {
  assetId: string;
  imageRole: ImageRole;
  box: Box | null;
  description: string;
}

export interface Review {
  id: string;
  findingId: string;
  decision: Decision;
  reason: string;
  reviewedAt: string;
}

export interface Finding {
  id: string;
  runId: string;
  category: Category;
  summary: string;
  checkStep: string;
  referenceEvidence: Evidence | null;
  actualEvidence: Evidence | null;
  evidenceStatus: EvidenceStatus;
  issues: string[];
  currentReview: Review | null;
  reviewHistory: Review[];
}

export interface RunDetail extends Run {
  findings: Finding[];
}

export const CATEGORY_LABEL: Record<Category, string> = {
  missing_element: "Missing element",
  clipped_text: "Clipped text",
  layout_mismatch: "Layout mismatch",
};

export function isRunActive(status: RunStatus): boolean {
  return status === "queued" || status === "running";
}

export function isRunFinished(status: RunStatus): boolean {
  return status === "completed" || status === "needs_review";
}

export function capturesMatch(a: Capture, b: Capture): boolean {
  return (
    a.viewportWidthCss === b.viewportWidthCss &&
    a.viewportHeightCss === b.viewportHeightCss &&
    Math.abs(a.devicePixelRatio - b.devicePixelRatio) < 1e-9 &&
    a.contentStateKey === b.contentStateKey
  );
}

/** Explains why a reference/actual pair cannot be compared, or null if it can. */
export function pairProblem(reference: Asset | undefined, actual: Asset | undefined): string | null {
  if (!reference || !actual) return "Choose one reference and one actual image.";
  if (reference.role !== "reference" || actual.role !== "actual") return "Image roles are swapped.";
  if (!capturesMatch(reference.capture, actual.capture)) {
    return "Viewport size, device pixel ratio, and content state must match.";
  }
  return null;
}

/**
 * Converts an original-pixel box to percentages of the image, so overlays
 * stay aligned however large the image is displayed.
 */
export function boxToPercent(box: Box, widthPx: number, heightPx: number) {
  return {
    left: (box.x / widthPx) * 100,
    top: (box.y / heightPx) * 100,
    width: (box.width / widthPx) * 100,
    height: (box.height / heightPx) * 100,
  };
}

/**
 * Region to show around an evidence box: the box plus context padding,
 * clamped to the image so the crop never shows space outside it.
 */
export function cropWindow(box: Box, widthPx: number, heightPx: number, pad = 0.35, minPad = 16): Box {
  const px = Math.max(minPad, Math.round(box.width * pad));
  const py = Math.max(minPad, Math.round(box.height * pad));
  const x = Math.max(0, box.x - px);
  const y = Math.max(0, box.y - py);
  const x1 = Math.min(widthPx, box.x + box.width + px);
  const y1 = Math.min(heightPx, box.y + box.height + py);
  return { x, y, width: x1 - x, height: y1 - y };
}

/** Smallest box covering every given box. */
export function unionBox(boxes: Box[]): Box | null {
  if (boxes.length === 0) return null;
  const x = Math.min(...boxes.map((b) => b.x));
  const y = Math.min(...boxes.map((b) => b.y));
  const x1 = Math.max(...boxes.map((b) => b.x + b.width));
  const y1 = Math.max(...boxes.map((b) => b.y + b.height));
  return { x, y, width: x1 - x, height: y1 - y };
}

export interface ReviewTally {
  total: number;
  accepted: number;
  rejected: number;
  unreviewed: number;
}

export function tallyReviews(findings: Finding[]): ReviewTally {
  const t = { total: findings.length, accepted: 0, rejected: 0, unreviewed: 0 };
  for (const f of findings) {
    if (!f.currentReview) t.unreviewed++;
    else if (f.currentReview.decision === "accepted") t.accepted++;
    else t.rejected++;
  }
  return t;
}

/** Mirrors the backend rule: a rejection needs a reason. */
export function reviewProblem(decision: Decision, reason: string): string | null {
  if (decision === "rejected" && reason.trim() === "") return "Give a reason for rejecting.";
  if (reason.length > 1000) return "Reason must be at most 1000 characters.";
  return null;
}

export const CONTENT_STATE_PATTERN = /^[a-z0-9][a-z0-9._-]{0,63}$/;
