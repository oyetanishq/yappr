import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router";
import { ArrowLeft, Save, AlertCircle, CheckCircle2, FolderGit2, FileX, Sparkles } from "lucide-react";
import { useRepoConfig, useUpdateRepoConfig } from "@/lib/hooks";
import { PERSONALITIES, PERSONALITY_LABELS, PERSONALITY_DESCRIPTIONS, type Personality } from "@/lib/api";

// ── Personality slider data ───────────────────────────────────────────────────

const PERSONALITY_ICONS: Record<Personality, string> = {
	bestie: "🧸",
	senior_dev: "🧑‍💻",
	sigma: "🪨",
	toxic_tech_lead: "☠️",
};

const PERSONALITY_COLORS: Record<Personality, string> = {
	bestie: "var(--color-primary)",
	senior_dev: "var(--color-on-surface)",
	sigma: "var(--color-on-surface-variant)",
	toxic_tech_lead: "var(--color-error, #e53e3e)",
};

// ── Component ─────────────────────────────────────────────────────────────────

export default function RepoConfig() {
	const { owner, repo } = useParams<{ owner: string; repo: string }>();
	const navigate = useNavigate();
	const repoFullName = `${owner}/${repo}`;

	const { data: config, isLoading } = useRepoConfig(owner ?? "", repo ?? "");
	const { mutate: updateConfig, isPending, isError, isSuccess } = useUpdateRepoConfig();

	const [ignoredPaths, setIgnoredPaths] = useState<string>("");
	const [personality, setPersonality] = useState<Personality>("senior_dev");
	const [sliderIdx, setSliderIdx] = useState(1); // senior_dev is index 1

	// Populate form when config loads
	useEffect(() => {
		if (config) {
			setIgnoredPaths((config.ignored_paths ?? []).join("\n"));
			const p = config.personality || "senior_dev";
			setPersonality(p);
			setSliderIdx(PERSONALITIES.indexOf(p));
		}
	}, [config]);

	const handleSliderChange = (idx: number) => {
		setSliderIdx(idx);
		setPersonality(PERSONALITIES[idx]);
	};

	const handleSave = async () => {
		if (!owner || !repo) return;
		const paths = ignoredPaths
			.split("\n")
			.map((p) => p.trim())
			.filter((p) => p.length > 0);

		await updateConfig(owner, repo, { ignored_paths: paths, personality });
	};

	return (
		<div className="flex flex-col gap-8 max-w-3xl" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
			{/* ── Breadcrumb / Header ────────────────────────────────────────── */}
			<div className="flex flex-col gap-3">
				<button
					onClick={() => navigate("/dashboard/repositories")}
					className="flex items-center gap-2 text-xs uppercase tracking-widest text-on-surface-variant hover:text-on-surface w-fit cursor-pointer"
				>
					<ArrowLeft size={14} />
					Back to Repositories
				</button>

				<div className="border-[3px] border-border-stark hard-shadow bg-surface-container-low p-5 flex items-center gap-4">
					<div className="w-10 h-10 flex items-center justify-center border-[3px] border-border-stark bg-primary-container shrink-0">
						<FolderGit2 size={18} />
					</div>
					<div>
						<p className="text-[10px] uppercase tracking-widest text-on-surface-variant">Repo Configuration</p>
						<h1 className="text-xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
							{repoFullName}
						</h1>
					</div>
				</div>
			</div>

			{/* ── Loading skeleton ───────────────────────────────────────────── */}
			{isLoading && (
				<div className="flex flex-col gap-4">
					{[1, 2].map((i) => (
						<div key={i} className="border-[3px] border-border-stark bg-surface-container p-6 animate-pulse h-40" />
					))}
				</div>
			)}

			{!isLoading && (
				<>
					{/* ── Personality Slider ─────────────────────────────────────── */}
					<section className="border-[3px] border-border-stark hard-shadow bg-surface-container p-6 flex flex-col gap-5">
						<div className="flex items-center gap-3">
							<Sparkles size={16} className="text-primary" />
							<h2 className="text-xs uppercase tracking-widest font-bold text-on-surface">Reviewer Personality</h2>
						</div>
						<p className="text-xs text-on-surface-variant">Choose how Yappr writes its code review comments for this repo.</p>

						{/* Slider track */}
						<div className="flex flex-col gap-4">
							<input
								id="personality-slider"
								type="range"
								min={0}
								max={PERSONALITIES.length - 1}
								step={1}
								value={sliderIdx}
								onChange={(e) => handleSliderChange(Number(e.target.value))}
								className="w-full h-2 appearance-none cursor-pointer accent-current"
								style={{ accentColor: PERSONALITY_COLORS[PERSONALITIES[sliderIdx]] }}
							/>

							{/* Stop labels */}
							<div className="grid grid-cols-4 gap-1">
								{PERSONALITIES.map((p, idx) => (
									<button
										key={p}
										id={`personality-${p}`}
										onClick={() => handleSliderChange(idx)}
										className={`flex flex-col items-center gap-1.5 p-2 border-[3px] transition-all cursor-pointer text-center ${
											sliderIdx === idx ? "border-border-stark bg-primary-container hard-shadow" : "border-transparent hover:border-border-stark bg-surface-container-low"
										}`}
									>
										<span className="text-xl">{PERSONALITY_ICONS[p]}</span>
										<span className={`text-[10px] font-bold uppercase tracking-wide leading-tight ${sliderIdx === idx ? "text-on-surface" : "text-on-surface-variant"}`}>
											{PERSONALITY_LABELS[p]}
										</span>
									</button>
								))}
							</div>

							{/* Active personality description */}
							<div className="border-[3px] border-border-stark bg-surface-container-low p-4">
								<p className="text-lg font-bold mb-1" style={{ fontFamily: "var(--font-space-grotesk)" }}>
									{PERSONALITY_ICONS[PERSONALITIES[sliderIdx]]} {PERSONALITY_LABELS[PERSONALITIES[sliderIdx]]}
								</p>
								<p className="text-xs text-on-surface-variant">{PERSONALITY_DESCRIPTIONS[PERSONALITIES[sliderIdx]]}</p>
							</div>
						</div>
					</section>

					{/* ── Ignored Files/Folders ──────────────────────────────────── */}
					<section className="border-[3px] border-border-stark hard-shadow bg-surface-container p-6 flex flex-col gap-5">
						<div className="flex items-center gap-3">
							<FileX size={16} className="text-on-surface-variant" />
							<h2 className="text-xs uppercase tracking-widest font-bold text-on-surface">Ignored Files & Folders</h2>
						</div>
						<p className="text-xs text-on-surface-variant">Files and folders matching these patterns will be skipped during code review. One pattern per line. Supports glob patterns.</p>

						<textarea
							id="ignored-paths-input"
							value={ignoredPaths}
							onChange={(e) => setIgnoredPaths(e.target.value)}
							placeholder={"dist/\nnode_modules/\n**/*.lock\n*.pb.go\n*.min.js\ncoverage/"}
							rows={8}
							className="w-full border-[3px] border-border-stark bg-surface p-4 text-sm text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:border-primary resize-y"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						/>

						<div className="flex flex-wrap gap-2">
							{["dist/", "node_modules/", "**/*.lock", "*.pb.go", "coverage/"].map((example) => (
								<button
									key={example}
									onClick={() =>
										setIgnoredPaths((prev) => {
											const existing = prev
												.split("\n")
												.map((l) => l.trim())
												.filter(Boolean);
											if (existing.includes(example)) return prev;
											return [...existing, example].join("\n");
										})
									}
									className="text-[10px] font-semibold uppercase tracking-wide px-2 py-1 border-2 border-border-stark bg-surface-container-low hover:bg-primary-container cursor-pointer transition-colors"
								>
									+ {example}
								</button>
							))}
						</div>
					</section>

					{/* ── Save / Feedback ───────────────────────────────────────── */}
					<div className="flex items-center gap-4">
						<button
							id="save-repo-config-btn"
							onClick={handleSave}
							disabled={isPending}
							className={`flex items-center gap-2 px-6 py-3 border-[3px] border-border-stark hard-shadow font-bold uppercase text-sm cursor-pointer transition-all ${
								isPending ? "bg-surface-container text-on-surface-variant" : "bg-on-surface text-surface hover:bg-primary hover:text-on-primary"
							}`}
						>
							<Save size={16} />
							{isPending ? "Saving…" : "Save Configuration"}
						</button>

						{isSuccess && !isPending && (
							<div className="flex items-center gap-2 text-xs text-primary">
								<CheckCircle2 size={14} />
								<span className="uppercase tracking-wide font-semibold">Saved!</span>
							</div>
						)}

						{isError && !isPending && (
							<div className="flex items-center gap-2 text-xs text-error">
								<AlertCircle size={14} />
								<span className="uppercase tracking-wide font-semibold">Save failed. Try again.</span>
							</div>
						)}
					</div>
				</>
			)}
		</div>
	);
}
