import { useEffect } from "react";
import { useNavigate } from "react-router";
import { ArrowRight, Zap, Shield, GitBranch } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { authApi } from "@/lib/api";
import { Noise } from "@/components/noise";
import Navbar from "@/components/navbar";

const features = [
	{ icon: Zap, label: "AI Code Review", desc: "Instant analysis on every PR" },
	{ icon: Shield, label: "Bug Detection", desc: "Catch issues before merge" },
	{ icon: GitBranch, label: "Architecture Mapping", desc: "Visualise your codebase" },
];

export default function LoginPage() {
	const { status } = useAuthStore();
	const navigate = useNavigate();

	// If already authenticated, bounce to dashboard
	useEffect(() => {
		if (status === "authenticated") {
			navigate("/dashboard", { replace: true });
		}
	}, [status, navigate]);

	if (status === "loading" || status === "authenticated") {
		return (
			<div className="min-h-screen flex items-center justify-center grid-bg" style={{ backgroundColor: "var(--color-background)" }}>
				<div className="w-10 h-10 border-[3px] border-border-stark border-t-primary animate-spin" />
			</div>
		);
	}

	return (
		<div className="min-h-screen flex flex-col grid-bg relative overflow-hidden" style={{ backgroundColor: "var(--color-background)", color: "var(--color-on-surface)" }}>
			<Noise />

			{/* Top bar */}
			<Navbar />

			<main className="flex-1 flex items-center justify-center px-4 py-16 pt-28 md:pt-16">
				<div className="w-full max-w-4xl grid md:grid-cols-2 gap-8 items-center">
					{/* Left: headline */}
					<div className="flex flex-col gap-6">
						<div
							className="inline-flex items-center gap-2 px-3 py-1.5 border-[3px] border-border-stark bg-primary-container hard-shadow w-fit"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							<div className="w-2 h-2 bg-border-stark rounded-full animate-pulse" />
							<span className="text-xs font-semibold text-on-primary-container uppercase tracking-widest">Beta Access</span>
						</div>

						<h1 className="text-5xl md:text-6xl font-bold leading-none uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
							Code review, <span className="bg-secondary-container px-2 border-[3px] border-border-stark hard-shadow inline-block -rotate-1">amplified</span>
						</h1>

						<p className="text-base text-on-surface-variant leading-relaxed" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							Yappr connects to your GitHub repos and brings AI-powered analysis directly into your PR workflow.
						</p>

						{/* Feature pills */}
						<div className="flex flex-col gap-3">
							{features.map(({ icon: Icon, label, desc }) => (
								<div key={label} className="flex items-center gap-4 p-3 border-[3px] border-border-stark bg-surface-container hard-shadow">
									<div className="w-9 h-9 flex items-center justify-center border-[3px] border-border-stark bg-primary-container shrink-0">
										<Icon size={16} />
									</div>
									<div>
										<p className="text-sm font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
											{label}
										</p>
										<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
											{desc}
										</p>
									</div>
								</div>
							))}
						</div>
					</div>

					{/* Right: login card */}
					<div className="flex flex-col border-[3px] border-border-stark hard-shadow bg-surface-container-low p-8 gap-6">
						<div className="flex flex-col gap-2">
							<h2 className="text-2xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
								Get started
							</h2>
							<p className="text-sm text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
								Sign in with GitHub to connect your repositories.
							</p>
						</div>

						<div className="border-t-[3px] border-border-stark" />

						<button
							id="github-login-btn"
							onClick={() => authApi.loginWithGithub()}
							className="group flex items-center justify-between w-full px-5 py-4 border-[3px] border-border-stark bg-on-surface text-surface hard-shadow hover:bg-primary hover:text-on-primary cursor-pointer"
							style={{ fontFamily: "var(--font-space-grotesk)" }}
						>
							<div className="flex items-center gap-3">
								{/* <Github size={20} /> */}
								<span className="font-bold uppercase text-sm tracking-wide">Continue with GitHub</span>
							</div>
							<ArrowRight size={16} className="transition-transform duration-200 group-hover:translate-x-1" />
						</button>

						<p className="text-xs text-on-surface-variant text-center" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							By continuing you agree to our <span className="underline cursor-pointer hover:text-on-surface">Terms of Service</span> and{" "}
							<span className="underline cursor-pointer hover:text-on-surface">Privacy Policy</span>.
						</p>
					</div>
				</div>
			</main>
		</div>
	);
}
