import { useState, useEffect, useMemo } from "react";
import { useNavigate, Link } from "react-router";
import { LogOut, Settings, Trash2, Monitor, RefreshCw, AlertTriangle, ChevronLeft } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { useSessions, useRevokeSession } from "@/lib/hooks";
import { type Session } from "@/lib/api";
import { Noise } from "@/components/noise";

function formatDate(iso: string) {
	return new Intl.DateTimeFormat("en-US", {
		dateStyle: "medium",
		timeStyle: "short",
	}).format(new Date(iso));
}

function isExpired(iso: string) {
	return new Date(iso) < new Date();
}

function SessionItem({ session, isCurrent, isRevoking, onRevoke }: { session: Session; isCurrent: boolean; isRevoking: boolean; onRevoke: (id: string) => void }) {
	const [location, setLocation] = useState<string | null>(null);

	useEffect(() => {
		if (!session.ip) return;

		if (session.ip === "127.0.0.1" || session.ip === "::1" || session.ip.startsWith("192.168.") || session.ip.startsWith("10.") || session.ip.startsWith("172.")) {
			setLocation("Local Network");
			return;
		}

		fetch(`https://ip-api.com/json/${session.ip}?fields=city,country`)
			.then((res) => res.json())
			.then((data) => {
				if (data.city && data.country) {
					setLocation(`${data.city}, ${data.country}`);
				} else {
					setLocation("Unknown Location");
				}
			})
			.catch(() => setLocation("Unknown Location"));
	}, [session.ip]);

	// Format User-Agent
	const ua = session.user_agent || "";
	// Basic parser
	const browserMatch = ua.match(/(firefox|msie|chrome|safari|trident|edge|edg)\/?\s*(\d+)/i);
	const osMatch = ua.match(/(mac os x|windows nt|linux|android|iphone|ipad)/i);

	const browser = browserMatch ? browserMatch[1] : "";
	const os = osMatch ? osMatch[1] : "";

	const deviceStr = browser || os ? `${browser} on ${os}` : ua || "Unknown Device";

	return (
		<div
			id={`session-${session.id}`}
			className={`border-[3px] border-border-stark hard-shadow p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4 ${isCurrent ? "bg-primary-container" : "bg-surface-container"}`}
		>
			<div className="flex items-start gap-4">
				<div className={`w-10 h-10 flex items-center justify-center border-[3px] border-border-stark shrink-0 ${isCurrent ? "bg-on-surface text-surface" : "bg-surface-container-highest"}`}>
					<Monitor size={16} />
				</div>
				<div className="min-w-0">
					<div className="flex flex-wrap items-center gap-2 mb-1">
						<span className="text-sm font-bold uppercase truncate max-w-[200px] sm:max-w-none" style={{ fontFamily: "var(--font-space-grotesk)" }}>
							{deviceStr}
						</span>
						{isCurrent && (
							<span
								className="px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-widest bg-on-surface text-surface border border-border-stark"
								style={{ fontFamily: "var(--font-jetbrains-mono)" }}
							>
								Current
							</span>
						)}
					</div>

					<div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-on-surface-variant mb-2" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						<span>
							<span className="opacity-60">IP: </span>
							{session.ip || "Unknown"}
						</span>
						{location && (
							<span>
								<span className="opacity-60">Loc: </span>
								{location}
							</span>
						)}
					</div>

					<div className="flex flex-wrap gap-x-4 gap-y-0.5 text-[10px] text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						<span>
							<span className="opacity-60">Created: </span>
							{formatDate(session.created_at)}
						</span>
						<span>
							<span className="opacity-60">Expires: </span>
							{formatDate(session.expires_at)}
						</span>
					</div>
				</div>
			</div>

			<button
				id={`revoke-session-${session.id}`}
				onClick={() => onRevoke(session.id)}
				disabled={isRevoking}
				className={`flex items-center gap-2 px-3 py-2 border-[3px] border-border-stark hard-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed shrink-0 ${
					isCurrent ? "bg-error-container text-on-error-container hover:bg-error hover:text-on-error" : "bg-surface-container-highest hover:bg-error-container hover:text-on-error-container"
				}`}
				style={{ fontFamily: "var(--font-jetbrains-mono)" }}
			>
				{isRevoking ? <RefreshCw size={14} className="animate-spin" /> : <Trash2 size={14} />}
				<span className="text-xs font-semibold uppercase">{isCurrent ? "Sign Out" : "Revoke"}</span>
			</button>
		</div>
	);
}

