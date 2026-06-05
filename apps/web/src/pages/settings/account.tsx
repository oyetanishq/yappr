import { useAuthStore } from "@/store/auth";

export default function SettingsAccount() {
	const { user } = useAuthStore();

	return (
		<div className="flex flex-col gap-10 max-w-4xl">
			{/* Account section */}
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
		</div>
	);
}
