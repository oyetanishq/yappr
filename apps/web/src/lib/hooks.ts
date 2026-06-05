import { useState, useEffect, useCallback } from "react";
import { authApi, githubApi, type Session, type Installation, type InstallationRepo } from "@/lib/api";

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
