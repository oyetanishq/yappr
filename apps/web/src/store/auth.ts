import { create } from "zustand";
import { authApi } from "@/lib/api";

type AuthStatus = "loading" | "authenticated" | "unauthenticated";

interface AuthState {
	status: AuthStatus;
	setStatus: (status: AuthStatus) => void;
	logout: () => Promise<void>;
}

/**
 * Zustand owns *client* auth state: is the user logged in / loading?
 * The actual User object is fetched + cached by React Query (useMe hook).
 * Zustand and React Query are kept in sync via AuthProvider.
 */
export const useAuthStore = create<AuthState>((set) => ({
	status: "loading",

	setStatus: (status) => set({ status }),

	logout: async () => {
		try {
			await authApi.logout();
		} finally {
			set({ status: "unauthenticated" });
		}
	},
}));
