import http from "k6/http";
import { ENV } from "@/config/env";

export const githubApi = {
	installations: (cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "GET /api/v1/github/installations" };
		return http.get(`${ENV.BASE_URL}/api/v1/github/installations`, params);
	},
	installationRepos: (installationId: string, cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "GET /api/v1/github/installations/:id/repos" };
		return http.get(`${ENV.BASE_URL}/api/v1/github/installations/${installationId}/repos`, params);
	},
};
