import { Plus, FolderGit2 } from "lucide-react";
import { useInstallations } from "@/lib/hooks";
import { githubApi } from "@/lib/api";
import { InstallationCard } from "./installation-card";

export default function DashboardRepositories() {
	const { data: installations = [], isLoading: installationsLoading } = useInstallations();

	const handleConnectRepo = () => githubApi.install();

	return (
		<div className="flex flex-col gap-10 max-w-4xl">
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
		</div>
	);
}
