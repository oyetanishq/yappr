import { GitBranch, Search } from "lucide-react";

export default function ArchitectureMappingSection() {
	return (
		<section className="max-w-6xl mx-auto mb-24 flex flex-col lg:flex-row-reverse items-center gap-12">
			{/* Text side */}
			<div className="flex-1">
				<div className="flex items-center gap-4 mb-6">
					<span className="p-3 border-[3px] border-border-stark hard-shadow bg-tertiary-container text-on-tertiary-container">
						<GitBranch size={32} />
					</span>
					<h3 className="text-3xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						Architecture Mapping
					</h3>
				</div>
				<p className="text-lg text-on-surface-variant mb-6 bg-surface-container-low p-4 border-l-4 border-border-stark leading-relaxed" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					We don&apos;t just look at isolated diffs. YAPPR builds a comprehensive mental model of your entire monolith. It detects downstream ripple effects, circular dependencies, and
					architectural anti-patterns before they hit production.
					<br />
					<br />
					Change a core auth interface? We&apos;ll tell you exactly which microservices you just broke.
				</p>
			</div>

			{/* Visual side */}
			<div className="flex-1 w-full bg-surface-container-highest border-[3px] border-border-stark hard-shadow p-6 relative">
				{/* Search badge */}
				<div className="absolute -top-4 -left-4 w-12 h-12 bg-secondary-container border-[3px] border-border-stark rounded-full z-10 flex items-center justify-center hard-shadow">
					<Search size={20} className="text-on-secondary-container" />
				</div>

				<img
					alt="Architecture Diagram"
					className="w-full aspect-video object-cover border-[3px] border-border-stark relative z-0"
					src="https://lh3.googleusercontent.com/aida-public/AB6AXuAOzqaxwMNnbyLaRdH9Z61VHqVSuI9sZr0YoDpCoRM2Cn87TIisnY7VfAp67WJpjHZL28GA5xVQ-IGsxJnAG_i8BzYMnahIH7KUVe9fQqprPkiEUSHuobG_Uc2WC7107XpgI5kRjufBsj4s7EQBMKBttjILgsWwCmkP8dXk6S4Uad4pa-fuSrjnZWOgHdJvpeQOns2V7RS3R5BAD0N-9qbuGK3-lq8DGumB84KT2N36Q8EefrS-U8MA7jPm2NEG0kMmWY4OADqWZcnu"
				/>
			</div>
		</section>
	);
}
