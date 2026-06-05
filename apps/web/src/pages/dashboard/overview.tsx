import { GitBranch, Star, Zap, Shield, ChevronRight } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { useInstallations } from "@/lib/hooks";

const otherActions = [
	{ label: "Configure Review Rules", desc: "Customise what Yappr checks for", href: "#" },
	{ label: "Invite Team Members", desc: "Collaborate with your team", href: "#" },
];

export default function DashboardOverview() {
	const { user } = useAuthStore();
	const { data: installations = [], isLoading: installationsLoading } = useInstallations();

	const statCards = [
		{ label: "PRs Reviewed", value: "—", icon: GitBranch },
		{ label: "Issues Found", value: "—", icon: Shield },
		{
			label: "Repos Connected",
			value: installationsLoading ? "…" : String(installations.length),
			icon: Star,
		},
		{ label: "AI Reviews", value: "—", icon: Zap },
	];

	return (
		<div className="flex flex-col gap-10 max-w-4xl">
			{/* Welcome banner */}
			<section className="border-[3px] border-border-stark hard-shadow bg-surface-container-low p-6 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
				<div className="flex items-center gap-4">
					{user?.avatar_url ? (
						<img src={user.avatar_url} alt={user.login} className="w-16 h-16 border-[3px] border-border-stark" />
					) : (
						<div className="w-16 h-16 border-[3px] border-border-stark bg-primary-container" />
					)}
					<div>
						<p className="text-xs text-on-surface-variant uppercase tracking-widest" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							Welcome back
						</p>
						<h1 className="text-3xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
							{user?.name || user?.login || "Developer"}
						</h1>
						<a
							href={`https://github.com/${user?.login}`}
							target="_blank"
							rel="noopener noreferrer"
							className="inline-flex items-center gap-1.5 text-xs text-on-surface-variant hover:text-on-surface mt-0.5"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							@{user?.login}
						</a>
					</div>
				</div>

				<div className="flex items-center gap-2 px-4 py-2 border-[3px] border-border-stark bg-primary-container hard-shadow" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					<div className="w-2 h-2 bg-border-stark rounded-full animate-pulse" />
					<span className="text-xs font-semibold uppercase tracking-widest text-on-primary-container">Beta Access</span>
				</div>
			</section>

			{/* Stats grid */}
			<section>
				<h2 className="text-xs uppercase tracking-widest text-on-surface-variant mb-4" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					At a Glance
				</h2>
				<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
					{statCards.map(({ label, value, icon: Icon }) => (
						<div key={label} className="border-[3px] border-border-stark hard-shadow bg-surface-container p-5 flex flex-col gap-3">
							<div className="w-9 h-9 flex items-center justify-center border-[3px] border-border-stark bg-primary-container">
								<Icon size={16} />
							</div>
							<div>
								<p className="text-3xl font-bold" style={{ fontFamily: "var(--font-space-grotesk)" }}>
									{value}
								</p>
								<p className="text-xs text-on-surface-variant mt-0.5" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
									{label}
								</p>
							</div>
						</div>
					))}
				</div>
			</section>

			{/* Quick actions */}
			<section>
				<h2 className="text-xs uppercase tracking-widest text-on-surface-variant mb-4" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					Quick Actions
				</h2>
				<div className="flex flex-col gap-3">
					{otherActions.map(({ label, desc, href }) => (
						<a
							key={label}
							href={href}
							className="group flex items-center justify-between border-[3px] border-border-stark hard-shadow bg-surface-container-low p-5 hover:bg-primary-container"
						>
							<div>
								<p className="font-bold uppercase text-sm" style={{ fontFamily: "var(--font-space-grotesk)" }}>
									{label}
								</p>
								<p className="text-xs text-on-surface-variant mt-0.5" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
									{desc}
								</p>
							</div>
							<ChevronRight size={18} className="shrink-0 transition-transform duration-150 group-hover:translate-x-1" />
						</a>
					))}
				</div>
			</section>

			{/* Coming soon banner */}
			<section className="border-[3px] border-border-stark border-dashed p-8 flex flex-col items-center gap-3 text-center">
				<Zap size={32} className="text-on-surface-variant" />
				<p className="text-lg font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					Full Dashboard Coming Soon
				</p>
				<p className="text-sm text-on-surface-variant max-w-sm" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					Repository analytics, PR review history, and team collaboration tools are on the way.
				</p>
			</section>
		</div>
	);
}
