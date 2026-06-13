import http from "k6/http";
import { ENV } from "@/config/env";

export const authApi = {
	me: (cookie: string) => {
		return http.get(`${ENV.BASE_URL}/api/v1/auth/me`, {
			headers: {
				Cookie: `__session=${cookie}`,
			},
		});
	},
	sessions: (cookie: string) => {
		return http.get(`${ENV.BASE_URL}/api/v1/auth/sessions`, {
			headers: {
				Cookie: `__session=${cookie}`,
			},
		});
	},
};
