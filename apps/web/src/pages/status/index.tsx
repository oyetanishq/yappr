import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { Activity, CheckCircle2, XCircle, RefreshCw, Loader2, Database, Server, Bot } from "lucide-react";
import Navbar from "@/components/navbar";
import Footer from "@/components/footer";
import { Noise } from "@/components/noise";
import { statusApi, type HealthResponse, type ServiceHealth } from "@/lib/api";

interface ServiceMeta {
	key: "redis" | "mongo" | "agent";
	label: string;
	description: string;
	icon: typeof Server;
}

const SERVICES: ServiceMeta[] = [
	{ key: "redis", label: "Redis", description: "Sessions, caching & rate limits", icon: Server },
	{ key: "mongo", label: "MongoDB", description: "Users, installations & repo config", icon: Database },
	{ key: "agent", label: "Review Agent", description: "PR review & bug-detection worker", icon: Bot },
];

export default function StatusPage() {
	const [health, setHealth] = useState<HealthResponse | null>(null);
	const [loading, setLoading] = useState(true);
	const [failed, setFailed] = useState(false);

	const load = useCallback(async () => {
		setLoading(true);
		setFailed(false);
		try {
			setHealth(await statusApi.health());
		} catch {
			// The API itself is unreachable — treat everything as down.
			setHealth(null);
			setFailed(true);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		load();
	}, [load]);

	// The API being unreachable is itself an outage; surface it as "degraded".
	const overall: "ok" | "degraded" | null = failed ? "degraded" : (health?.status ?? null);

	return (
		<div className="min-h-screen flex flex-col relative grid-bg" style={{ backgroundColor: "var(--color-background)", color: "var(--color-on-surface)" }}>
			<Navbar />
			<Noise />

			<main className="grow pt-28 px-6 pb-10 md:pt-36">
				<section className="max-w-4xl mx-auto">
					{/* Header */}
					<div className="flex flex-col sm:flex-row sm:items-center gap-4 justify-between mb-8">
						<div className="flex items-center gap-4">
							<span className="p-3 border-[3px] border-border-stark hard-shadow bg-primary text-on-primary">
								<Activity size={32} />
							</span>
							<h1 className="text-4xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
								System Status
							</h1>
						</div>
						<button
							onClick={load}
							disabled={loading}
							className="flex items-center justify-center gap-2 px-4 py-3 border-[3px] border-border-stark hard-shadow bg-surface hover:bg-surface-container-highest font-bold uppercase text-[12px] tracking-wider cursor-pointer disabled:opacity-60 disabled:cursor-not-allowed"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							{loading ? <Loader2 size={16} className="animate-spin" /> : <RefreshCw size={16} />}
							Refresh
						</button>
					</div>

					{/* Overall banner */}
					<OverallBanner overall={overall} loading={loading && !health} />

					{/* Per-service grid */}
					<div className="grid grid-cols-1 gap-4">
						{SERVICES.map((svc) => (
							<ServiceRow key={svc.key} meta={svc} health={failed ? { status: "down", latency_ms: 0, error: "API unreachable" } : health?.services[svc.key]} loading={loading && !health} />
						))}
					</div>

					<p className="text-[11px] text-on-surface-variant mt-6 uppercase tracking-wide" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						Live probe of core infrastructure. Not seeing what you expect? <Link to="/" className="underline hover:text-primary">Back to home</Link>.
					</p>
				</section>
			</main>

			<Footer />
		</div>
	);
}

function OverallBanner({ overall, loading }: { overall: "ok" | "degraded" | null; loading: boolean }) {
	const isOk = overall === "ok";
	const bg = loading ? "bg-surface-container-highest" : isOk ? "bg-primary-container" : "bg-error-container";

	return (
		<div className={`border-[3px] border-border-stark hard-shadow p-6 mb-8 flex items-center gap-4 ${bg}`} style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
			{loading ? (
				<Loader2 size={28} className="animate-spin shrink-0" />
			) : isOk ? (
				<CheckCircle2 size={28} className="text-primary shrink-0" />
			) : (
				<XCircle size={28} className="text-error shrink-0" />
			)}
			<div>
				<p className="font-bold uppercase text-lg" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					{loading ? "Checking systems…" : isOk ? "All systems operational" : "Degraded — some systems down"}
				</p>
				<p className="text-[12px] text-on-surface-variant uppercase tracking-wide">{loading ? "Probing dependencies" : isOk ? "Everything is running normally" : "One or more dependencies are unreachable"}</p>
			</div>
		</div>
	);
}

function ServiceRow({ meta, health, loading }: { meta: ServiceMeta; health?: ServiceHealth; loading: boolean }) {
	const Icon = meta.icon;
	const isOk = health?.status === "ok";

	return (
		<div className="border-[3px] border-border-stark hard-shadow bg-surface p-5 flex items-center gap-4" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
			<span className="p-2.5 border-[3px] border-border-stark bg-surface-container-highest shrink-0">
				<Icon size={22} />
			</span>

			<div className="grow min-w-0">
				<h3 className="font-bold uppercase text-sm" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					{meta.label}
				</h3>
				<p className="text-[11px] text-on-surface-variant uppercase tracking-wide truncate">{meta.description}</p>
			</div>

			{/* Latency */}
			{!loading && health && (
				<span className="hidden sm:block text-[11px] text-on-surface-variant uppercase tracking-wide shrink-0">{health.latency_ms}ms</span>
			)}

			{/* Status pill */}
			<span
				className={`flex items-center gap-2 px-3 py-1.5 border-[3px] border-border-stark font-bold uppercase text-[12px] shrink-0 ${
					loading ? "bg-surface-container-highest" : isOk ? "bg-primary text-on-primary" : "bg-error text-on-error"
				}`}
			>
				{loading ? (
					<>
						<Loader2 size={14} className="animate-spin" /> Checking
					</>
				) : isOk ? (
					<>
						<CheckCircle2 size={14} /> Operational
					</>
				) : (
					<>
						<XCircle size={14} /> Down
					</>
				)}
			</span>
		</div>
	);
}
