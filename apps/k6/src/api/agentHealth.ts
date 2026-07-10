import http from "k6/http";
import { ENV } from "@/config/env";

export const agentHealthApi = {
	check: () => {
		const params: any = { tags: { name: "GET /health (agent)" } };
		return http.get(`${ENV.AGENT_BASE_URL}/health`, params);
	},
};
