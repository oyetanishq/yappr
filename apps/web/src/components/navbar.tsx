import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router";
import { LogOut, LayoutDashboard } from "lucide-react";
import { useAuthStore } from "@/store/auth";

interface NavBarProps {}

export default function Navbar({}: NavBarProps) {
	const [scrolled, setScrolled] = useState(false);
	const { status, user, logout } = useAuthStore();
	const navigate = useNavigate();

	useEffect(() => {
		const handleScroll = () => setScrolled(window.scrollY > 20);
		window.addEventListener("scroll", handleScroll, { passive: true });
		return () => window.removeEventListener("scroll", handleScroll);
	}, []);

	const baseClasses = "flex justify-between items-center px-6 h-16 fixed z-50 bg-background border-border-stark transition-all duration-300 ease-in-out left-1/2 -translate-x-1/2";
	const scrolledClasses = "top-0 w-full max-w-full rounded-none border-b-[3px] border-t-0 border-x-0 shadow-none";
	const unscrolledClasses = "top-4 w-[calc(100%-2rem)] max-w-7xl rounded-xl border-[3px] shadow-[4px_4px_0px_0px_rgba(0,0,0,1)]";

	const handleLogout = async () => {
		await logout();
		navigate("/login", { replace: true });
	};

	return (
		<nav id="navbar" className={`${baseClasses} ${scrolled ? scrolledClasses : unscrolledClasses}`}>
			{/* Left: Logo + Nav Links */}
			<div className="flex items-center gap-6">
				<Link to="/" className="font-bold tracking-tighter uppercase text-on-surface text-2xl md:text-lg" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					YAPPR
				</Link>
			</div>

			{/* Right: auth-aware actions */}
			<div className="flex items-center gap-3">
				{status === "authenticated" && user ? (
					<>
						<Link to="/dashboard" aria-label="Dashboard" className="p-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container cursor-pointer">
							<LayoutDashboard size={18} />
						</Link>

						{/* Avatar */}
						<Link to="/dashboard" className="flex items-center gap-2 px-2 py-1.5 border-[3px] border-border-stark hard-shadow bg-surface-container hover:bg-primary-container">
							<img src={user.avatar_url} alt={user.login} className="w-6 h-6 rounded-full border-2 border-border-stark" />
							<span className="text-sm font-semibold hidden sm:block" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								{user.login}
							</span>
						</Link>

						<button
							id="navbar-logout-btn"
							onClick={handleLogout}
							className="flex items-center gap-2 px-3 py-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-error-container cursor-pointer"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							<LogOut size={16} />
						</button>
					</>
				) : status === "unauthenticated" ? (
					<Link
						id="navbar-login-btn"
						to="/login"
						className="flex items-center gap-2 px-4 py-2 border-[3px] border-border-stark hard-shadow bg-on-surface text-surface hover:bg-primary hover:text-on-primary font-bold uppercase text-sm"
						style={{ fontFamily: "var(--font-space-grotesk)" }}
					>
						Login
					</Link>
				) : (
					// loading — placeholder skeleton
					<div className="w-20 h-9 border-[3px] border-border-stark bg-surface-container-highest animate-pulse" />
				)}
			</div>
		</nav>
	);
}
