import { useNavigate, Link } from "react-router";
import { LogOut, GitBranch, Star, Zap, Shield, Settings, ChevronRight, Plus, FolderGit2 } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { useMe, useInstallations } from "@/lib/hooks";
import { githubApi } from "@/lib/api";
import { Noise } from "@/components/noise";
import { InstallationCard } from "@/pages/dashboard/installation-card";

const otherActions = [
	{ label: "Configure Review Rules", desc: "Customise what Yappr checks for", href: "#" },
	{ label: "Invite Team Members", desc: "Collaborate with your team", href: "#" },
];

export default function DashboardPage() {
	const { logout } = useAuthStore();
	const { data: user } = useMe();
	const { data: installations = [], isLoading: installationsLoading } = useInstallations();
	const navigate = useNavigate();

	const handleLogout = async () => {
		await logout();
		navigate("/login", { replace: true });
	};

	const handleConnectRepo = () => githubApi.install();

	const statCards = [
		{ label: "PRs Reviewed", value: "—", icon: GitBranch },
		{ label: "Issues Found", value: "—", icon: Shield },
		{
			label: "Repos Connected",
			value: installationsLoading ? "…" : String(installations.length), // Note: This shows installations count for now
			icon: Star,
		},
		{ label: "AI Reviews", value: "—", icon: Zap },
	];

	return (
		<div className="min-h-screen flex flex-col grid-bg relative" style={{ backgroundColor: "var(--color-background)", color: "var(--color-on-surface)" }}>
			<Noise />

			{/* Top bar */}
			<header className="flex items-center justify-between px-6 h-16 border-b-[3px] border-border-stark bg-background/80 backdrop-blur-sm sticky top-0 z-40">
				<div className="flex items-center gap-6">
					<span className="font-bold tracking-tighter uppercase text-on-surface text-xl" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						YAPPR
					</span>
					<div
						className="hidden md:flex items-center gap-1 px-2 py-1 border-[3px] border-border-stark bg-primary-container text-on-primary-container text-xs font-semibold uppercase"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						Dashboard
					</div>
				</div>

				<div className="flex items-center gap-3">
					<Link
						to="/settings"
						id="dashboard-settings-btn"
						aria-label="Settings"
						className="p-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container cursor-pointer"
					>
						<Settings size={18} />
					</Link>

					{/* User avatar + name */}
					<div className="flex items-center gap-2 px-3 py-1.5 border-[3px] border-border-stark bg-surface-container hard-shadow">
						{user?.avatar_url ? (
							<img src={user.avatar_url} alt={user.login} className="w-6 h-6 rounded-full border-2 border-border-stark" />
						) : (
							<div className="w-6 h-6 rounded-full border-2 border-border-stark bg-primary-container" />
						)}
						<span className="text-sm font-semibold hidden sm:block" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							{user?.login ?? "—"}
						</span>
					</div>

					<button
						id="logout-btn"
						onClick={handleLogout}
						className="flex items-center gap-2 px-3 py-2 border-[3px] border-border-stark hard-shadow bg-error-container text-on-error-container hover:bg-error hover:text-on-error cursor-pointer"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						<LogOut size={16} />
						<span className="text-xs font-semibold uppercase hidden sm:block">Logout</span>
					</button>
				</div>
			</header>

			<main className="flex-1 px-6 py-10 max-w-6xl mx-auto w-full flex flex-col gap-10">
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
						Overview
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

				{/* Connected Repositories */}
				<section>
					<div className="flex items-center justify-between mb-4">
						<h2 className="text-xs uppercase tracking-widest text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							Connected Repositories
						</h2>
						<button
							id="connect-repo-btn"
							onClick={handleConnectRepo}
							className="flex items-center gap-2 px-3 py-1.5 border-[3px] border-border-stark hard-shadow bg-on-surface text-surface hover:bg-primary hover:text-on-primary cursor-pointer"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							<Plus size={14} />
							<span className="text-xs font-semibold uppercase">Connect Repo</span>
						</button>
					</div>

					{/* Loading */}
					{installationsLoading && (
						<div className="flex flex-col gap-3">
							{[1, 2].map((i) => (
								<div key={i} className="border-[3px] border-border-stark bg-surface-container p-5 animate-pulse h-16" />
							))}
						</div>
					)}

					{/* Empty state */}
					{!installationsLoading && installations.length === 0 && (
						<div className="border-[3px] border-border-stark border-dashed p-10 flex flex-col items-center gap-4 text-center">
							<div className="w-12 h-12 border-[3px] border-border-stark bg-surface-container flex items-center justify-center">
								<FolderGit2 size={24} className="text-on-surface-variant" />
							</div>
							<div>
								<p className="text-sm font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
									No Repositories Connected
								</p>
								<p className="text-xs text-on-surface-variant mt-1 max-w-xs" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
									Connect your first GitHub repository to start getting AI-powered code reviews on every PR.
								</p>
							</div>
							<button
								id="connect-repo-empty-btn"
								onClick={handleConnectRepo}
								className="flex items-center gap-2 px-5 py-2.5 border-[3px] border-border-stark hard-shadow bg-on-surface text-surface hover:bg-primary hover:text-on-primary cursor-pointer"
								style={{ fontFamily: "var(--font-jetbrains-mono)" }}
							>
								<Plus size={16} />
								<span className="text-xs font-bold uppercase">Connect a Repository</span>
							</button>
						</div>
					)}

					{/* Installations list */}
					{!installationsLoading && installations.length > 0 && (
						<div className="flex flex-col gap-3">
							{installations.map((inst) => (
								<InstallationCard key={inst.id} inst={inst} />
							))}

							{/* Add more */}
							<button
								onClick={handleConnectRepo}
								className="flex items-center justify-center gap-2 px-4 py-3 border-[3px] border-border-stark border-dashed bg-surface-container-low hover:bg-surface-container cursor-pointer w-full mt-2"
								style={{ fontFamily: "var(--font-jetbrains-mono)" }}
							>
								<Plus size={14} className="text-on-surface-variant" />
								<span className="text-xs font-semibold uppercase text-on-surface-variant">Add Another Repository</span>
							</button>
						</div>
					)}
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
			</main>
		</div>
	);
}
