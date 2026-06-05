import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { authApi, githubApi, queryKeys, type User } from "@/lib/api";

// ── Auth ──────────────────────────────────────────────────────────────────────

/**
 * Fetches the currently authenticated user.
 * retry: false — a 401 is not a transient error, don't hammer the server.
 */
export function useMe() {
	return useQuery({
		queryKey: queryKeys.me,
		queryFn: async () => {
			const res = await authApi.me();
			return res.data as User;
		},
		retry: false,
		staleTime: 1000 * 60 * 5, // user data is stable — refetch every 5 min
	});
}

// ── Sessions ──────────────────────────────────────────────────────────────────

export function useSessions() {
	return useQuery({
		queryKey: queryKeys.sessions,
		queryFn: async () => {
			const res = await authApi.sessions();
			return res.data ?? [];
		},
	});
}

export function useRevokeSession() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (id: string) => authApi.revokeSession(id),
		onSuccess: (_data, id) => {
			// Optimistically remove the revoked session from cache
			queryClient.setQueryData(queryKeys.sessions, (old: typeof queryKeys.sessions | undefined) => {
				if (!Array.isArray(old)) return old;
				return old.filter((s: { id: string }) => s.id !== id);
			});
		},
	});
}

// ── GitHub ────────────────────────────────────────────────────────────────────

export function useInstallations() {
	return useQuery({
		queryKey: queryKeys.installations,
		queryFn: async () => {
			const res = await githubApi.installations();
			return res.data ?? [];
		},
	});
}

export function useInstallationRepos(installationId: number) {
	return useQuery({
		queryKey: queryKeys.installationRepos(installationId),
		queryFn: async () => {
			const res = await githubApi.installationRepos(installationId);
			return res.data ?? [];
		},
		enabled: !!installationId,
	});
}
