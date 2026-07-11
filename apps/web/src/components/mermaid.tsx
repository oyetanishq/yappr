import { useEffect, useRef, useState } from "react";
import mermaid from "mermaid";

let initialized = false;
let renderSeq = 0;

/** Reads a CSS custom property from :root, with a fallback. */
function cssVar(name: string, fallback: string): string {
	if (typeof window === "undefined") return fallback;
	const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
	return v || fallback;
}

/** Initializes mermaid once, themed to match the app's neo-brutalist palette. */
function ensureInit() {
	if (initialized) return;
	initialized = true;
	mermaid.initialize({
		startOnLoad: false,
		securityLevel: "strict",
		theme: "base",
		fontFamily: cssVar("--font-jetbrains-mono", "monospace"),
		themeVariables: {
			background: cssVar("--color-surface", "#fcf9f4"),
			primaryColor: cssVar("--color-primary-container", "#00ccff"),
			primaryTextColor: cssVar("--color-on-surface", "#1c1c19"),
			primaryBorderColor: cssVar("--color-border-stark", "#000000"),
			lineColor: cssVar("--color-border-stark", "#000000"),
			secondaryColor: cssVar("--color-surface-container-highest", "#e5e2dd"),
			tertiaryColor: cssVar("--color-surface-container-low", "#f6f3ee"),
			textColor: cssVar("--color-on-surface", "#1c1c19"),
			fontSize: "13px",
		},
	});
}

/** Renders a Mermaid diagram from source, falling back to the raw source on error. */
export default function Mermaid({ chart }: { chart: string }) {
	const ref = useRef<HTMLDivElement>(null);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		let cancelled = false;
		ensureInit();
		// Unique per render call so StrictMode's double-invoke can't collide on the temp DOM id.
		const id = "mermaid-" + ++renderSeq;

		mermaid
			.render(id, chart.trim())
			.then(({ svg, bindFunctions }) => {
				if (cancelled || !ref.current) return;
				ref.current.innerHTML = svg;
				bindFunctions?.(ref.current);
				setError(null);
			})
			.catch((err: unknown) => {
				if (cancelled) return;
				setError(err instanceof Error ? err.message : String(err));
			});

		return () => {
			cancelled = true;
		};
	}, [chart]);

	// The ref target stays mounted (just hidden on error) so a later successful
	// re-render can always write into it and clear the error state.
	return (
		<>
			{error && (
				<div className="border-[3px] border-border-stark bg-surface p-4">
					<p className="text-[11px] font-bold uppercase text-on-surface-variant mb-2" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						Could not render diagram
					</p>
					<pre className="text-xs overflow-x-auto whitespace-pre-wrap" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						{chart.trim()}
					</pre>
				</div>
			)}
			<div
				ref={ref}
				hidden={!!error}
				className="mermaid-diagram border-[3px] border-border-stark bg-surface p-4 overflow-x-auto flex justify-center"
			/>
		</>
	);
}
