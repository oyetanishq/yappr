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
	is_current: boolean;
	user_agent: string;
	ip: string;
}

export interface Installation {
	id: string;
	installation_id: number;
	user_id: string;
	account_login: string;
	app_id: string;
	created_at: string;
	updated_at: string;
}

export interface InstallationRepo {
	id: number;
	name: string;
	full_name: string;
	private: boolean;
	html_url: string;
	description: string;
	updated_at: string;
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

// ── GitHub ────────────────────────────────────────────────────────────────────

export const githubApi = {
	/** Full-page navigation — redirects to GitHub's repo picker via the API */
	install: () => {
		window.location.href = `${BASE_URL}/api/v1/github/install`;
	},

	installations: () => request<ApiResponse<Installation[]>>("/api/v1/github/installations"),

	installationRepos: (id: number) => request<ApiResponse<InstallationRepo[]>>(`/api/v1/github/installations/${id}/repos`),
};

// ── Repo Config ───────────────────────────────────────────────────────────────

export type Personality = "bestie" | "senior_dev" | "sigma" | "toxic_tech_lead";

export const PERSONALITY_LABELS: Record<Personality, string> = {
	bestie: "The Bestie",
	senior_dev: "The Senior Dev",
	sigma: "The Sigma",
	toxic_tech_lead: "Toxic Tech Lead",
};

export const PERSONALITY_DESCRIPTIONS: Record<Personality, string> = {
	bestie: "Casual, warm, emoji-friendly 🧸. Celebrates wins, softens critiques.",
	senior_dev: "Professional, precise, mentorship-focused 🧑‍💻. The balanced default.",
	sigma: "Ultra-terse. Facts only. No fluff. No greetings. Pure signal 🪨.",
	toxic_tech_lead: "Brutal, sarcastic, zero-tolerance — but technically ruthless ☠️.",
};

export const PERSONALITIES: Personality[] = ["bestie", "senior_dev", "sigma", "toxic_tech_lead"];

export interface RepoConfig {
	id?: string;
	repo_full_name: string;
	user_id?: string;
	ignored_paths: string[];
	personality: Personality;
	created_at?: string;
	updated_at?: string;
}

export const repoApi = {
	getConfig: (owner: string, repo: string) => request<ApiResponse<RepoConfig>>(`/api/v1/repos/${owner}/${repo}/config`),

	updateConfig: (owner: string, repo: string, payload: { ignored_paths: string[]; personality: Personality }) =>
		request<ApiResponse<RepoConfig>>(`/api/v1/repos/${owner}/${repo}/config`, {
			method: "PUT",
			body: payload,
		}),
};
