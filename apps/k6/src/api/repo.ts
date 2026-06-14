import http from "k6/http";
import { ENV } from "@/config/env";

export const repoApi = {
	getConfig: (owner: string, repo: string, cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "GET /api/v1/repos/:owner/:repo/config" };
		return http.get(`${ENV.BASE_URL}/api/v1/repos/${owner}/${repo}/config`, params);
	},
	updateConfig: (owner: string, repo: string, body: any, cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}`, "Content-Type": "application/json" } } : { headers: { "Content-Type": "application/json" } };
		params.tags = { name: "PUT /api/v1/repos/:owner/:repo/config" };
		return http.put(`${ENV.BASE_URL}/api/v1/repos/${owner}/${repo}/config`, JSON.stringify(body), params);
	},
};
