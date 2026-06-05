import { useEffect } from "react";
import { useMe } from "@/lib/hooks";
import { useAuthStore } from "@/store/auth";

/**
 * AuthProvider runs useMe on app startup.
 * React Query owns the User data; Zustand owns the auth status flag.
 * This component keeps them in sync.
 */
export default function AuthProvider({ children }: { children: React.ReactNode }) {
	const { data, status } = useMe();
	const setStatus = useAuthStore((s) => s.setStatus);

	useEffect(() => {
		if (status === "pending") {
			setStatus("loading");
		} else if (status === "success" && data) {
			setStatus("authenticated");
		} else if (status === "error") {
			setStatus("unauthenticated");
		}
	}, [status, data, setStatus]);

	return <>{children}</>;
}
