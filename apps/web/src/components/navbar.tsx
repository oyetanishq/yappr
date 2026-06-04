// import { useState, useEffect } from "react";
// import { Bell, Settings } from "lucide-react";

// const navLinks = [
// 	{ label: "Dashboard", href: "#" },
// 	{ label: "Projects", href: "#" },
// 	{ label: "Docs", href: "#" },
// ];

// export default function Navbar() {
// 	const [scrolled, setScrolled] = useState(false);

// 	useEffect(() => {
// 		const handleScroll = () => setScrolled(window.scrollY > 20);
// 		window.addEventListener("scroll", handleScroll, { passive: true });
// 		return () => window.removeEventListener("scroll", handleScroll);
// 	}, []);

// 	const baseClasses = "flex justify-between items-center px-6 h-16 fixed z-50 bg-background border-border-stark transition-all duration-300 left-1/2 -translate-x-1/2";

// 	const scrolledClasses = "top-0 w-full border-b-[3px]";
// 	const unscrolledClasses = "top-4 w-[calc(100%-2rem)] max-w-7xl rounded-xl border-[3px] shadow-[4px_4px_0px_0px_rgba(0,0,0,1)]";

// 	return (
// 		<nav id="navbar" className={`${baseClasses} ${scrolled ? scrolledClasses : unscrolledClasses}`}>
// 			{/* Left: Logo + Nav Links */}
// 			<div className="flex items-center gap-6">
// 				<span className="font-bold tracking-tighter uppercase text-on-surface text-2xl md:text-lg" style={{ fontFamily: "var(--font-space-grotesk)" }}>
// 					YAPPR
// 				</span>
// 				<div className="hidden md:flex gap-4">
// 					{navLinks.map((link) => (
// 						<a
// 							key={link.label}
// 							href={link.href}
// 							className="text-sm text-on-surface-variant hover:bg-primary hover:text-on-primary transition-colors px-2 py-1"
// 							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
// 						>
// 							{link.label}
// 						</a>
// 					))}
// 				</div>
// 			</div>

// 			{/* Right: Actions + Avatar */}
// 			<div className="flex items-center gap-4">
// 				<button aria-label="Notifications" className="p-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container">
// 					<Bell size={20} />
// 				</button>
// 				<button aria-label="Settings" className="p-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container">
// 					<Settings size={20} />
// 				</button>
// 				<div className="w-10 h-10 border-[3px] border-border-stark hard-shadow bg-tertiary-container overflow-hidden rounded-full">
// 					<img
// 						alt="User Avatar"
// 						className="w-full h-full object-cover"
// 						src="https://lh3.googleusercontent.com/aida-public/AB6AXuCL1TwiA-LFcRACvMplU2GP-Q0E4w6dfPc6qqXAUL5XJ9QwS8zTjfHHpvMK4DC4UpZSTC2Kx-rMkRhJ8PxBIhQZ62Meiby2oecMTYO-awtIGzGZV14DcMNjn0RmTAghb2yvqipVa90EpHQUFQcUdIOzXt-AFWNbcG-F2yLVOMyTb70hbkLObOCVV2AuuXg-8tpXz2YhH2FCdg5GtKW5FG2m85mOvDTFU0XktQHOlQn1pg2DvhzMdkN0gJME7VaYe32-NG6wKApAjx55"
// 					/>
// 				</div>
// 			</div>
// 		</nav>
// 	);
// }

import { useState, useEffect } from "react";
import { Bell, Settings } from "lucide-react";

const navLinks = [
	{ label: "Dashboard", href: "#" },
	{ label: "Projects", href: "#" },
	{ label: "Docs", href: "#" },
];

export default function Navbar() {
	const [scrolled, setScrolled] = useState(false);

	useEffect(() => {
		const handleScroll = () => setScrolled(window.scrollY > 20);
		window.addEventListener("scroll", handleScroll, { passive: true });
		return () => window.removeEventListener("scroll", handleScroll);
	}, []);

	// Added ease-in-out for a smoother feel
	const baseClasses = "flex justify-between items-center px-6 h-16 fixed z-50 bg-background border-border-stark transition-all duration-300 ease-in-out left-1/2 -translate-x-1/2";

	// Added max-w-full, rounded-none, explicit border-0s, and shadow-none
	const scrolledClasses = "top-0 w-full max-w-full rounded-none border-b-[3px] border-t-0 border-x-0 shadow-none";

	// Unchanged
	const unscrolledClasses = "top-4 w-[calc(100%-2rem)] max-w-7xl rounded-xl border-[3px] shadow-[4px_4px_0px_0px_rgba(0,0,0,1)]";

	return (
		<nav id="navbar" className={`${baseClasses} ${scrolled ? scrolledClasses : unscrolledClasses}`}>
			{/* Left: Logo + Nav Links */}
			<div className="flex items-center gap-6">
				<span className="font-bold tracking-tighter uppercase text-on-surface text-2xl md:text-lg" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					YAPPR
				</span>
				<div className="hidden md:flex gap-4">
					{navLinks.map((link) => (
						<a
							key={link.label}
							href={link.href}
							className="text-sm text-on-surface-variant hover:bg-primary hover:text-on-primary transition-colors px-2 py-1"
							style={{ fontFamily: "var(--font-jetbrains-mono)" }}
						>
							{link.label}
						</a>
					))}
				</div>
			</div>

			{/* Right: Actions + Avatar */}
			<div className="flex items-center gap-4">
				<button aria-label="Notifications" className="p-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container">
					<Bell size={20} />
				</button>
				<button aria-label="Settings" className="p-2 border-[3px] border-border-stark hard-shadow bg-surface-container-highest hover:bg-primary-container">
					<Settings size={20} />
				</button>
				<div className="w-10 h-10 border-[3px] border-border-stark hard-shadow bg-tertiary-container overflow-hidden rounded-full">
					<img
						alt="User Avatar"
						className="w-full h-full object-cover"
						src="https://lh3.googleusercontent.com/aida-public/AB6AXuCL1TwiA-LFcRACvMplU2GP-Q0E4w6dfPc6qqXAUL5XJ9QwS8zTjfHHpvMK4DC4UpZSTC2Kx-rMkRhJ8PxBIhQZ62Meiby2oecMTYO-awtIGzGZV14DcMNjn0RmTAghb2yvqipVa90EpHQUFQcUdIOzXt-AFWNbcG-F2yLVOMyTb70hbkLObOCVV2AuuXg-8tpXz2YhH2FCdg5GtKW5FG2m85mOvDTFU0XktQHOlQn1pg2DvhzMdkN0gJME7VaYe32-NG6wKApAjx55"
					/>
				</div>
			</div>
		</nav>
	);
}
