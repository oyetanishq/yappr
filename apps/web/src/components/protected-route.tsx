import { Navigate } from "react-router";
import { useAuthStore } from "@/store/auth";

/**
 * ProtectedRoute guards any route that requires authentication.
 *
 * - loading        → full-page spinner (prevents flicker)
 * - authenticated  → renders children
 * - unauthenticated → redirects to /login
 */
export default function ProtectedRoute({ children }: { children: React.ReactNode }) {
	const status = useAuthStore((s) => s.status);

	if (status === "loading") {
		return (
			<div className="min-h-screen flex items-center justify-center grid-bg" style={{ backgroundColor: "var(--color-background)" }}>
				<div className="flex flex-col items-center gap-4">
					<div className="w-12 h-12 border-[3px] border-border-stark border-t-primary animate-spin" />
					<p className="text-sm text-on-surface-variant" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						authenticating...
					</p>
				</div>
			</div>
		);
	}

	if (status === "unauthenticated") {
		return <Navigate to="/login" replace />;
	}

	return <>{children}</>;
}
