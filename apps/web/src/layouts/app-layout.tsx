import { useNavigate, Link, Outlet, useLocation } from "react-router";
import { LogOut, Settings, LayoutDashboard } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { Noise } from "@/components/noise";

export default function AppLayout() {
	const { user, logout } = useAuthStore();
	const navigate = useNavigate();
	const location = useLocation();

	const handleLogout = async () => {
		await logout();
		navigate("/login", { replace: true });
	};

	const isSettings = location.pathname.startsWith("/settings");

	return (
		<div className="min-h-screen flex flex-col grid-bg relative" style={{ backgroundColor: "var(--color-background)", color: "var(--color-on-surface)" }}>
			<Noise />

			{/* Top bar */}
			<header className="flex items-center justify-between px-6 h-16 border-b-[3px] border-border-stark bg-background/80 backdrop-blur-sm sticky top-0 z-40">
				<div className="flex items-center gap-6">
					<Link to="/dashboard" className="font-bold tracking-tighter uppercase text-on-surface text-xl hover:opacity-80" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						YAPPR
					</Link>
					<div
						className="hidden md:flex items-center gap-1 px-2 py-1 border-[3px] border-border-stark bg-primary-container text-on-primary-container text-xs font-semibold uppercase"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						{isSettings ? "Settings" : "Dashboard"}
					</div>
				</div>

				<div className="flex items-center gap-3">
					<Link
						to="/dashboard"
						aria-label="Dashboard"
						className={`p-2 border-[3px] border-border-stark hard-shadow cursor-pointer ${!isSettings ? "bg-primary-container" : "bg-surface-container-highest hover:bg-primary-container"}`}
					>
						<LayoutDashboard size={18} />
					</Link>

					<Link
						to="/settings"
						aria-label="Settings"
						className={`p-2 border-[3px] border-border-stark hard-shadow cursor-pointer ${isSettings ? "bg-primary-container" : "bg-surface-container-highest hover:bg-primary-container"}`}
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

			{/* Main Content */}
			<div className="flex-1 flex flex-col max-w-7xl mx-auto w-full">
				<Outlet />
			</div>
		</div>
	);
}
