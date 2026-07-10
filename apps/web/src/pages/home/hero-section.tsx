import { Terminal } from "lucide-react";
import { useNavigate } from "react-router";

const REPO_URL = "https://github.com/oyetanishq/yappr";

export default function HeroSection() {
	const navigate = useNavigate();

	return (
		<section className="max-w-6xl mx-auto mb-24 relative">
			{/* Main card */}
			<div className="bg-surface border-[3px] border-border-stark terminal-shadow p-8 md:p-16 text-center relative z-10">
				{/* Window chrome bar */}
				<div className="absolute top-0 left-0 w-full h-8 border-b-[3px] border-border-stark bg-inverse-primary flex items-center px-4 gap-2">
					<div className="w-3 h-3 bg-error border-2 border-border-stark" />
					<div className="w-3 h-3 bg-secondary-container border-2 border-border-stark" />
					<div className="w-3 h-3 bg-terminal-green border-2 border-border-stark" />
					<span className="ml-auto text-[12px] font-bold leading-tight uppercase tracking-wider" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						main.py
					</span>
				</div>

				{/* Headline */}
				<h1 className="mt-8 mb-6 text-on-surface uppercase text-4xl md:text-5xl lg:text-7xl font-bold leading-tight tracking-tight" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					THE AI THAT <span className="bg-primary-container border-[3px] border-border-stark px-2 inline-block -rotate-2 transform">YAPS</span> ABOUT YOUR CODE
				</h1>

				{/* Subtext */}
				<p
					className="text-lg text-on-surface-variant max-w-2xl mx-auto mb-10 border-l-4 border-border-stark pl-4 text-left bg-surface-container-low p-4 leading-relaxed"
					style={{ fontFamily: "var(--font-jetbrains-mono)" }}
				>
					Opinionated, aggressive, and incredibly accurate PR reviews. Stop merging trash and start shipping bulletproof infrastructure.
				</p>

				{/* CTA Buttons */}
				<div className="flex flex-col sm:flex-row gap-4 justify-center items-center">
					<button
						onClick={() => navigate("/login")}
						className="bg-primary-container text-on-surface font-bold text-[12px] leading-tight px-8 py-4 border-[3px] border-border-stark hard-shadow uppercase tracking-wider hover:bg-inverse-primary w-full sm:w-auto cursor-pointer"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						GET STARTED FOR FREE
					</button>
					<a
						href={REPO_URL}
						target="_blank"
						rel="noopener noreferrer"
						className="bg-surface text-on-surface font-bold text-[12px] leading-tight px-8 py-4 border-[3px] border-border-stark hard-shadow uppercase tracking-wider hover:bg-surface-container-highest w-full sm:w-auto flex items-center justify-center gap-2 cursor-pointer"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						<Terminal size={18} />
						View Repo
					</a>
				</div>
			</div>

			{/* Decorative backgrounds */}
			<div className="absolute -top-6 -right-6 w-32 h-32 bg-tertiary-container border-[3px] border-border-stark -z-10 opacity-50" />
			<div className="absolute -bottom-10 -left-10 w-48 h-48 bg-error-container border-[3px] border-border-stark rounded-full -z-10 opacity-20" />
		</section>
	);
}
