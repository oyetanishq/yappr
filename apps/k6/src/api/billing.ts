import http from "k6/http";
import { ENV } from "@/config/env";

export const billingApi = {
	cancel: (cookie?: string) => {
		const params: any = cookie ? { headers: { Cookie: `__session=${cookie}` } } : {};
		params.tags = { name: "POST /api/v1/billing/cancel" };
		return http.post(`${ENV.BASE_URL}/api/v1/billing/cancel`, null, params);
	},
	// webhook is unauthenticated — secured by HMAC-SHA256. We never send a valid
	// signature, so no state is ever mutated: this only exercises the verify path.
	webhook: (body: string, signature?: string) => {
		const params: any = { headers: { "Content-Type": "application/json" } };
		if (signature !== undefined) {
			params.headers["X-Razorpay-Signature"] = signature;
		}
		params.tags = { name: "POST /api/v1/billing/webhook" };
		return http.post(`${ENV.BASE_URL}/api/v1/billing/webhook`, body, params);
	},
};
