import { useCallback, useEffect, useState, type ButtonHTMLAttributes, type ReactNode } from "react";
import type { Category, EvidenceStatus, RunOrigin, RunStatus } from "../../../domain/types";

export function cx(...classes: (string | false | null | undefined)[]) {
  return classes.filter(Boolean).join(" ");
}

type Variant = "primary" | "secondary" | "danger" | "ghost";

const variants: Record<Variant, string> = {
  primary:
    "bg-neon text-paper font-semibold shadow-[0_0_18px_-4px_var(--color-neon)] hover:-translate-y-px hover:shadow-[0_0_28px_-2px_var(--color-neon)] active:translate-y-px disabled:bg-line-2 disabled:text-ink-3 disabled:shadow-none disabled:translate-y-0",
  secondary: "bg-paper-2 text-ink ring-1 ring-line-2 hover:ring-neon/60 hover:text-neon disabled:text-ink-3",
  danger: "bg-paper-2 text-reject ring-1 ring-reject/50 hover:bg-reject/10 hover:shadow-[0_0_16px_-4px_var(--color-reject)] disabled:text-reject/40",
  ghost: "text-ink-2 hover:bg-ink/5 disabled:text-ink-3",
};

export function Button({ variant = "primary", className, ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return (
    <button
      type="button"
      {...props}
      className={cx(
        "inline-flex items-center justify-center gap-1.5 rounded-md px-3.5 py-2 text-sm font-medium transition-all disabled:cursor-not-allowed",
        variants[variant],
        className,
      )}
    />
  );
}

/** Four printer's crop marks around the parent (which must be positioned). */
export function CropMarks({ className }: { className?: string }) {
  const corner = "pointer-events-none absolute size-3 border-neon/60";
  return (
    <span aria-hidden className={className}>
      <span className={cx(corner, "-top-2 -left-2 border-t border-l")} />
      <span className={cx(corner, "-top-2 -right-2 border-t border-r")} />
      <span className={cx(corner, "-bottom-2 -left-2 border-b border-l")} />
      <span className={cx(corner, "-right-2 -bottom-2 border-r border-b")} />
    </span>
  );
}

export function Card({
  title,
  kicker,
  actions,
  children,
  className,
}: {
  title?: ReactNode;
  kicker?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={cx("sheet-shadow min-w-0 rounded-lg border border-line bg-sheet", className)}>
      {(title || actions) && (
        <div className="flex items-center justify-between gap-2 border-b border-line/70 px-4 py-3">
          <div className="min-w-0">
            {kicker && <p className="font-mono text-[10px] tracking-widest text-ink-3 uppercase">{kicker}</p>}
            <h2 className="text-sm font-semibold text-ink">{title}</h2>
          </div>
          {actions}
        </div>
      )}
      <div className="p-4">{children}</div>
    </section>
  );
}

/** Explanatory note. Plain language, shown where a user would otherwise guess. */
export function Callout({ tone = "info", title, children }: { tone?: "info" | "warn"; title?: ReactNode; children: ReactNode }) {
  return (
    <div
      className={cx(
        "rounded-md border-l-2 px-3 py-2 text-xs leading-relaxed",
        tone === "info" ? "border-design bg-design/5 text-ink-2" : "border-clipped bg-clipped/10 text-ink",
      )}
    >
      {title && <p className="mb-0.5 font-semibold text-ink">{title}</p>}
      {children}
    </div>
  );
}

export function ErrorBanner({ error, onDismiss }: { error: unknown; onDismiss?: () => void }) {
  if (!error) return null;
  const message = error instanceof Error ? error.message : String(error);
  return (
    <div role="alert" className="flex items-start justify-between gap-3 rounded-md border border-missing/30 bg-missing/5 px-3 py-2 text-sm text-reject">
      <span>{message}</span>
      {onDismiss && (
        <button type="button" onClick={onDismiss} className="text-reject/60 hover:text-reject" aria-label="Dismiss">
          ×
        </button>
      )}
    </div>
  );
}

const statusStyle: Record<RunStatus, string> = {
  queued: "bg-ink/5 text-ink-2",
  running: "bg-design/10 text-design",
  completed: "bg-accept/10 text-accept",
  needs_review: "bg-clipped/15 text-clipped",
  failed: "bg-missing/10 text-missing",
};

export function StatusBadge({ status }: { status: RunStatus }) {
  return (
    <span className={cx("rounded px-1.5 py-0.5 font-mono text-[11px] font-medium uppercase", statusStyle[status])}>
      {status.replace("_", " ")}
    </span>
  );
}

/** Origin is always visible so mock or fixture output is never mistaken for model output. */
export function OriginBadge({ origin }: { origin: RunOrigin }) {
  const real = origin === "meta_api";
  return (
    <span
      className={cx(
        "rounded px-1.5 py-0.5 font-mono text-[11px] font-semibold uppercase",
        real ? "bg-design text-paper shadow-[0_0_12px_-2px_var(--color-design)]" : "border border-dashed border-missing/60 bg-sheet text-missing",
      )}
      title={real ? "Hosted model output" : "Not model output"}
    >
      {real ? "Muse Spark" : origin}
    </span>
  );
}

export function EvidenceBadge({ status }: { status: EvidenceStatus }) {
  return status === "validated" ? (
    <span className="rounded bg-accept/10 px-1.5 py-0.5 text-[11px] text-accept" title="Boxes passed the evidence rules for this category">
      evidence ok
    </span>
  ) : (
    <span className="rounded bg-clipped/15 px-1.5 py-0.5 text-[11px] text-clipped" title="Boxes are missing or unusual; check by eye">
      check evidence
    </span>
  );
}

/** What each defect category means, written for someone seeing the tool for the first time. */
export const CATEGORY_INFO: Record<Category, { label: string; color: string; text: string; bg: string; meaning: string; boxes: string }> = {
  missing_element: {
    label: "Missing element",
    color: "var(--color-missing)",
    text: "text-missing",
    bg: "bg-missing",
    meaning: "Something in the design does not appear in the build.",
    boxes: "Boxed on the design only. There is nothing to box in the build.",
  },
  clipped_text: {
    label: "Clipped text",
    color: "var(--color-clipped)",
    text: "text-clipped",
    bg: "bg-clipped",
    meaning: "Text in the build is cut off, truncated, or overflows its container.",
    boxes: "Boxed on both: the full text in the design, the cut text in the build.",
  },
  layout_mismatch: {
    label: "Layout mismatch",
    color: "var(--color-layout)",
    text: "text-layout",
    bg: "bg-layout",
    meaning: "An element exists in both but sits at a different position or size.",
    boxes: "Boxed on both, so you can see where it is versus where it should be.",
  },
};

export function CategoryGlyph({ category, className }: { category: Category; className?: string }) {
  const common = { className: cx("size-4 shrink-0", className), viewBox: "0 0 16 16", fill: "none", stroke: "currentColor", strokeWidth: 1.5, "aria-hidden": true };
  if (category === "missing_element") {
    return (
      <svg {...common}>
        <rect x="2.5" y="2.5" width="11" height="11" rx="1" strokeDasharray="2 2" />
        <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" />
      </svg>
    );
  }
  if (category === "clipped_text") {
    return (
      <svg {...common}>
        <path d="M2 5h12M2 8h12M2 11h6" />
        <path d="M11 2.5v11" strokeDasharray="1.5 1.5" />
      </svg>
    );
  }
  return (
    <svg {...common}>
      <rect x="2" y="2" width="7" height="7" rx="1" strokeDasharray="2 2" />
      <rect x="7" y="7" width="7" height="7" rx="1" />
    </svg>
  );
}

export function CategoryTag({ category }: { category: Category }) {
  const info = CATEGORY_INFO[category];
  return (
    <span className={cx("inline-flex items-center gap-1 text-[11px] font-semibold tracking-wide uppercase", info.text)}>
      <CategoryGlyph category={category} className="size-3.5" />
      {info.label}
    </span>
  );
}

const STEPS = ["Upload", "Pair", "Analyse", "Review", "Export"] as const;
export type Step = (typeof STEPS)[number];

/** The five-step workflow, so users always know where they are and what comes next. */
export function Steps({ current, done = [] }: { current: Step; done?: Step[] }) {
  return (
    <ol className="flex flex-wrap items-center gap-x-1 gap-y-2 font-mono text-[11px]" aria-label="Workflow">
      {STEPS.map((s, i) => {
        const isCurrent = s === current;
        const isDone = done.includes(s);
        return (
          <li key={s} className="flex items-center gap-1">
            <span
              aria-current={isCurrent ? "step" : undefined}
              className={cx(
                "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5",
                isCurrent ? "bg-neon/10 text-neon ring-1 ring-neon shadow-[0_0_14px_-4px_var(--color-neon)]" : isDone ? "text-accept" : "text-ink-3",
              )}
            >
              <span
                className={cx(
                  "grid size-4 place-items-center rounded-full text-[9px]",
                  isCurrent ? "bg-neon text-paper" : isDone ? "bg-accept text-paper" : "ring-1 ring-line-2",
                )}
              >
                {isDone ? "✓" : i + 1}
              </span>
              {s}
            </span>
            {i < STEPS.length - 1 && <span className="h-px w-3 bg-line-2" aria-hidden />}
          </li>
        );
      })}
    </ol>
  );
}

export const inputClass =
  "w-full rounded-md border border-line-2 bg-paper-2 px-2.5 py-1.5 text-sm text-ink placeholder:text-ink-3 focus:border-neon focus:outline-none focus:ring-2 focus:ring-neon/25";

export function Field({ label, hint, children }: { label: string; hint?: ReactNode; children: ReactNode }) {
  return (
    <label className="block min-w-0 space-y-1">
      <span className="text-xs font-medium text-ink-2">{label}</span>
      {children}
      {hint && <span className="block text-[11px] leading-snug text-ink-3">{hint}</span>}
    </label>
  );
}

/** Loads data on mount and whenever deps change; reload() refetches. */
export function useLoad<T>(load: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [version, setVersion] = useState(0);
  const stableLoad = useCallback(load, deps);
  useEffect(() => {
    let cancelled = false;
    setError(null);
    stableLoad().then(
      (d) => !cancelled && setData(d),
      (e) => !cancelled && setError(e),
    );
    return () => {
      cancelled = true;
    };
  }, [stableLoad, version]);
  return { data, error, reload: () => setVersion((v) => v + 1), setData };
}

export function formatBytes(n: number) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export function formatTime(iso: string) {
  return new Date(iso).toLocaleString();
}
