import { type LucideIcon, SlidersHorizontal, Terminal } from "lucide-react";
import { useNavigate } from "react-router";

interface DiffLine {
	type: "removed" | "added";
	content: string;
}

interface CodeDiffProps {
	filename: string;
	lines: DiffLine[];
}

function CodeDiff({ filename, lines }: CodeDiffProps) {
	return (
		<div
			className="overflow-hidden mt-8 mb-6 border-[3px] border-border-stark"
			style={{
				backgroundColor: "#fff",
				fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
				textAlign: "left",
			}}
		>
			{/* File header */}
			<div
				style={{
					backgroundColor: "rgb(246,248,250)",
					borderBottom: "3px solid #000",
					color: "rgb(31,35,40)",
					padding: "8px 16px",
					fontWeight: 600,
					fontSize: "12px",
					textTransform: "uppercase",
				}}
			>
				{filename}
			</div>
			<div style={{ padding: 0, fontSize: "12px", lineHeight: "20px" }}>
				{lines.map((line, i) => (
					<div
						key={i}
						className="border-b border-gray-200 last:border-b-0"
						style={{
							backgroundColor: line.type === "removed" ? "#ffeef0" : "#e6ffed",
							color: line.type === "removed" ? "#cf222e" : "#1a7f37",
							padding: "0 16px",
						}}
					>
						{line.content}
					</div>
				))}
			</div>
		</div>
	);
}

const diffLines: DiffLine[] = [
	{ type: "removed", content: "- for item in items:" },
	{ type: "removed", content: '- db.query("SELECT * FROM users WHERE id=" + item.id)' },
	{ type: "added", content: "+ # Batch query to prevent N+1 and SQLi" },
	{ type: "added", content: '+ db.query("SELECT * FROM users WHERE id IN ?", [item_ids])' },
];

interface AiCodeReviewProps {
	icon: LucideIcon;
	iconBg: string;
	iconColor: string;
	title: string;
	description: React.ReactNode;
	actionLabel: string;
	actionIcon: LucideIcon;
	reverse?: boolean;
	visualSlot: React.ReactNode;
	onAction?: () => void;
}

function FeatureRow({ icon: Icon, iconBg, iconColor, title, description, actionLabel, actionIcon: ActionIcon, reverse = false, visualSlot, onAction }: AiCodeReviewProps) {
	return (
		<section className={`max-w-6xl mx-auto mb-24 flex flex-col gap-12 items-center ${reverse ? "lg:flex-row-reverse" : "lg:flex-row"}`}>
			{/* Text side */}
			<div className="flex-1">
				<div className="flex items-center gap-4 mb-6">
					<span className={`p-3 border-[3px] border-border-stark hard-shadow`} style={{ backgroundColor: iconBg, color: iconColor }}>
						<Icon size={32} />
					</span>
					<h3 className="text-3xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						{title}
					</h3>
				</div>
				<p className="text-lg text-on-surface-variant mb-6 bg-surface-container-low p-4 border-l-4 border-border-stark leading-relaxed" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					{description}
				</p>
				<button
					onClick={onAction}
					className="bg-surface text-on-surface font-bold text-[12px] px-6 py-3 border-[3px] border-border-stark hard-shadow uppercase tracking-wider hover:bg-surface-container-highest flex items-center gap-2 cursor-pointer"
					style={{ fontFamily: "var(--font-jetbrains-mono)" }}
				>
					<ActionIcon size={16} />
					{actionLabel}
				</button>
			</div>

			{/* Visual side */}
			<div className="flex-1 w-full">{visualSlot}</div>
		</section>
	);
}

export default function AiCodeReviewSection() {
	const navigate = useNavigate();

	return (
		<FeatureRow
			icon={Terminal}
			iconBg="var(--color-primary)"
			iconColor="var(--color-on-primary)"
			title="AI-Powered Code Reviews"
			onAction={() => navigate("/login")}
			description={
				<>
					Experience the &ldquo;Yapping&rdquo; personality. YAPPR doesn&rsquo;t just review your code; it roasts it. Tune the personality from <em>&lsquo;Helpful Senior Dev&rsquo;</em> to{" "}
					<em>&lsquo;Toxic Tech Lead&rsquo;</em> depending on how much reality your team can handle.
				</>
			}
			actionLabel="Configure Personality"
			actionIcon={SlidersHorizontal}
			visualSlot={
				<div className="bg-surface border-[3px] border-border-stark hard-shadow p-6 relative overflow-hidden group">
					{/* Badge */}
					<div
						className="absolute top-0 right-0 p-2 bg-error-container border-l-[3px] border-b-[3px] border-border-stark text-[12px] font-bold uppercase"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						Toxic_Mode: ON
					</div>

					<CodeDiff filename="data_parser.py - diff" lines={diffLines} />

					{/* Review box */}
					<div className="border-[3px] border-border-stark p-4 bg-error text-on-error hard-shadow">
						<div className="text-[12px] font-bold mb-2 flex items-center gap-2 uppercase" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							⚠ YAPPR REVIEW
						</div>
						<p className="text-sm leading-relaxed" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							An N+1 query AND string concatenation for SQL? Are you actively trying to get us hacked while simultaneously taking down the database? I rewrote this to batch the queries
							safely. Please read a book.
						</p>
					</div>
				</div>
			}
		/>
	);
}
