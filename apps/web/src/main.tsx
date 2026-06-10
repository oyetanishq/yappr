import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./global.css";

import { BrowserRouter, Route, Routes, Navigate } from "react-router";

import AuthProvider from "@/components/auth-provider";
import ProtectedRoute from "@/components/protected-route";

import Home from "@/pages/home";
import LoginPage from "@/pages/login";
import NotFound from "@/pages/not-found";

import AppLayout from "@/layouts/app-layout";
import DashboardLayout from "@/pages/dashboard/layout";
import DashboardOverview from "@/pages/dashboard/overview";
import DashboardRepositories from "@/pages/dashboard/repositories";
import RepoConfig from "@/pages/dashboard/repo-config";
import SettingsLayout from "@/pages/settings/layout";
import SettingsAccount from "@/pages/settings/account";
import SettingsBilling from "@/pages/settings/billing";
import SettingsSessions from "@/pages/settings/sessions";

createRoot(document.getElementById("root")!).render(
	<StrictMode>
		<BrowserRouter>
			<AuthProvider>
				<Routes>
					{/* Public routes */}
					<Route path="/" element={<Home />} />
					<Route path="/login" element={<LoginPage />} />

					{/* Protected routes */}
					<Route
						element={
							<ProtectedRoute>
								<AppLayout />
							</ProtectedRoute>
						}
					>
						<Route path="/dashboard" element={<DashboardLayout />}>
							<Route index element={<Navigate to="overview" replace />} />
							<Route path="overview" element={<DashboardOverview />} />
							<Route path="repositories" element={<DashboardRepositories />} />
							<Route path="repos/:owner/:repo/config" element={<RepoConfig />} />
						</Route>

						<Route path="/settings" element={<SettingsLayout />}>
							<Route index element={<Navigate to="account" replace />} />
							<Route path="account" element={<SettingsAccount />} />
							<Route path="billing" element={<SettingsBilling />} />
							<Route path="sessions" element={<SettingsSessions />} />
						</Route>
					</Route>

					{/* 404 */}
					<Route path="*" element={<NotFound />} />
				</Routes>
			</AuthProvider>
		</BrowserRouter>
	</StrictMode>,
);
