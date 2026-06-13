import { check } from "k6";
import { authApi } from "@/api/auth";
import { SharedArray } from "k6/data";

// In k6, open() evaluates relative to the script file.
// Since esbuild outputs to dist/main.js, users.json is at ../users.json
const users = new SharedArray("seeded users", function () {
	try {
		return JSON.parse(open("../test-data/users.json"));
	} catch (e) {
		console.warn("users.json not found, make sure to run the seed script first");
		return [];
	}
});

export function authTest() {
	if (users.length === 0) {
		return;
	}

	// Pick a random user from the seeded users
	const randomUser = users[Math.floor(Math.random() * users.length)];
	const validCookie = (randomUser as any).cookie;
	const invalidCookie = "invalid_session_token_12345";

	// --- SUCCESS CASES (200 OK) ---
	// Test /api/v1/auth/me
	const meResValid = authApi.me(validCookie);
	check(meResValid, { "/api/v1/auth/me (valid auth) - status is 200": (r) => r.status === 200 });

	// Test /api/v1/auth/sessions
	const sessionResValid = authApi.sessions(validCookie);
	check(sessionResValid, { "/api/v1/auth/sessions (valid auth) - status is 200": (r) => r.status === 200 });

	// --- FAILURE CASES (401 Unauthorized) ---
	// 1. Missing Cookie
	const meResMissing = authApi.me();
	check(meResMissing, { "/api/v1/auth/me (missing auth) - status is 401": (r) => r.status === 401 });

	const sessionResMissing = authApi.sessions();
	check(sessionResMissing, { "/api/v1/auth/sessions (missing auth) - status is 401": (r) => r.status === 401 });

	// 2. Invalid Cookie
	const meResInvalid = authApi.me(invalidCookie);
	check(meResInvalid, { "/api/v1/auth/me (invalid auth) - status is 401": (r) => r.status === 401 });

	const sessionResInvalid = authApi.sessions(invalidCookie);
	check(sessionResInvalid, { "/api/v1/auth/sessions (invalid auth) - status is 401": (r) => r.status === 401 });
}
