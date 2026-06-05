import { Bug, AlertTriangle, Info } from "lucide-react";
import type { LucideIcon } from "lucide-react";

interface BugReportItem {
	severity: "critical" | "warn" | "info";
	label: string;
	title: string;
	description: string;
}

interface BugCardProps extends BugReportItem {
	badgeBg: string;
	badgeColor: string;
	icon: LucideIcon;
	containerBg: string;
	labelColor: string;
}

function BugCard({ label, title, description, badgeBg, badgeColor, icon: Icon, containerBg, labelColor }: BugCardProps) {
	return (
		<div
			className="flex items-start gap-4 p-4 border-[3px] border-border-stark shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] hover:-translate-y-1 transition-transform"
			style={{ backgroundColor: containerBg }}
		>
			<div className="w-8 h-8 border-2 border-border-stark flex items-center justify-center shrink-0" style={{ backgroundColor: badgeBg, color: badgeColor }}>
				<Icon size={14} />
			</div>
			<div>
				<span
					className="text-[12px] font-bold uppercase mb-1 block"
					style={{
						fontFamily: "var(--font-jetbrains-mono)",
						color: labelColor,
					}}
				>
					{label}
				</span>
				<h4 className="text-sm font-bold mb-1" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					{title}
				</h4>
				<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					{description}
				</p>
			</div>
		</div>
	);
}

const bugs: BugReportItem[] = [
	{
		severity: "critical",
		label: "CRITICAL: Memory Leak",
		title: "Connection pool exhaustion",
		description: "Detected in `auth_worker.py`. Unclosed DB connections inside a retry loop will crash the pod under load.",
	},
	{
		severity: "warn",
		label: "WARN: Deprecated API",
		title: "Legacy Endpoint Usage",
		description: "Calling `v1/payments` endpoint which is scheduled for deprecation in 30 days. Update to `v2/checkout`.",
	},
	{
		severity: "info",
		label: "INFO: Nitpick (Spacing)",
		title: "Inconsistent Indentation",
		description: "Line 42 has mixed tabs and spaces. PEP8 violation but functionally harmless.",
	},
];

const bugCardConfig: Omit<BugCardProps, keyof BugReportItem>[] = [
	{
		badgeBg: "var(--color-error)",
		badgeColor: "var(--color-on-error)",
		icon: AlertTriangle,
		containerBg: "var(--color-error-container)",
		labelColor: "var(--color-on-error-container)",
	},
	{
		badgeBg: "var(--color-tertiary)",
		badgeColor: "var(--color-on-tertiary)",
		icon: AlertTriangle,
		containerBg: "var(--color-tertiary-container)",
		labelColor: "var(--color-on-tertiary-container)",
	},
	{
		badgeBg: "var(--color-outline)",
		badgeColor: "var(--color-on-surface)",
		icon: Info,
		containerBg: "var(--color-surface-container-highest)",
		labelColor: "var(--color-on-surface-variant)",
	},
];

export default function BugReportsSection() {
	return (
		<section className="max-w-6xl mx-auto mb-24 flex flex-col lg:flex-row items-center gap-12">
			{/* Text side */}
			<div className="flex-1">
				<div className="flex items-center gap-4 mb-6">
					<span className="p-3 border-[3px] border-border-stark hard-shadow bg-secondary text-on-secondary">
						<Bug size={32} />
					</span>
					<h3 className="text-3xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						Severity-Coded Bug Reports
					</h3>
				</div>
				<p className="text-lg text-on-surface-variant mb-6 bg-surface-container-low p-4 border-l-4 border-border-stark leading-relaxed" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					Stop treating minor formatting issues like SEV1 incidents. YAPPR intelligently categorizes flaws so you can focus on the critical issues that actually matter and ignore the trivial
					stuff until the next sprint.
				</p>
			</div>

			{/* Visual side */}
			<div className="flex-1 w-full border-[3px] border-border-stark hard-shadow p-6" style={{ backgroundColor: "var(--color-cream-darker)" }}>
				<div className="flex flex-col gap-4">
					{bugs.map((bug, i) => (
						<BugCard key={bug.severity} {...bug} {...bugCardConfig[i]} />
					))}
				</div>
			</div>
		</section>
	);
}
