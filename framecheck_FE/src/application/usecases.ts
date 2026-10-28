import {
  CONTENT_STATE_PATTERN,
  isRunActive,
  pairProblem,
  reviewProblem,
  type Asset,
  type Capture,
  type Decision,
  type ImageRole,
  type RunDetail,
} from "../domain/types";
import { ApiError, type ExportFormat, type FrameCheckApi } from "./ports";

const MAX_IMAGE_BYTES = 10 * 1024 * 1024;
const POLL_INTERVAL_MS = 1000;

export function captureProblem(c: Capture): string | null {
  const intOk = (n: number) => Number.isInteger(n) && n >= 1 && n <= 10000;
  if (!intOk(c.viewportWidthCss) || !intOk(c.viewportHeightCss)) {
    return "Viewport width and height must be whole CSS pixels (1–10000).";
  }
  if (!(c.devicePixelRatio >= 0.5 && c.devicePixelRatio <= 5)) return "Device pixel ratio must be 0.5–5.";
  if (!CONTENT_STATE_PATTERN.test(c.contentStateKey)) {
    return "Content state: lowercase letters, digits, '.', '_' or '-' (e.g. default, logged-in).";
  }
  return null;
}

/** Use cases the UI calls. Each validates locally first for fast feedback; the backend re-validates. */
export function createUseCases(api: FrameCheckApi) {
  return {
    listProjects: () => api.listProjects(),
    getProject: (id: string) => api.getProject(id),
    getComparison: (id: string) => api.getComparison(id),
    getRun: (id: string) => api.getRun(id),

    async createProject(name: string) {
      const trimmed = name.trim();
      if (!trimmed) throw new ApiError(400, "INVALID_NAME", "Give the project a name.");
      return api.createProject(trimmed);
    },

    async uploadAsset(projectId: string, file: File, role: ImageRole, capture: Capture) {
      if (!["image/png", "image/jpeg"].includes(file.type)) {
        throw new ApiError(400, "INVALID_IMAGE", "Choose a PNG or JPEG image.");
      }
      if (file.size > MAX_IMAGE_BYTES) throw new ApiError(413, "IMAGE_TOO_LARGE", "Image must be at most 10 MB.");
      const problem = captureProblem(capture);
      if (problem) throw new ApiError(400, "INVALID_CAPTURE", problem);
      return api.uploadAsset(projectId, file, role, capture);
    },

    async createComparison(projectId: string, reference: Asset | undefined, actual: Asset | undefined, requirements: string) {
      const problem = pairProblem(reference, actual);
      if (problem || !reference || !actual) throw new ApiError(409, "CAPTURE_MISMATCH", problem ?? "Invalid pair.");
      return api.createComparison(projectId, reference.id, actual.id, requirements);
    },

    /** One key per user click: a double click or retry cannot start two runs. */
    startRun(comparisonId: string, allowDuplicate = false) {
      return api.startRun(comparisonId, crypto.randomUUID(), allowDuplicate);
    },

    /** Polls until the run leaves queued/running, or the signal aborts. */
    async waitForRun(runId: string, onUpdate: (r: RunDetail) => void, signal: AbortSignal) {
      for (;;) {
        const run = await api.getRun(runId);
        if (signal.aborted) return;
        onUpdate(run);
        if (!isRunActive(run.status)) return;
        await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
        if (signal.aborted) return;
      }
    },

    async reviewFinding(findingId: string, decision: Decision, reason: string) {
      const problem = reviewProblem(decision, reason);
      if (problem) throw new ApiError(400, "INVALID_REASON", problem);
      return api.reviewFinding(findingId, decision, reason.trim());
    },

    exportRun: (runId: string, format: ExportFormat) => api.exportRun(runId, format),
  };
}

export type UseCases = ReturnType<typeof createUseCases>;
