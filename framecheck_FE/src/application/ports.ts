import type {
  Capture,
  ComparisonDetail,
  Decision,
  ImageRole,
  Asset,
  Project,
  ProjectDetail,
  Review,
  Run,
  RunDetail,
} from "../domain/types";

export type ExportFormat = "json" | "markdown";

export interface ExportFile {
  filename: string;
  blob: Blob;
}

/** Errors surfaced to the UI carry the backend's stable code. */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export interface FrameCheckApi {
  listProjects(): Promise<Project[]>;
  createProject(name: string): Promise<Project>;
  getProject(id: string): Promise<ProjectDetail>;
  uploadAsset(projectId: string, file: File, role: ImageRole, capture: Capture): Promise<Asset>;
  createComparison(
    projectId: string,
    referenceAssetId: string,
    actualAssetId: string,
    requirements: string,
  ): Promise<ComparisonDetail>;
  getComparison(id: string): Promise<ComparisonDetail>;
  startRun(comparisonId: string, idempotencyKey: string, allowDuplicate: boolean): Promise<Run>;
  getRun(id: string): Promise<RunDetail>;
  reviewFinding(findingId: string, decision: Decision, reason: string): Promise<Review>;
  exportRun(runId: string, format: ExportFormat): Promise<ExportFile>;
}
