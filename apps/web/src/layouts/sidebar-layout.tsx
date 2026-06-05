import { NavLink, Outlet } from "react-router";
import type { LucideIcon } from "lucide-react";

export interface SidebarLink {
	name: string;
	href: string;
	icon: LucideIcon;
}

export interface SidebarLayoutProps {
	title: string;
	description: string;
	links: SidebarLink[];
}

export default function SidebarLayout({ title, description, links }: SidebarLayoutProps) {
	return (
		<div className="flex flex-col md:flex-row w-full flex-1 px-4 sm:px-6 py-8 md:py-10 gap-8">
			{/* Sidebar Navigation */}
			<aside className="w-full md:w-64 shrink-0 flex flex-col gap-6">
				<div>
					<h1 className="text-3xl font-bold uppercase tracking-tight" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						{title}
					</h1>
					<p className="text-sm text-on-surface-variant mt-1" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						{description}
					</p>
				</div>

				<nav className="flex flex-col gap-2">
					{links.map((link) => (
						<NavLink
							key={link.name}
							to={link.href}
							className={({ isActive }) =>
								`flex items-center gap-3 px-4 py-3 border-[3px] border-border-stark hard-shadow cursor-pointer transition-colors ${
									isActive ? "bg-primary text-on-primary" : "bg-surface-container-low hover:bg-primary-container hover:text-on-primary-container"
								}`
							}
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							<link.icon size={16} />
							<span className="text-sm font-bold uppercase">{link.name}</span>
						</NavLink>
					))}
				</nav>
			</aside>

			{/* Main Content Area */}
			<main className="flex-1 min-w-0 flex flex-col gap-8">
				<Outlet />
			</main>
		</div>
	);
}
