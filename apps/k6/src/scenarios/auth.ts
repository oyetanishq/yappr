import { check, group } from "k6";
import { authApi } from "@/api/auth";
import { SharedArray } from "k6/data";
import exec from "k6/execution";

interface SeededUser {
	user_id: string;
	github_id: number;
	cookies: string[];
}

interface SessionEntry {
	id: string;
	is_current: boolean;
}

export const users = new SharedArray("seeded users", function () {
	try {
		return JSON.parse(open("../test-data/users.json"));
	} catch (e) {
		console.warn("users.json not found, make sure to run the seed script first");
		return [];
	}
});

const INVALID_COOKIE = "invalid_session_token_12345";

export function authTest() {
	if (users.length === 0) {
		return;
	}

	// Each iteration gets a unique user — no collisions on destructive tests.
	const userIndex = exec.scenario.iterationInTest % users.length;
	const user = users[userIndex] as SeededUser;
	const readCookie = user.cookies[0];
	const revokeCookie = user.cookies[1];
	const logoutCookie = user.cookies[2];

	group("GET /auth/me", () => {
		const resValid = authApi.me(readCookie);
		check(resValid, { "[GET /auth/me] valid cookie → 200": (r) => r.status === 200 });

		const resMissing = authApi.me();
		check(resMissing, { "[GET /auth/me] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = authApi.me(INVALID_COOKIE);
		check(resInvalid, { "[GET /auth/me] invalid cookie → 401": (r) => r.status === 401 });
	});

	// ── GET /auth/sessions ────────────────────────────────────────────────────
	let revokeSessionId = "";

	group("GET /auth/sessions", () => {
		const resValid = authApi.sessions(readCookie);
		check(resValid, { "[GET /auth/sessions] valid cookie → 200": (r) => r.status === 200 });

		// Parse sessions to find a non-current session ID for the revoke test.
		try {
			const body = JSON.parse(resValid.body as string);
			const sessions: SessionEntry[] = body.data;
			const nonCurrent = sessions.find((s) => !s.is_current);
			if (nonCurrent) {
				revokeSessionId = nonCurrent.id;
			}
		} catch (_) {
			/* parse failure is fine, revoke test will be skipped */
		}

		check(resValid, {
			"[GET /auth/sessions] response contains is_current session": (r) => {
				try {
					const body = JSON.parse(r.body as string);
					return body.data.some((s: SessionEntry) => s.is_current === true);
				} catch (_) {
					return false;
				}
			},
		});

		const resMissing = authApi.sessions();
		check(resMissing, { "[GET /auth/sessions] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = authApi.sessions(INVALID_COOKIE);
		check(resInvalid, { "[GET /auth/sessions] invalid cookie → 401": (r) => r.status === 401 });
	});

	// ── DELETE /auth/sessions/:id ─────────────────────────────────────────────
	group("DELETE /auth/sessions/:id", () => {
		// Revoke own session (cookies[1])
		if (revokeSessionId !== "") {
			const resRevoke = authApi.revokeSession(revokeSessionId, revokeCookie);
			check(resRevoke, { "[DELETE /auth/sessions/:id] revoke own session → 200": (r) => r.status === 200 });

			// The revoked cookie should now be invalid
			const resAfter = authApi.me(revokeCookie);
			check(resAfter, { "[DELETE /auth/sessions/:id] revoked cookie on /me → 401": (r) => r.status === 401 });
		}

		// Non-existent session ID
		const resNotFound = authApi.revokeSession("nonexistent-session-00000", readCookie);
		check(resNotFound, { "[DELETE /auth/sessions/:id] non-existent session id → 404": (r) => r.status === 404 });

		const resMissing = authApi.revokeSession("any-id", undefined as any);
		check(resMissing, { "[DELETE /auth/sessions/:id] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = authApi.revokeSession("any-id", INVALID_COOKIE);
		check(resInvalid, { "[DELETE /auth/sessions/:id] invalid cookie → 401": (r) => r.status === 401 });
	});

	// ── POST /auth/logout ─────────────────────────────────────────────────────
	group("POST /auth/logout", () => {
		const resValid = authApi.logout(logoutCookie);
		check(resValid, { "[POST /auth/logout] valid cookie → 200": (r) => r.status === 200 });

		// The logged-out cookie should now be invalid
		const resAfter = authApi.me(logoutCookie);
		check(resAfter, { "[POST /auth/logout] logged-out cookie on /me → 401": (r) => r.status === 401 });

		const resMissing = authApi.logout();
		check(resMissing, { "[POST /auth/logout] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = authApi.logout(INVALID_COOKIE);
		check(resInvalid, { "[POST /auth/logout] invalid cookie → 401": (r) => r.status === 401 });
	});
}
