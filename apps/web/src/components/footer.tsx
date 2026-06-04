const footerLinks = [
	{ label: "System Status", href: "#" },
	{ label: "API Docs", href: "#" },
	{ label: "Credits", href: "#" },
];

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
				{footerLinks.map((link) => (
					<a
						key={link.label}
						href={link.href}
						className="text-[12px] uppercase transition-all hover:text-primary-container"
						style={{
							fontFamily: "var(--font-jetbrains-mono)",
							color: "var(--color-surface-variant)",
						}}
					>
						{link.label}
					</a>
				))}
			</div>
		</footer>
	);
}
