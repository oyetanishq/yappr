import { Options } from "k6/options";
import { healthTest } from "@/scenarios/health";
import { authTest, users } from "@/scenarios/auth";

export const options: Options = {
	scenarios: {
		auth: {
			executor: "shared-iterations",
			vus: Math.min(200, users.length || 1),
			iterations: users.length || 1,
			maxDuration: "5m",
		},
	},
	thresholds: {
		http_req_duration: ["p(95)<500"], // 95% of requests must complete below 500ms
	},
};

export default function () {
	healthTest();
	authTest();
}
