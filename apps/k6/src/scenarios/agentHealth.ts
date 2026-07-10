import { check } from "k6";
import { agentHealthApi } from "@/api/agentHealth";

export function agentHealthTest() {
	const res = agentHealthApi.check();

	check(res, { "/health (agent) - status is 200": (r) => r.status === 200 });
}
