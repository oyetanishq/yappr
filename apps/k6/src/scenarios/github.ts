import { check, group } from "k6";
import { githubApi } from "@/api/github";
import { users } from "@/scenarios/auth";
import exec from "k6/execution";

interface SeededUser {
	user_id: string;
	github_id: number;
	cookies: string[];
	repos: string[];
}

const INVALID_COOKIE = "invalid_session_token_12345";

export function githubTest() {
	if (users.length === 0) {
		return;
	}

	// Each iteration gets a unique user
	const userIndex = exec.scenario.iterationInTest % users.length;
	const user = users[userIndex] as SeededUser;
	const cookie = user.cookies[0];

	// ── GET /github/installations ────────────────────────────────────────────
	// Seeded users have no installations, so the happy path returns an empty
	// 200 — this still exercises the auth middleware + Mongo read under load.
	group("GET /github/installations", () => {
		const resValid = githubApi.installations(cookie);
		check(resValid, { "[GET /installations] valid cookie → 200": (r) => r.status === 200 });

		const resMissing = githubApi.installations();
		check(resMissing, { "[GET /installations] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = githubApi.installations(INVALID_COOKIE);
		check(resInvalid, { "[GET /installations] invalid cookie → 401": (r) => r.status === 401 });
	});

	// ── GET /github/installations/:id/repos ──────────────────────────────────
	// No installations are seeded, so we only exercise the cheap negative paths
	// (no GitHub API call is reached for any of these).
	group("GET /github/installations/:id/repos", () => {
		// Non-numeric id → 400 (rejected before any DB/GitHub work).
		const resBadId = githubApi.installationRepos("not-a-number", cookie);
		check(resBadId, { "[GET /installations/:id/repos] non-numeric id → 400": (r) => r.status === 400 });

		// Valid-looking but non-existent installation → 404 (ownership lookup fails).
		const resNotFound = githubApi.installationRepos("999999999", cookie);
		check(resNotFound, { "[GET /installations/:id/repos] unknown id → 404": (r) => r.status === 404 });

		const resMissing = githubApi.installationRepos("999999999");
		check(resMissing, { "[GET /installations/:id/repos] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = githubApi.installationRepos("999999999", INVALID_COOKIE);
		check(resInvalid, { "[GET /installations/:id/repos] invalid cookie → 401": (r) => r.status === 401 });
	});
}
