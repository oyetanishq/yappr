import { lazy, Suspense, useState } from "react";
import { ChevronDown, GitPullRequest, ExternalLink, CheckCircle2, XCircle, Loader2, Ban, RefreshCw } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { usePRRuns, usePRRun } from "@/lib/hooks";
import { type PRRun, type PRRunStatus } from "@/lib/api";

// Mermaid pulls in a heavy rendering lib; load it only when a run detail is opened.
const Mermaid = lazy(() => import("@/components/mermaid"));

export default function DashboardPrRuns() {
	const { data: runs = [], isLoading, isError, refetch } = usePRRuns();

	return (
		<div className="flex flex-col gap-6 max-w-4xl">
			<section>
				<div className="flex items-center justify-between mb-4">
					<h2 className="text-xs uppercase tracking-widest text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						PR Review Runs
					</h2>
					<button
						onClick={() => refetch()}
						disabled={isLoading}
						className="flex items-center gap-2 px-3 py-1.5 border-[3px] border-border-stark hard-shadow bg-surface hover:bg-surface-container-highest text-xs font-semibold uppercase cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						{isLoading ? <Loader2 size={14} className="animate-spin" /> : <RefreshCw size={14} />}
						Refresh
					</button>
				</div>

				{/* Loading */}
				{isLoading && (
					<div className="flex flex-col gap-3">
						{[1, 2, 3].map((i) => (
							<div key={i} className="border-[3px] border-border-stark bg-surface-container p-5 animate-pulse h-20" />
						))}
					</div>
				)}

				{/* Error */}
				{!isLoading && isError && (
					<div className="border-[3px] border-border-stark border-dashed bg-error-container p-8 text-center" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						<p className="text-sm font-bold uppercase">Failed to load runs</p>
						<p className="text-xs text-on-surface-variant mt-1">Please refresh to try again.</p>
					</div>
				)}

				{/* Empty */}
				{!isLoading && !isError && runs.length === 0 && (
					<div className="border-[3px] border-border-stark border-dashed p-10 flex flex-col items-center gap-4 text-center">
						<div className="w-12 h-12 border-[3px] border-border-stark bg-surface-container flex items-center justify-center">
							<GitPullRequest size={24} className="text-on-surface-variant" />
						</div>
						<div>
							<p className="text-sm font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
								No PR Runs Yet
							</p>
							<p className="text-xs text-on-surface-variant mt-1 max-w-xs" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								Open a pull request on a connected repository and Yappr's review will show up here.
							</p>
						</div>
					</div>
				)}

				{/* List */}
				{!isLoading && !isError && runs.length > 0 && (
					<div className="flex flex-col gap-3">
						{runs.map((run) => (
							<RunRow key={run.id} run={run} />
						))}
					</div>
				)}
			</section>
		</div>
	);
}

// ── Status pill ────────────────────────────────────────────────────────────────

function StatusPill({ status }: { status: PRRunStatus }) {
	const base = "flex items-center gap-1.5 px-2.5 py-1 border-[3px] border-border-stark font-bold uppercase text-[11px] shrink-0";

	switch (status) {
		case "completed":
			return (
				<span className={`${base} bg-primary text-on-primary`}>
					<CheckCircle2 size={13} /> Completed
				</span>
			);
		case "failed":
			return (
				<span className={`${base} bg-error text-on-error`}>
					<XCircle size={13} /> Failed
				</span>
			);
		case "limit_reached":
			return (
				<span className={`${base} bg-error-container text-on-error-container`}>
					<Ban size={13} /> Limit Reached
				</span>
			);
		case "processing":
		default:
			return (
				<span className={`${base} bg-surface-container-highest`}>
					<Loader2 size={13} className="animate-spin" /> Processing
				</span>
			);
	}
}

// ── Run row (expandable) ────────────────────────────────────────────────────────

