import Navbar from "@/components/navbar";
import HeroSection from "@/pages/home/hero-section";
import AiCodeReviewSection from "@/pages/home/ai-code-review-section";
import ArchitectureMappingSection from "@/pages/home/architecture-mapping-section";
import BugReportsSection from "@/pages/home/bug-report-section";
import PricingSection from "@/pages/home/pricing-section";
import Footer from "@/components/footer";
import { Noise } from "@/components/noise";

export default function Home() {
	return (
		<div
			className="min-h-screen flex flex-col relative grid-bg"
			style={{
				backgroundColor: "var(--color-background)",
				color: "var(--color-on-surface)",
			}}
		>
			<Navbar />
			<Noise />

			<main className="grow pt-28 px-6 pb-10 md:pt-36">
				{/* Hero */}
				<HeroSection />

				{/* Core Competencies Header */}
				<div className="max-w-6xl mx-auto mb-16">
					<h2
						className="text-4xl font-bold uppercase inline-block bg-primary-container px-4 py-2 border-[3px] border-border-stark hard-shadow -rotate-1"
						style={{ fontFamily: "var(--font-space-grotesk)" }}
					>
						Core Competencies
					</h2>
				</div>

				{/* Feature Sections */}
				<AiCodeReviewSection />
				<ArchitectureMappingSection />
				<BugReportsSection />

				{/* Pricing */}
				<PricingSection />
			</main>

			<Footer />
		</div>
	);
}
