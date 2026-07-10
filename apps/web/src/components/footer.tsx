import { Link } from "react-router";

const REPO_URL = "https://github.com/oyetanishq/yappr";
const PORTFOLIO_URL = "https://tanishqsingh.com";

const linkClass = "text-[12px] uppercase transition-all hover:text-primary-container";
const linkStyle = {
	fontFamily: "var(--font-jetbrains-mono)",
	color: "var(--color-surface-variant)",
} as const;

export default function Footer() {
	return (
		<footer className="w-full py-4 px-6 flex justify-between items-center border-t-[3px] border-black mt-auto" style={{ backgroundColor: "var(--color-inverse-surface)" }}>
			<span
				className="text-[12px] font-bold uppercase"
				style={{
					fontFamily: "var(--font-jetbrains-mono)",
					color: "var(--color-terminal-green)",
				}}
			>
				&copy;2026 YAPPR_SYSTEMS
			</span>
			<div className="flex gap-4">
				{/* Internal system status page */}
				<Link to="/status" className={linkClass} style={linkStyle}>
					System Status
				</Link>

				{/* Docs → GitHub repository (new tab) */}
				<a href={REPO_URL} target="_blank" rel="noopener noreferrer" className={linkClass} style={linkStyle}>
					Docs
				</a>

				{/* Credits → personal portfolio (new tab) */}
				<a href={PORTFOLIO_URL} target="_blank" rel="noopener noreferrer" className={linkClass} style={linkStyle}>
					Credits
				</a>
			</div>
		</footer>
	);
}
