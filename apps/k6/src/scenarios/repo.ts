import { check, group } from "k6";
import { repoApi } from "@/api/repo";
import { users } from "@/scenarios/auth";
import exec from "k6/execution";

interface SeededUser {
	user_id: string;
	github_id: number;
	cookies: string[];
	repos: string[];
}

const INVALID_COOKIE = "invalid_session_token_12345";

export function repoConfigTest() {
	if (users.length === 0) {
		return;
	}

	// Each iteration gets a unique user
	const userIndex = exec.scenario.iterationInTest % users.length;
	const user = users[userIndex] as SeededUser;

	// We only need one cookie for these tests (no destructive operations)
	const cookie = user.cookies[0];

	// Pick a random repo from the user's seeded repos to increase cardinality
	const randomRepoFullName = user.repos[Math.floor(Math.random() * user.repos.length)];
	const [owner, repo] = randomRepoFullName.split("/");

	// ── GET /repos/:owner/:repo/config ───────────────────────────────────────────
	group("GET /repos/:owner/:repo/config", () => {
		const resValid = repoApi.getConfig(owner, repo, cookie);
		check(resValid, { "[GET /config] valid cookie → 200": (r) => r.status === 200 });

		const resMissing = repoApi.getConfig(owner, repo);
		check(resMissing, { "[GET /config] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = repoApi.getConfig(owner, repo, INVALID_COOKIE);
		check(resInvalid, { "[GET /config] invalid cookie → 401": (r) => r.status === 401 });
	});

	// ── PUT /repos/:owner/:repo/config ───────────────────────────────────────────
	group("PUT /repos/:owner/:repo/config", () => {
		// Valid payload with Free Tier allowed personality
		const validBody = {
			ignored_paths: ["dist/", "node_modules/"],
			personality: "senior_dev",
		};
		const resValid = repoApi.updateConfig(owner, repo, validBody, cookie);
		check(resValid, { "[PUT /config] valid body (free tier personality) → 200": (r) => r.status === 200 });

		// Valid payload with Pro Tier personality (Seeded users are Free tier)
		const proBody = {
			ignored_paths: ["dist/"],
			personality: "toxic_tech_lead",
		};
		const resPro = repoApi.updateConfig(owner, repo, proBody, cookie);
		check(resPro, { "[PUT /config] valid body (pro tier personality) → 402": (r) => r.status === 402 });

		// Invalid payload (e.g. malformed or invalid fields)
		const invalidBody = "this is not json";
		// To send malformed body using our helper, we pass the string directly, but JSON.stringify will wrap it in quotes.
		// So we use k6 http directly for this specific malformed test, or just test an empty object/missing required fields if applicable.
		// Actually, the handler uses c.ShouldBindJSON, which will fail if the body is not an object or array.
		const resInvalidBody = repoApi.updateConfig(owner, repo, invalidBody, cookie);
		check(resInvalidBody, { "[PUT /config] invalid body → 400": (r) => r.status === 400 });

		// Missing cookie
		const resMissing = repoApi.updateConfig(owner, repo, validBody);
		check(resMissing, { "[PUT /config] missing cookie → 401": (r) => r.status === 401 });

		// Invalid cookie
		const resInvalid = repoApi.updateConfig(owner, repo, validBody, INVALID_COOKIE);
		check(resInvalid, { "[PUT /config] invalid cookie → 401": (r) => r.status === 401 });
	});
}
