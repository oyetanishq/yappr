import { useEffect } from "react";
import { useAuthStore } from "@/store/auth";

/**
 * AuthProvider calls fetchMe once on app startup so the Zustand store
 * is hydrated before any page renders. No context needed — the store is global.
 */
export default function AuthProvider({ children }: { children: React.ReactNode }) {
	const fetchMe = useAuthStore((s) => s.fetchMe);

	useEffect(() => {
		fetchMe();
	}, [fetchMe]);

	return <>{children}</>;
}
