import { Options } from "k6/options";
import { healthTest } from "@/scenarios/health";

export const options: Options = {
	stages: [
		{ duration: "5s", target: 50 },  // Ramp up to 50 VUs
		{ duration: "10s", target: 100 }, // Ramp up to 100 VUs
		{ duration: "15s", target: 100 }, // Stay at 100 VUs
		{ duration: "5s", target: 0 },   // Ramp down to 0 VUs
	],
	thresholds: {
		http_req_duration: ["p(95)<500"], // 95% of requests must complete below 500ms
	},
};

export default function () {
	healthTest();
}
