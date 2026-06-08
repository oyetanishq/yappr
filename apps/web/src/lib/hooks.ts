import { useState, useEffect, useCallback } from "react";
import { authApi, githubApi, repoApi, type Session, type Installation, type InstallationRepo, type RepoConfig, type Personality } from "@/lib/api";

// ── Sessions ──────────────────────────────────────────────────────────────────

export function useSessions() {
	const [data, setData] = useState<Session[]>([]);
	const [isLoading, setIsLoading] = useState(true);
	const [isFetching, setIsFetching] = useState(true);
	const [isError, setIsError] = useState(false);

	const refetch = useCallback(async () => {
		setIsFetching(true);
		setIsError(false);
		try {
			const res = await authApi.sessions();
			setData(res.data ?? []);
		} catch (err) {
			setIsError(true);
		} finally {
			setIsLoading(false);
			setIsFetching(false);
		}
	}, []);

	useEffect(() => {
		refetch();
	}, [refetch]);

	return { data, isLoading, isFetching, isError, refetch };
}

export function useRevokeSession(options?: { onSuccess?: () => void }) {
	const [isPending, setIsPending] = useState(false);
	const [variables, setVariables] = useState<string | null>(null);

	const mutate = async (id: string, mutateOptions?: { onSuccess?: () => void }) => {
		setIsPending(true);
		setVariables(id);
		try {
			await authApi.revokeSession(id);
			mutateOptions?.onSuccess?.();
			options?.onSuccess?.();
		} catch (err) {
			console.error("Failed to revoke session", err);
		} finally {
			setIsPending(false);
			setVariables(null);
		}
	};

	return { mutate, isPending, variables };
}

// ── GitHub ────────────────────────────────────────────────────────────────────

export function useInstallations() {
	const [data, setData] = useState<Installation[]>([]);
	const [isLoading, setIsLoading] = useState(true);
	const [isError, setIsError] = useState(false);

	const refetch = useCallback(async () => {
		setIsLoading(true);
		setIsError(false);
		try {
			const res = await githubApi.installations();
			setData(res.data ?? []);
		} catch (err) {
			setIsError(true);
		} finally {
			setIsLoading(false);
		}
	}, []);

	useEffect(() => {
		refetch();
	}, [refetch]);

	return { data, isLoading, isError, refetch };
}

export function useInstallationRepos(installationId: number) {
	const [data, setData] = useState<InstallationRepo[]>([]);
	const [isLoading, setIsLoading] = useState(false);
	const [isError, setIsError] = useState(false);

	const refetch = useCallback(async () => {
		if (!installationId) return;
		setIsLoading(true);
		setIsError(false);
		try {
			const res = await githubApi.installationRepos(installationId);
			setData(res.data ?? []);
		} catch (err) {
			setIsError(true);
		} finally {
			setIsLoading(false);
		}
	}, [installationId]);

	useEffect(() => {
		refetch();
	}, [refetch]);

	return { data, isLoading, isError, refetch };
}

/**
 * Fetches repos for ALL installations in parallel and returns a flat list.
 * Used by the overview stat card to show total repos connected (not installation count).
 */
export function useAllRepos(installations: Installation[]) {
	const [data, setData] = useState<InstallationRepo[]>([]);
	const [isLoading, setIsLoading] = useState(false);
	const [isError, setIsError] = useState(false);

	useEffect(() => {
		if (installations.length === 0) {
			setData([]);
			return;
		}

		let cancelled = false;
		setIsLoading(true);
		setIsError(false);

		Promise.all(installations.map((inst) => githubApi.installationRepos(inst.installation_id).then((res) => res.data ?? [])))
			.then((results) => {
				if (!cancelled) {
					setData(results.flat());
				}
			})
			.catch(() => {
				if (!cancelled) setIsError(true);
			})
			.finally(() => {
				if (!cancelled) setIsLoading(false);
			});

		return () => {
			cancelled = true;
		};
	}, [installations]);

	return { data, isLoading, isError };
}

// ── Repo Config ───────────────────────────────────────────────────────────────

export function useRepoConfig(owner: string, repo: string) {
	const [data, setData] = useState<RepoConfig | null>(null);
	const [isLoading, setIsLoading] = useState(true);
	const [isError, setIsError] = useState(false);

	const refetch = useCallback(async () => {
		if (!owner || !repo) return;
		setIsLoading(true);
		setIsError(false);
		try {
			const res = await repoApi.getConfig(owner, repo);
			setData(res.data);
		} catch (err) {
			setIsError(true);
		} finally {
			setIsLoading(false);
		}
	}, [owner, repo]);

	useEffect(() => {
		refetch();
	}, [refetch]);

	return { data, isLoading, isError, refetch };
}

export function useUpdateRepoConfig() {
	const [isPending, setIsPending] = useState(false);
	const [isError, setIsError] = useState(false);
	const [isSuccess, setIsSuccess] = useState(false);

	const mutate = async (owner: string, repo: string, payload: { ignored_paths: string[]; personality: Personality }, options?: { onSuccess?: (data: RepoConfig) => void; onError?: () => void }) => {
		setIsPending(true);
		setIsError(false);
		setIsSuccess(false);
		try {
			const res = await repoApi.updateConfig(owner, repo, payload);
			setIsSuccess(true);
			options?.onSuccess?.(res.data);
		} catch (err) {
			setIsError(true);
			options?.onError?.();
		} finally {
			setIsPending(false);
		}
	};

	return { mutate, isPending, isError, isSuccess };
}
