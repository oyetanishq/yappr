import { useEffect } from "react";
import { useAuthStore } from "@/store/auth";

/**
 * AuthProvider fetches the user on app startup.
 */
export default function AuthProvider({ children }: { children: React.ReactNode }) {
	const fetchUser = useAuthStore((s) => s.fetchUser);

	useEffect(() => {
		fetchUser();
	}, [fetchUser]);

	return <>{children}</>;
}
