import { check } from "k6";
import { healthApi } from "@/api/health";

export function healthTest() {
	const res = healthApi.check();

	check(res, {
		"status is 200": (r) => r.status === 200,
		"body has status ok": (r) => {
			try {
				const body = r.json() as any;
				return body && body.status === "ok";
			} catch (e) {
				return false;
			}
		},
	});
}
