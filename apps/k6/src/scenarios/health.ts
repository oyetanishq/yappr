import { check } from "k6";
import { healthApi } from "@/api/health";

export function healthTest() {
	const res = healthApi.check();

	check(res, { "/health - status is 200": (r) => r.status === 200 });
}
