const BASE_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

interface RequestOptions {
	method?: HttpMethod;
	body?: unknown;
}

export class ApiError extends Error {
	public status: number;

	constructor(status: number, message: string) {
		super(message);
		this.status = status;
		this.name = "ApiError";
	}
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const { method = "GET", body } = options;

	const res = await fetch(`${BASE_URL}${path}`, {
		method,
		credentials: "include", // send the __session HttpOnly cookie
		headers: {
			"Content-Type": "application/json",
		},
		...(body !== undefined ? { body: JSON.stringify(body) } : {}),
	});

	if (!res.ok) {
		throw new ApiError(res.status, `API error ${res.status}: ${path}`);
	}

	return res.json() as Promise<T>;
}

// ── Auth ─────────────────────────────────────────────────────────────────────

export interface User {
	id: string;
	github_id: number;
	login: string;
	name: string;
	email: string;
	avatar_url: string;
	created_at: string;
	updated_at: string;
}

interface ApiResponse<T> {
	data: T;
}

export interface Session {
	id: string;
	created_at: string;
	expires_at: string;
}

export const authApi = {
	/** Full-page navigation – not a fetch call */
	loginWithGithub: () => {
		window.location.href = `${BASE_URL}/api/v1/auth/github`;
	},

	me: () => request<ApiResponse<User>>("/api/v1/auth/me"),

	logout: () => request<ApiResponse<{ message: string }>>("/api/v1/auth/logout", { method: "POST" }),

	sessions: () => request<ApiResponse<Session[]>>("/api/v1/auth/sessions"),

	revokeSession: (id: string) => request<ApiResponse<{ message: string }>>(`/api/v1/auth/sessions/${id}`, { method: "DELETE" }),
};

// ── Query keys ────────────────────────────────────────────────────────────────
// Centralised query key factory — import these everywhere instead of raw strings.
export const queryKeys = {
	me: ["auth", "me"] as const,
	sessions: ["auth", "sessions"] as const,
};
