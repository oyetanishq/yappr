import { check, group } from "k6";
import { billingApi } from "@/api/billing";
import { users } from "@/scenarios/auth";
import exec from "k6/execution";

interface SeededUser {
	user_id: string;
	github_id: number;
	cookies: string[];
	repos: string[];
}

const INVALID_COOKIE = "invalid_session_token_12345";

// A minimal, well-formed Razorpay webhook envelope. It is never accepted because
// we cannot produce a valid HMAC signature without the secret — so no billing
// state is ever mutated by these tests.
const WEBHOOK_BODY = JSON.stringify({
	event: "subscription.activated",
	payload: { subscription: { entity: { id: "sub_stress", notes: { user_id: "nobody" } } } },
});

// > 1 MB (maxWebhookBody). Rejected with 413 before HMAC verification.
const OVERSIZED_BODY = "x".repeat((1 << 20) + 1024);

export function billingTest() {
	if (users.length === 0) {
		return;
	}

	// Each iteration gets a unique user (all seeded users are Free tier).
	const userIndex = exec.scenario.iterationInTest % users.length;
	const user = users[userIndex] as SeededUser;
	const cookie = user.cookies[0];

	// ── POST /billing/cancel ─────────────────────────────────────────────────
	// Free-tier guard returns 400 before any Razorpay call, so this is safe to
	// hammer. Auth middleware short-circuits the missing/invalid cookie cases.
	group("POST /billing/cancel", () => {
		const resFree = billingApi.cancel(cookie);
		check(resFree, { "[POST /billing/cancel] free-tier user → 400": (r) => r.status === 400 });

		const resMissing = billingApi.cancel();
		check(resMissing, { "[POST /billing/cancel] missing cookie → 401": (r) => r.status === 401 });

		const resInvalid = billingApi.cancel(INVALID_COOKIE);
		check(resInvalid, { "[POST /billing/cancel] invalid cookie → 401": (r) => r.status === 401 });
	});

	// ── POST /billing/webhook ────────────────────────────────────────────────
	// Unauthenticated but HMAC-guarded. Stresses the signature-verification hot
	// path with zero side effects (no valid signature is ever sent).
	group("POST /billing/webhook", () => {
		const resNoSig = billingApi.webhook(WEBHOOK_BODY);
		check(resNoSig, { "[POST /billing/webhook] missing signature → 401": (r) => r.status === 401 });

		const resBadSig = billingApi.webhook(WEBHOOK_BODY, "deadbeefdeadbeefdeadbeefdeadbeef");
		check(resBadSig, { "[POST /billing/webhook] invalid signature → 401": (r) => r.status === 401 });

		// The 1 MB oversized body is bandwidth-heavy — firing it every iteration
		// dwarfs all other traffic and skews the results. Exercise it on ~1-in-50
		// iterations: still covered, without dominating data_sent.
		if (exec.scenario.iterationInTest % 50 === 0) {
			const resTooLarge = billingApi.webhook(OVERSIZED_BODY, "deadbeef");
			check(resTooLarge, { "[POST /billing/webhook] oversized body → 413": (r) => r.status === 413 });
		}
	});
}
