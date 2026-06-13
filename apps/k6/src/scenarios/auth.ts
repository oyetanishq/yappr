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
	const cookie = (randomUser as any).cookie;

	// Test /api/v1/auth/me
	const meRes = authApi.me(cookie);
	check(meRes, { "/api/v1/auth/me - status is 200": (r) => r.status === 200 });

	// Test /api/v1/auth/sessions
	const sessionRes = authApi.sessions(cookie);
	check(sessionRes, { "/api/v1/auth/sessions - status is 200": (r) => r.status === 200 });
}