export default function SettingsPage() {
	const { user, logout } = useAuthStore();
	const navigate = useNavigate();

	const { data: allSessions = [], isLoading, isError, refetch, isFetching } = useSessions();
	const { mutate: revoke, isPending: isRevokePending, variables: revokingId } = useRevokeSession({ onSuccess: refetch });

	const activeSessions = allSessions.filter((s) => !isExpired(s.expires_at));

	// The current session is explicitly marked by the backend
	const currentSessionId = useMemo(() => {
		const current = activeSessions.find((s) => s.is_current);
		return current ? current.id : null;
	}, [activeSessions]);

	const handleRevoke = (id: string) => {
		if (id === currentSessionId) {
			revoke(id, {
				onSuccess: async () => {
					await logout();
					navigate("/login", { replace: true });
				},
			});
		} else {
			revoke(id);
		}
	};

	const handleLogout = async () => {
		await logout();
		navigate("/login", { replace: true });
	};

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
						className="hidden md:flex items-center gap-1 px-2 py-1 border-[3px] border-border-stark bg-surface-container-highest text-on-surface text-xs font-semibold uppercase"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						<Settings size={12} />
						<span>Settings</span>
					</div>
				</div>

				<div className="flex items-center gap-3">
					<Link
						to="/dashboard"
						className="flex items-center gap-2 px-3 py-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container cursor-pointer text-sm"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						<ChevronLeft size={14} />
						<span className="hidden sm:block">Dashboard</span>
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
						id="settings-logout-btn"
						onClick={handleLogout}
						className="flex items-center gap-2 px-3 py-2 border-[3px] border-border-stark hard-shadow bg-error-container text-on-error-container hover:bg-error hover:text-on-error cursor-pointer"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						<LogOut size={16} />
						<span className="text-xs font-semibold uppercase hidden sm:block">Logout</span>
					</button>
				</div>
			</header>

			<main className="flex-1 px-6 py-10 max-w-4xl mx-auto w-full flex flex-col gap-10">
				{/* Page title */}
				<div>
					<h1 className="text-4xl font-bold uppercase tracking-tight" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						Settings
					</h1>
					<p className="text-sm text-on-surface-variant mt-1" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						Manage your account and active sessions.
					</p>
				</div>

				{/* ── Sessions section ────────────────────────────────────────────────── */}
				<section>
					<div className="flex items-center justify-between mb-4">
						<div>
							<h2 className="text-xs uppercase tracking-widest text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								Active Sessions
							</h2>
							<p className="text-xs text-on-surface-variant mt-0.5" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								{!isLoading && `${activeSessions.length} active session${activeSessions.length !== 1 ? "s" : ""}`}
							</p>
						</div>
						<button
							id="sessions-refresh-btn"
							onClick={() => refetch()}
							disabled={isFetching}
							className="flex items-center gap-2 px-3 py-1.5 border-[3px] border-border-stark hard-shadow bg-surface-container hover:bg-primary-container cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							<RefreshCw size={14} className={isFetching ? "animate-spin" : ""} />
							<span className="text-xs font-semibold uppercase">Refresh</span>
						</button>
					</div>

					{/* Error state */}
					{isError && (
						<div className="flex items-center gap-3 border-[3px] border-error bg-error-container text-on-error-container p-4 mb-4">
							<AlertTriangle size={16} className="shrink-0" />
							<p className="text-sm" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								Failed to load sessions. Please try again.
							</p>
						</div>
					)}

					{/* Loading skeleton */}
					{isLoading && (
						<div className="flex flex-col gap-3">
							{[1, 2, 3].map((i) => (
								<div key={i} className="border-[3px] border-border-stark bg-surface-container p-5 animate-pulse h-20" />
							))}
						</div>
					)}

					{/* Empty state */}
					{!isLoading && activeSessions.length === 0 && !isError && (
						<div className="border-[3px] border-border-stark border-dashed p-10 flex flex-col items-center gap-3 text-center">
							<Monitor size={32} className="text-on-surface-variant" />
							<p className="text-sm font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
								No Active Sessions
							</p>
							<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								You have no currently active sessions.
							</p>
						</div>
					)}

					{/* Session list */}
					{!isLoading && activeSessions.length > 0 && (
						<div className="flex flex-col gap-3">
							{activeSessions.map((session) => (
								<SessionItem
									key={session.id}
									session={session}
									isCurrent={session.id === currentSessionId}
									isRevoking={isRevokePending && revokingId === session.id}
									onRevoke={handleRevoke}
								/>
							))}
						</div>
					)}

					{/* Revoke all other sessions */}
					{!isLoading && activeSessions.filter((s) => s.id !== currentSessionId).length > 0 && (
						<div className="mt-4 flex justify-end">
							<button
								id="revoke-all-sessions-btn"
								onClick={() => {
									activeSessions.filter((s) => s.id !== currentSessionId).forEach((s) => revoke(s.id));
								}}
								disabled={isRevokePending}
								className="flex items-center gap-2 px-4 py-2 border-[3px] border-border-stark hard-shadow bg-error-container text-on-error-container hover:bg-error hover:text-on-error cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
								style={{ fontFamily: "var(--font-jetbrains-mono)" }}
							>
								<Trash2 size={14} />
								<span className="text-xs font-semibold uppercase">Revoke All Other Sessions</span>
							</button>
						</div>
					)}
				</section>

				{/* ── Account section ───────────────────────────────────────────────────── */}
				<section>
					<h2 className="text-xs uppercase tracking-widest text-on-surface-variant mb-4" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						Account
					</h2>
					<div className="border-[3px] border-border-stark hard-shadow bg-surface-container p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
						<div className="flex items-center gap-4">
							{user?.avatar_url ? (
								<img src={user.avatar_url} alt={user.login} className="w-14 h-14 border-[3px] border-border-stark" />
							) : (
								<div className="w-14 h-14 border-[3px] border-border-stark bg-primary-container" />
							)}
							<div>
								<p className="text-base font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
									{user?.name || user?.login}
								</p>
								<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
									@{user?.login}
								</p>
								{user?.email && (
									<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
										{user.email}
									</p>
								)}
							</div>
						</div>
						<a
							href={`https://github.com/${user?.login}`}
							target="_blank"
							rel="noopener noreferrer"
							className="flex items-center gap-2 px-3 py-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container cursor-pointer text-xs font-semibold uppercase"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							View on GitHub
						</a>
					</div>
				</section>
			</main>
		</div>
	);
}