function RunRow({ run }: { run: PRRun }) {
	const [expanded, setExpanded] = useState(false);
	// Lazy-load full detail (review content) only once expanded.
	const { data: detail, isLoading: detailLoading } = usePRRun(expanded ? run.id : "");

	return (
		<div className="border-[3px] border-border-stark hard-shadow bg-surface-container flex flex-col overflow-hidden">
			{/* Header (always visible) */}
			<div className="p-4 flex items-center justify-between gap-4 cursor-pointer hover:bg-surface-container-highest" onClick={() => setExpanded(!expanded)}>
				<div className="flex items-center gap-3 min-w-0">
					<div className="w-9 h-9 flex items-center justify-center border-[3px] border-border-stark bg-primary-container shrink-0">
						<GitPullRequest size={16} />
					</div>
					<div className="min-w-0">
						<div className="flex items-center gap-2 min-w-0">
							<p className="text-sm font-bold truncate" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								{run.repo_full_name} <span className="text-on-surface-variant">#{run.pr_number}</span>
							</p>
						</div>
						<p className="text-xs text-on-surface-variant truncate" style={{ fontFamily: "var(--font-jetbrains-mono)" }} title={run.pr_title}>
							{run.pr_title || "(no title)"}
						</p>
						<p className="text-[11px] text-on-surface-variant mt-0.5 uppercase tracking-wide" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							@{run.author} · +{run.additions} -{run.deletions} · {run.files_changed} files · {relativeTime(run.created_at)}
						</p>
					</div>
				</div>

				<div className="flex items-center gap-3 shrink-0">
					<a
						href={run.pr_url}
						target="_blank"
						rel="noopener noreferrer"
						onClick={(e) => e.stopPropagation()}
						className="hidden sm:flex items-center gap-1.5 px-2.5 py-1.5 border-[3px] border-border-stark hard-shadow bg-surface hover:bg-primary-container text-xs font-semibold uppercase cursor-pointer"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						title="View on GitHub"
					>
						<ExternalLink size={13} /> GitHub
					</a>
					<StatusPill status={run.status} />
					<div className={`p-1 transition-transform duration-200 ${expanded ? "rotate-180" : ""}`}>
						<ChevronDown size={18} />
					</div>
				</div>
			</div>

			{/* Body (expanded) */}
			{expanded && (
				<div className="border-t-[3px] border-border-stark bg-surface-container-low p-4">
					{/* Mobile GitHub link (hidden in header on small screens) */}
					<a
						href={run.pr_url}
						target="_blank"
						rel="noopener noreferrer"
						className="sm:hidden inline-flex items-center gap-1.5 mb-4 px-2.5 py-1.5 border-[3px] border-border-stark hard-shadow bg-surface hover:bg-primary-container text-xs font-semibold uppercase"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						<ExternalLink size={13} /> View on GitHub
					</a>

					{detailLoading && (
						<div className="flex items-center gap-2 text-xs text-on-surface-variant uppercase tracking-wide" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							<Loader2 size={14} className="animate-spin" /> Loading review…
						</div>
					)}

					{!detailLoading && detail && <RunDetail run={detail} />}
				</div>
			)}
		</div>
	);
}

// ── Run detail (review content) ─────────────────────────────────────────────────

function RunDetail({ run }: { run: PRRun }) {
	if (run.status === "failed") {
		return (
			<div className="border-[3px] border-border-stark bg-error-container p-4" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
				<p className="text-xs font-bold uppercase mb-1">Review Failed</p>
				<p className="text-xs text-on-error-container break-words">{run.error || "The review pipeline errored."}</p>
			</div>
		);
	}

	if (run.status === "limit_reached") {
		return (
			<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
				This PR was not reviewed because the monthly free-tier review limit was reached.
			</p>
		);
	}

	if (run.status === "processing") {
		return (
			<p className="text-xs text-on-surface-variant flex items-center gap-2" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
				<Loader2 size={14} className="animate-spin" /> This review is still in progress.
			</p>
		);
	}

	// Completed — render the stored review content.
	const hasContent = run.summary || run.file_changes || run.bug_report || run.flow_diagram;
	if (!hasContent) {
		return (
			<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
				No review content was stored for this run.
			</p>
		);
	}

	return (
		<div className="flex flex-col gap-5">
			{run.summary && <Section title="PR Summary" markdown={run.summary} />}
			{run.file_changes && <Section title="File Changes" markdown={run.file_changes} />}
			{run.flow_diagram && (
				<div>
					<h3 className="text-[11px] uppercase tracking-widest text-on-surface-variant mb-2 font-bold" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						Architecture Flow
					</h3>
					<Suspense
						fallback={
							<div className="border-[3px] border-border-stark bg-surface p-4 flex items-center gap-2 text-xs text-on-surface-variant uppercase tracking-wide" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								<Loader2 size={14} className="animate-spin" /> Rendering diagram…
							</div>
						}
					>
						<Mermaid chart={stripMermaidFence(run.flow_diagram)} />
					</Suspense>
				</div>
			)}
			{run.bug_report && <Section title="Bugs & Edge Cases" markdown={run.bug_report} />}
		</div>
	);
}

function Section({ title, markdown }: { title: string; markdown: string }) {
	return (
		<div>
			<h3 className="text-[11px] uppercase tracking-widest text-on-surface-variant mb-2 font-bold" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
				{title}
			</h3>
			<div className="review-md border-[3px] border-border-stark bg-surface p-4">
				<ReactMarkdown remarkPlugins={[remarkGfm]}>{markdown}</ReactMarkdown>
			</div>
		</div>
	);
}

// ── helpers ─────────────────────────────────────────────────────────────────────

/** Formats an ISO timestamp as a compact relative time (e.g. "3h ago"). */
function relativeTime(iso: string): string {
	const then = new Date(iso).getTime();
	if (Number.isNaN(then)) return "";
	const diff = Date.now() - then;
	const sec = Math.round(diff / 1000);
	if (sec < 60) return "just now";
	const min = Math.round(sec / 60);
	if (min < 60) return `${min}m ago`;
	const hr = Math.round(min / 60);
	if (hr < 24) return `${hr}h ago`;
	const day = Math.round(hr / 24);
	if (day < 30) return `${day}d ago`;
	return new Date(iso).toLocaleDateString();
}

/** Strips a ```mermaid ... ``` wrapper if the stored diagram already includes one. */
function stripMermaidFence(text: string): string {
	const trimmed = text.trim();
	const start = trimmed.indexOf("```mermaid");
	if (start < 0) return trimmed;
	const rest = trimmed.slice(start + "```mermaid".length);
	const end = rest.lastIndexOf("```");
	return (end < 0 ? rest : rest.slice(0, end)).trim();
}
