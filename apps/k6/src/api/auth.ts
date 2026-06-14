import http from "k6/http";
import { ENV } from "@/config/env";

export const authApi = {
	me: (cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "GET /api/v1/auth/me" };
		return http.get(`${ENV.BASE_URL}/api/v1/auth/me`, params);
	},
	sessions: (cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "GET /api/v1/auth/sessions" };
		return http.get(`${ENV.BASE_URL}/api/v1/auth/sessions`, params);
	},
	revokeSession: (sessionId: string, cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "DELETE /api/v1/auth/sessions/:id" };
		return http.del(`${ENV.BASE_URL}/api/v1/auth/sessions/${sessionId}`, null, params);
	},
	logout: (cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "POST /api/v1/auth/logout" };
		return http.post(`${ENV.BASE_URL}/api/v1/auth/logout`, null, params);
	},
};
