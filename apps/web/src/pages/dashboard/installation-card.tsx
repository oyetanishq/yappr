import { FolderGit2, ChevronDown, Lock, Globe, Settings } from "lucide-react";
import { useInstallationRepos } from "@/lib/hooks";
import { type Installation } from "@/lib/api";
import { useState } from "react";
import { useNavigate } from "react-router";

export function InstallationCard({ inst }: { inst: Installation }) {
	const [expanded, setExpanded] = useState(false);
	const navigate = useNavigate();
	const { data: repos = [], isLoading } = useInstallationRepos(expanded ? inst.installation_id : 0);

	return (
		<div className="border-[3px] border-border-stark hard-shadow bg-surface-container flex flex-col transition-all overflow-hidden">
			{/* Header (always visible) */}
			<div className="p-4 flex items-center justify-between gap-4 cursor-pointer hover:bg-surface-container-highest" onClick={() => setExpanded(!expanded)}>
				<div className="flex items-center gap-3">
					<div className="w-9 h-9 flex items-center justify-center border-[3px] border-border-stark bg-primary-container shrink-0">
						<FolderGit2 size={16} />
					</div>
					<div>
						<p className="text-sm font-bold" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							{inst.account_login || `Installation #${inst.installation_id}`}
						</p>
						<p className="text-xs text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							ID: {inst.installation_id}
						</p>
					</div>
				</div>
				<div className="flex items-center gap-3">
					<a
						href={`https://github.com/settings/installations/${inst.installation_id}`}
						target="_blank"
						rel="noopener noreferrer"
						onClick={(e) => e.stopPropagation()}
						className="flex items-center gap-1.5 px-2.5 py-1.5 border-[3px] border-border-stark hard-shadow bg-surface hover:bg-primary-container text-xs font-semibold uppercase cursor-pointer"
						style={{ fontFamily: "var(--font-jetbrains-mono)" }}
					>
						Manage
					</a>
					<div className={`p-1.5 border-[3px] border-transparent transition-transform duration-200 ${expanded ? "rotate-180" : ""}`}>
						<ChevronDown size={18} />
					</div>
				</div>
			</div>

			{/* Body (expanded) */}
			{expanded && (
				<div className="border-t-[3px] border-border-stark bg-surface-container-low p-4">
					{isLoading ? (
						<div className="flex flex-col gap-2">
							{[1, 2, 3].map((i) => (
								<div key={i} className="h-10 bg-surface-container border-[3px] border-border-stark animate-pulse" />
							))}
						</div>
					) : repos.length === 0 ? (
						<p className="text-xs text-on-surface-variant p-4 text-center border-[3px] border-border-stark border-dashed" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
							No repositories selected. Click 'Manage' to grant access to repositories.
						</p>
					) : (
						<div className="flex flex-col gap-2">
							{repos.map((repo) => {
								const [owner, repoName] = repo.full_name.split("/");
								return (
									<div key={repo.id} className="flex items-center justify-between p-3 border-[3px] border-border-stark bg-surface">
										<div className="flex items-center gap-3">
											{repo.private ? <Lock size={14} className="text-error" /> : <Globe size={14} className="text-primary" />}
											<a
												href={repo.html_url}
												target="_blank"
												rel="noopener noreferrer"
												className="text-sm font-bold hover:underline"
												style={{ fontFamily: "var(--font-jetbrains-mono)" }}
											>
												{repo.full_name}
											</a>
										</div>
										<div className="flex items-center gap-2">
											<div
												className="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 border-2 border-border-stark"
												style={{ fontFamily: "var(--font-jetbrains-mono)" }}
											>
												{repo.private ? "Private" : "Public"}
											</div>
											<button
												id={`configure-${repo.full_name.replace("/", "-")}`}
												onClick={(e) => {
													e.stopPropagation();
													navigate(`/dashboard/repos/${owner}/${repoName}/config`);
												}}
												className="flex items-center gap-1 px-2 py-1 border-[3px] border-border-stark bg-surface hover:bg-primary-container text-xs font-semibold uppercase cursor-pointer"
												style={{ fontFamily: "var(--font-jetbrains-mono)" }}
												title="Configure repo"
											>
												<Settings size={12} />
												<span>Config</span>
											</button>
										</div>
									</div>
								);
							})}
						</div>
					)}
				</div>
			)}
		</div>
	);
}
