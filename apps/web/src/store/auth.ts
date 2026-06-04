import { create } from "zustand";
import { ApiError, authApi, type User } from "@/lib/api";

type AuthStatus = "loading" | "authenticated" | "unauthenticated";

interface AuthState {
	user: User | null;
	status: AuthStatus;
	fetchMe: () => Promise<void>;
	logout: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
	user: null,
	status: "loading",

	fetchMe: async () => {
		try {
			const res = await authApi.me();
			set({ user: res.data, status: "authenticated" });
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) {
				set({ user: null, status: "unauthenticated" });
			} else {
				// Network errors, etc. — treat as unauthenticated to avoid infinite spinner
				set({ user: null, status: "unauthenticated" });
			}
		}
	},

	logout: async () => {
		try {
			await authApi.logout();
		} finally {
			set({ user: null, status: "unauthenticated" });
		}
	},
}));
