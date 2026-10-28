export type Route =
  | { page: "projects" }
  | { page: "project"; projectId: string }
  | { page: "comparison"; comparisonId: string; runId?: string };

export function parseHash(hash: string): Route {
  const parts = hash.replace(/^#\/?/, "").split("/").filter(Boolean).map(decodeURIComponent);
  if (parts[0] === "projects" && parts[1]) return { page: "project", projectId: parts[1] };
  if (parts[0] === "comparisons" && parts[1]) {
    return { page: "comparison", comparisonId: parts[1], runId: parts[2] === "runs" ? parts[3] : undefined };
  }
  return { page: "projects" };
}

export const href = {
  projects: () => "#/",
  project: (id: string) => `#/projects/${encodeURIComponent(id)}`,
  comparison: (id: string, runId?: string) =>
    `#/comparisons/${encodeURIComponent(id)}` + (runId ? `/runs/${encodeURIComponent(runId)}` : ""),
};

export function navigate(to: string) {
  window.location.hash = to;
}
