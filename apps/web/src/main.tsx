import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./global.css";

import { BrowserRouter, Route, Routes } from "react-router";

import Home from "@/pages/home";
import NotFound from "@/pages/not-found";

const App = () => {
	return (
		<StrictMode>
			<BrowserRouter>
				<Routes>
					{/* home route */}
					<Route path="/" element={<Home />} />

					{/* not found page */}
					<Route path="*" element={<NotFound />} />
				</Routes>
			</BrowserRouter>
		</StrictMode>
	);
};

createRoot(document.getElementById("root")!).render(<App />);
