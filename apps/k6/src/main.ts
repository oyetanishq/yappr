import { Options } from "k6/options";
import { healthTest } from "@/scenarios/health";
import { agentHealthTest } from "@/scenarios/agentHealth";
import { authTest, authReadTest, users } from "@/scenarios/auth";
import { repoConfigTest } from "@/scenarios/repo";
import { githubTest } from "@/scenarios/github";
import { billingTest } from "@/scenarios/billing";

// ── Tunables (override at run time, e.g. `-e TARGET_RPS=600 -e PEAK_RPS=2000`) ──
// TARGET/PEAK are *iterations* per second. Each stress iteration fires ~12–15
// HTTP requests, so real req/s ≈ rate × ~13.
const TARGET_RPS = Number(__ENV.TARGET_RPS || 300);
const PEAK_RPS = Number(__ENV.PEAK_RPS || TARGET_RPS * 3);
const RAMP = __ENV.RAMP || "1m";
const HOLD = __ENV.HOLD || "3m";

export const options: Options = {
	scenarios: {
		// ── Open model: arrivals keep coming regardless of how the system copes,
		// so this exposes the saturation point instead of backing off like VUs do.
		stress: {
			executor: "ramping-arrival-rate",
			startRate: 20,
			timeUnit: "1s",
			preAllocatedVUs: Number(__ENV.PRE_VUS || 300),
			maxVUs: Number(__ENV.MAX_VUS || 2000),
			stages: [
				{ target: TARGET_RPS, duration: RAMP }, // warm up to plateau
				{ target: TARGET_RPS, duration: HOLD }, // hold — steady-state numbers
				{ target: PEAK_RPS, duration: RAMP }, // push toward the wall
				{ target: PEAK_RPS, duration: "1m" }, // sit on the wall
				{ target: 0, duration: "30s" }, // ramp down — observe recovery
			],
			exec: "stressScenario",
		},

		// ── Destructive auth flow (logout + session revoke) runs once per seeded
		// user. Kept OUT of the open-model loop: re-hitting a consumed session
		// would report false 401 failures that have nothing to do with load.
		destructiveAuth: {
			executor: "shared-iterations",
			vus: Math.min(300, users.length || 1),
			iterations: users.length || 1,
			maxDuration: "10m",
			exec: "destructiveScenario",
		},
	},
	thresholds: {
		// Abort the run once the error rate blows past 2% — that's the wall.
		// delayAbortEval gives the warm-up ramp a grace period first.
		http_req_failed: [{ threshold: "rate<0.02", abortOnFail: true, delayAbortEval: "1m" }],
		http_req_duration: ["p(95)<500", "p(99)<2000"],
		checks: ["rate>0.98"],
	},
};

// Idempotent read / verify-only workload — safe to hammer at any RPS.
export function stressScenario() {
	healthTest();
	agentHealthTest();
	authReadTest(); // GET /me + GET /sessions (cookies[0] is never consumed)
	repoConfigTest(); // GET config + idempotent PUT upsert
	githubTest(); // installations read + cheap negative paths
	billingTest(); // webhook HMAC 401 + free-tier cancel 400 (no writes)
}

// Destructive — one pass per user, bounded scenario only.
export function destructiveScenario() {
	authTest();
}
