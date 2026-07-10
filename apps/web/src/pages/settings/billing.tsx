import { useEffect, useState } from "react";
import { useAuthStore } from "@/store/auth";
import { useSubscribe, useCancelSubscription, useResumeSubscription } from "@/lib/hooks";
import { isProActive } from "@/lib/api";
import { Zap, CheckCircle2, AlertCircle, Loader2, RefreshCw } from "lucide-react";

export default function SettingsBilling() {
	const { user, fetchUser } = useAuthStore();
	const { subscribe, isPending: subscribePending, error: subscribeError } = useSubscribe();
	const { cancel, isPending: cancelPending, error: cancelError } = useCancelSubscription();
	const { resume, isPending: resumePending, error: resumeError } = useResumeSubscription();

	const [confirmingCancel, setConfirmingCancel] = useState(false);

	// Pro status mirrors the backend: "pro" AND not expired. Checking expiry here is
	// what lets a lapsed user (whose plan still reads "pro" until the webhook lands)
	// fall back to the Free state and get a working "Resubscribe" button.
	const isActivePro = isProActive(user);
	const isCancelling = isActivePro && !!user?.cancel_at_period_end;
	const hasLapsed = user?.plan === "pro" && !isActivePro;
	const prLimit = 10;
	const prCount = user?.pr_count_this_month || 0;
	const prUsagePercent = Math.min((prCount / prLimit) * 100, 100);

	const actionError = subscribeError || cancelError || resumeError;

	// When the user returns from the Razorpay checkout tab (or re-focuses the page),
	// re-sync their plan so the UI reflects a subscription/cancellation that the
	// background webhook activated while they were away.
	useEffect(() => {
		const syncOnVisible = () => {
			if (document.visibilityState === "visible") fetchUser();
		};
		window.addEventListener("focus", syncOnVisible);
		document.addEventListener("visibilitychange", syncOnVisible);
		return () => {
			window.removeEventListener("focus", syncOnVisible);
			document.removeEventListener("visibilitychange", syncOnVisible);
		};
	}, [fetchUser]);

	const handleCancel = () => {
		setConfirmingCancel(false);
		cancel({ onSuccess: fetchUser });
	};

	const handleResume = () => {
		resume({ onSuccess: fetchUser });
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
								className={`px-3 py-1 border-[3px] border-border-stark font-bold uppercase tracking-widest text-sm ${isActivePro ? "bg-primary text-on-primary" : "bg-surface-container-highest text-on-surface"}`}
							>
								{isActivePro ? "Pro Plan" : "Free Plan"}
							</div>
							{isActivePro && user?.plan_expires_at && (
								<p className="text-xs text-on-surface-variant">
									{isCancelling ? "Access ends on " : "Renews on "}
									{new Date(user.plan_expires_at).toLocaleDateString()}
								</p>
							)}
						</div>
						<p className="text-sm text-on-surface-variant">
							{isCancelling
								? "Your Pro plan is scheduled to end. Resume anytime before then to keep your features."
								: isActivePro
									? "You have access to all premium features."
									: "Upgrade to Pro to unlock unlimited PRs and custom personalities."}
						</p>
					</div>

					<div className="flex flex-col items-stretch md:items-end gap-2 md:min-w-[220px]">
						{/* Free or lapsed → (re)subscribe */}
						{!isActivePro && (
							<button
								onClick={() => subscribe()}
								disabled={subscribePending}
								className="flex items-center justify-center gap-2 px-6 py-3 border-[3px] border-border-stark hard-shadow bg-on-surface text-surface hover:bg-primary hover:text-on-primary font-bold uppercase text-sm cursor-pointer transition-all disabled:opacity-60 disabled:cursor-not-allowed min-w-[200px]"
							>
								{subscribePending ? <Loader2 size={16} className="animate-spin" /> : <Zap size={16} />}
								{subscribePending ? "Redirecting..." : hasLapsed ? "Resubscribe" : "Upgrade to Pro"}
							</button>
						)}

						{/* Active + cancellation scheduled → resume */}
						{isCancelling && (
							<>
								<button
									onClick={handleResume}
									disabled={resumePending}
									className="flex items-center justify-center gap-2 px-6 py-3 border-[3px] border-border-stark hard-shadow bg-on-surface text-surface hover:bg-primary hover:text-on-primary font-bold uppercase text-sm cursor-pointer transition-all disabled:opacity-60 disabled:cursor-not-allowed min-w-[200px]"
								>
									{resumePending ? <Loader2 size={16} className="animate-spin" /> : <RefreshCw size={16} />}
									{resumePending ? "Resuming..." : "Resume Subscription"}
								</button>
								<p className="text-[10px] text-error text-right">Scheduled to end at the current billing cycle.</p>
							</>
						)}

						{/* Active, not cancelling → cancel with a two-step confirm */}
						{isActivePro && !isCancelling && !confirmingCancel && (
							<button
								onClick={() => setConfirmingCancel(true)}
								disabled={cancelPending}
								className="flex items-center justify-center gap-2 px-6 py-3 border-[3px] border-border-stark hard-shadow bg-surface text-error hover:bg-error hover:text-on-error font-bold uppercase text-sm cursor-pointer transition-all disabled:opacity-60 disabled:cursor-not-allowed min-w-[200px]"
							>
								Cancel Subscription
							</button>
						)}
						{isActivePro && !isCancelling && confirmingCancel && (
							<div className="flex flex-col items-stretch gap-2 min-w-[200px]">
								<p className="text-[11px] text-on-surface-variant md:text-right">Cancel Pro? You keep access until the end of your billing cycle.</p>
								<div className="flex gap-2">
									<button
										onClick={handleCancel}
										disabled={cancelPending}
										className="flex-1 flex items-center justify-center gap-2 px-4 py-3 border-[3px] border-border-stark hard-shadow bg-error text-on-error font-bold uppercase text-sm cursor-pointer transition-all disabled:opacity-60 disabled:cursor-not-allowed"
									>
										{cancelPending ? <Loader2 size={16} className="animate-spin" /> : null}
										Yes, cancel
									</button>
									<button
										onClick={() => setConfirmingCancel(false)}
										disabled={cancelPending}
										className="flex-1 flex items-center justify-center px-4 py-3 border-[3px] border-border-stark hard-shadow bg-surface text-on-surface hover:bg-surface-container-highest font-bold uppercase text-sm cursor-pointer transition-all"
									>
										Keep Pro
									</button>
								</div>
							</div>
						)}

						{actionError && <p className="text-[11px] text-error md:text-right">{actionError}</p>}
					</div>
				</div>
			</section>

			{/* Usage Section */}
			<section>
				<h2 className="text-xs uppercase tracking-widest text-on-surface-variant mb-4">Monthly Usage</h2>
				<div className="border-[3px] border-border-stark hard-shadow bg-surface-container-low p-6">
					<div className="flex items-center justify-between mb-3">
						<h3 className="font-bold uppercase text-sm">PR Reviews</h3>
						<span className="text-sm font-bold">{isActivePro ? "Unlimited" : `${prCount} / ${prLimit}`}</span>
					</div>
					{!isActivePro && (
						<>
							<div className="w-full h-3 border-2 border-border-stark bg-surface-container-highest overflow-hidden">
								<div className={`h-full transition-all duration-500 ${prUsagePercent >= 100 ? "bg-error" : "bg-primary"}`} style={{ width: `${prUsagePercent}%` }} />
							</div>
							<p className="text-[10px] text-on-surface-variant mt-2 uppercase tracking-wide">Resets on the 1st of every month.</p>
						</>
					)}
					{isActivePro && (
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
					<FeatureCard active={isActivePro} title="Unlimited PRs" desc="No monthly limits on code reviews." />
					<FeatureCard active={isActivePro} title="All Personalities" desc="Unlock Bestie, Sigma, and Toxic Tech Lead." />
					<FeatureCard active={isActivePro} title="Architecture Mapping" desc="Get deep architectural insights in PR comments." />
					<FeatureCard active={isActivePro} title="Priority Support" desc="Get help faster when you need it." />
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
