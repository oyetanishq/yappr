import http from "k6/http";
import { ENV } from "@/config/env";

export const authApi = {
	me: (cookie?: string) => {
		const params = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		return http.get(`${ENV.BASE_URL}/api/v1/auth/me`, params);
	},
	sessions: (cookie?: string) => {
		const params = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		return http.get(`${ENV.BASE_URL}/api/v1/auth/sessions`, params);
	},
};
