import { create } from "zustand";
import { authApi, type User } from "@/lib/api";

type AuthStatus = "loading" | "authenticated" | "unauthenticated";

interface AuthState {
	status: AuthStatus;
	user: User | null;
	setStatus: (status: AuthStatus) => void;
	fetchUser: () => Promise<void>;
	logout: () => Promise<void>;
}

/**
 * Zustand owns *client* auth state: is the user logged in / loading?
 * It also stores the User object directly, replacing React Query for global auth state.
 */
export const useAuthStore = create<AuthState>((set) => ({
	status: "loading",
	user: null,

	setStatus: (status) => set({ status }),

	fetchUser: async () => {
		set({ status: "loading" });
		try {
			const res = await authApi.me();
			if (res.data) {
				set({ status: "authenticated", user: res.data });
			} else {
				set({ status: "unauthenticated", user: null });
			}
		} catch (err: any) {
			set({ status: "unauthenticated", user: null });
		}
	},

	logout: async () => {
		try {
			await authApi.logout();
		} finally {
			set({ status: "unauthenticated", user: null });
		}
	},
}));
