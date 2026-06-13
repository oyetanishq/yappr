import http from "k6/http";
import { ENV } from "@/config/env";

export const healthApi = {
	check: () => http.get(`${ENV.BASE_URL}/health`),
};
