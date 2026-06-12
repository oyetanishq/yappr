import { useAuthStore } from "@/store/auth";
import { useSubscribe, useCancelSubscription } from "@/lib/hooks";
import { Zap, CheckCircle2, AlertCircle, Loader2 } from "lucide-react";

export default function SettingsBilling() {
	const { user, fetchUser } = useAuthStore();
	const { subscribe, isPending: subscribePending } = useSubscribe();
	const { cancel, isPending: cancelPending, isSuccess: cancelSuccess } = useCancelSubscription();

	const isPro = user?.plan === "pro";
	const isCancelled = cancelSuccess || user?.cancel_at_period_end;
	const prLimit = 10;
	const prCount = user?.pr_count_this_month || 0;
	const prUsagePercent = Math.min((prCount / prLimit) * 100, 100);

	const handleCancel = () => {
		cancel({
			onSuccess: () => {
				fetchUser();
			},
		});
	};

	return (
		<div className="flex flex-col gap-10 max-w-4xl" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
			{/* Current Plan Section */}
			<section>
				<h2 className="text-xs uppercase tracking-widest text-on-surface-variant mb-4">Current Plan</h2>
				<div className="border-[3px] border-border-stark hard-shadow bg-surface-container p-6 flex flex-col md:flex-row md:items-center justify-between gap-6">
					<div className="flex flex-col gap-2">
						<div className="flex items-center gap-3">
							<div
								className={`px-3 py-1 border-[3px] border-border-stark font-bold uppercase tracking-widest text-sm ${isPro ? "bg-primary text-on-primary" : "bg-surface-container-highest text-on-surface"}`}
							>
								{isPro ? "Pro Plan" : "Free Plan"}
							</div>
							{isPro && user?.plan_expires_at && (
								<p className="text-xs text-on-surface-variant">
									{isCancelled ? "Expires on " : "Renews on "}
									{new Date(user.plan_expires_at).toLocaleDateString()}
								</p>
							)}
						</div>
						<p className="text-sm text-on-surface-variant">{isPro ? "You have access to all premium features." : "Upgrade to Pro to unlock unlimited PRs and custom personalities."}</p>
					</div>

					{!isPro && (
						<button
							onClick={() => subscribe()}
							disabled={subscribePending}
							className="flex items-center justify-center gap-2 px-6 py-3 border-[3px] border-border-stark hard-shadow bg-on-surface text-surface hover:bg-primary hover:text-on-primary font-bold uppercase text-sm cursor-pointer transition-all min-w-[200px]"
						>
							{subscribePending ? <Loader2 size={16} className="animate-spin" /> : <Zap size={16} />}
							{subscribePending ? "Redirecting..." : "Upgrade to Pro"}
						</button>
					)}
					{isPro && (
						<div className="flex flex-col items-end gap-2">
							<button
								onClick={handleCancel}
								disabled={cancelPending || isCancelled}
								className={`flex items-center justify-center gap-2 px-6 py-3 border-[3px] border-border-stark hard-shadow font-bold uppercase text-sm transition-all min-w-[200px] ${
									isCancelled ? "bg-surface-container-highest text-on-surface-variant cursor-not-allowed" : "bg-surface text-error hover:bg-error hover:text-on-error cursor-pointer"
								}`}
							>
								{cancelPending ? <Loader2 size={16} className="animate-spin" /> : null}
								{isCancelled ? "Cancelled" : "Cancel Subscription"}
							</button>
							{isCancelled && <p className="text-[10px] text-error">Your subscription will end at the current billing cycle.</p>}
						</div>
					)}
				</div>
			</section>

			{/* Usage Section */}
			<section>
				<h2 className="text-xs uppercase tracking-widest text-on-surface-variant mb-4">Monthly Usage</h2>
				<div className="border-[3px] border-border-stark hard-shadow bg-surface-container-low p-6">
					<div className="flex items-center justify-between mb-3">
						<h3 className="font-bold uppercase text-sm">PR Reviews</h3>
						<span className="text-sm font-bold">{isPro ? "Unlimited" : `${prCount} / ${prLimit}`}</span>
					</div>
					{!isPro && (
						<>
							<div className="w-full h-3 border-2 border-border-stark bg-surface-container-highest overflow-hidden">
								<div className={`h-full transition-all duration-500 ${prUsagePercent >= 100 ? "bg-error" : "bg-primary"}`} style={{ width: `${prUsagePercent}%` }} />
							</div>
							<p className="text-[10px] text-on-surface-variant mt-2 uppercase tracking-wide">Resets on the 1st of every month.</p>
						</>
					)}
					{isPro && (
						<div className="flex items-center gap-2 text-primary mt-2">
							<CheckCircle2 size={16} />
							<span className="text-xs uppercase tracking-widest font-bold">Unlimited PRs Unlocked</span>
						</div>
					)}
				</div>
			</section>

			{/* Features Section */}
			<section>
				<h2 className="text-xs uppercase tracking-widest text-on-surface-variant mb-4">Pro Features</h2>
				<div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<FeatureCard active={isPro} title="Unlimited PRs" desc="No monthly limits on code reviews." />
					<FeatureCard active={isPro} title="All Personalities" desc="Unlock Bestie, Sigma, and Toxic Tech Lead." />
					<FeatureCard active={isPro} title="Architecture Mapping" desc="Get deep architectural insights in PR comments." />
					<FeatureCard active={isPro} title="Priority Support" desc="Get help faster when you need it." />
				</div>
			</section>
		</div>
	);
}

function FeatureCard({ active, title, desc }: { active: boolean; title: string; desc: string }) {
	return (
		<div className={`p-5 border-[3px] border-border-stark ${active ? "bg-primary-container" : "bg-surface-container opacity-60"}`}>
			<div className="flex items-start gap-3 mb-2">
				{active ? <CheckCircle2 size={18} className="text-primary shrink-0 mt-0.5" /> : <AlertCircle size={18} className="text-on-surface-variant shrink-0 mt-0.5" />}
				<h4 className="font-bold uppercase text-sm" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					{title}
				</h4>
			</div>
			<p className="text-xs text-on-surface-variant ml-7 leading-relaxed">{desc}</p>
		</div>
	);
}
