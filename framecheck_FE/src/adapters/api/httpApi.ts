// HTTP implementation of FrameCheckApi. The only module that knows backend routes and DTO shapes.
import { ApiError, type ExportFormat, type FrameCheckApi } from "../../application/ports";
import type {
  Asset,
  Comparison,
  ComparisonDetail,
  Evidence,
  Finding,
  Project,
  ProjectDetail,
  Review,
  Run,
  RunDetail,
} from "../../domain/types";

type Json = any;

async function request(path: string, init?: RequestInit): Promise<Response> {
  let res: Response;
  try {
    res = await fetch(path, init);
  } catch {
    throw new ApiError(0, "NETWORK", "Cannot reach the FrameCheck API. Is the backend running?");
  }
  if (!res.ok) {
    let code = "HTTP_" + res.status;
    let message = res.statusText || "Request failed";
    try {
      const body = await res.json();
      code = body?.error?.code ?? code;
      message = body?.error?.message ?? message;
    } catch {
      // non-JSON error body (e.g. proxy failure)
    }
    throw new ApiError(res.status, code, message);
  }
  return res;
}

async function json<T>(path: string, init?: RequestInit, map: (j: Json) => T = (j) => j): Promise<T> {
  const res = await request(path, init);
  return map(await res.json());
}

function postJson(body: unknown, headers: Record<string, string> = {}): RequestInit {
  return { method: "POST", headers: { "Content-Type": "application/json", ...headers }, body: JSON.stringify(body) };
}

const toProject = (j: Json): Project => ({ id: j.id, name: j.name, createdAt: j.created_at });

const toAsset = (j: Json): Asset => ({
  id: j.id,
  projectId: j.project_id,
  role: j.role,
  sha256: j.sha256,
  mime: j.mime,
  bytes: j.bytes,
  widthPx: j.width_px,
  heightPx: j.height_px,
  capture: {
    viewportWidthCss: j.viewport_width_css,
    viewportHeightCss: j.viewport_height_css,
    devicePixelRatio: j.device_pixel_ratio,
    contentStateKey: j.content_state_key,
  },
  contentUrl: j.content_url,
  createdAt: j.created_at,
});

const toComparison = (j: Json): Comparison => ({
  id: j.id,
  projectId: j.project_id,
  referenceAssetId: j.reference_asset_id,
  actualAssetId: j.actual_asset_id,
  requirementVersionId: j.requirement_version_id,
  createdAt: j.created_at,
});

const toRun = (j: Json): Run => ({
  id: j.id,
  comparisonId: j.comparison_id,
  status: j.status,
  origin: j.origin,
  inputFingerprint: j.input_fingerprint,
  modelId: j.model_id,
  configVersion: j.config_version,
  promptVersion: j.prompt_version,
  providerRequestId: j.provider_request_id || undefined,
  latencyMs: j.latency_ms,
  inputTokens: j.input_tokens ?? 0,
  outputTokens: j.output_tokens ?? 0,
  failureCode: j.failure_code || undefined,
  failureMessage: j.failure_message || undefined,
  issues: j.issues ?? [],
  createdAt: j.created_at,
  startedAt: j.started_at,
  finishedAt: j.finished_at,
});

const toComparisonDetail = (j: Json): ComparisonDetail => ({
  ...toComparison(j),
  reference: toAsset(j.reference),
  actual: toAsset(j.actual),
  requirement: { id: j.requirement.id, version: j.requirement.version, text: j.requirement.text, sha256: j.requirement.sha256 },
  runs: (j.runs ?? []).map(toRun),
});

const toEvidence = (j: Json): Evidence | null =>
  j ? { assetId: j.asset_id, imageRole: j.image_role, box: j.box_px ?? null, description: j.description } : null;

const toReview = (j: Json): Review => ({
  id: j.id,
  findingId: j.finding_id,
  decision: j.decision,
  reason: j.reason,
  reviewedAt: j.reviewed_at,
});

const toFinding = (j: Json): Finding => ({
  id: j.id,
  runId: j.run_id,
  category: j.category,
  summary: j.summary,
  checkStep: j.check_step,
  referenceEvidence: toEvidence(j.reference_evidence),
  actualEvidence: toEvidence(j.actual_evidence),
  evidenceStatus: j.evidence_status,
  issues: j.issues ?? [],
  currentReview: j.current_review ? toReview(j.current_review) : null,
  reviewHistory: (j.review_history ?? []).map(toReview),
});

const enc = encodeURIComponent;

export function createHttpApi(): FrameCheckApi {
  return {
    listProjects: () => json("/v1/projects", undefined, (j) => j.projects.map(toProject)),
    createProject: (name) => json("/v1/projects", postJson({ name }), toProject),
    getProject: (id) =>
      json(`/v1/projects/${enc(id)}`, undefined, (j): ProjectDetail => ({
        ...toProject(j),
        assets: j.assets.map(toAsset),
        comparisons: j.comparisons.map(toComparison),
      })),
    uploadAsset(projectId, file, role, capture) {
      const form = new FormData();
      form.set("role", role);
      form.set("viewport_width_css", String(capture.viewportWidthCss));
      form.set("viewport_height_css", String(capture.viewportHeightCss));
      form.set("device_pixel_ratio", String(capture.devicePixelRatio));
      form.set("content_state_key", capture.contentStateKey);
      form.set("file", file);
      return json(`/v1/projects/${enc(projectId)}/assets`, { method: "POST", body: form }, toAsset);
    },
    createComparison: (projectId, referenceAssetId, actualAssetId, requirements) =>
      json(
        `/v1/projects/${enc(projectId)}/comparisons`,
        postJson({ reference_asset_id: referenceAssetId, actual_asset_id: actualAssetId, requirements }),
        toComparisonDetail,
      ),
    getComparison: (id) => json(`/v1/comparisons/${enc(id)}`, undefined, toComparisonDetail),
    startRun: (comparisonId, idempotencyKey, allowDuplicate) =>
      json(
        `/v1/comparisons/${enc(comparisonId)}/runs`,
        postJson({ allow_duplicate: allowDuplicate }, { "Idempotency-Key": idempotencyKey }),
        toRun,
      ),
    getRun: (id) =>
      json(`/v1/runs/${enc(id)}`, undefined, (j): RunDetail => ({ ...toRun(j), findings: j.findings.map(toFinding) })),
    reviewFinding: (findingId, decision, reason) =>
      json(`/v1/findings/${enc(findingId)}/reviews`, postJson({ decision, reason }), toReview),
    async exportRun(runId, format: ExportFormat) {
      const res = await request(`/v1/runs/${enc(runId)}/export?format=${format}`);
      const ext = format === "markdown" ? "md" : "json";
      return { filename: `framecheck-run-${runId}.${ext}`, blob: await res.blob() };
    },
  };
}
