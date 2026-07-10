import { Check, CheckCircle } from "lucide-react";
import { useNavigate } from "react-router";

interface PricingFeature {
	text: string;
	pro?: boolean;
}

interface PricingTierProps {
	name: string;
	price: string;
	priceUnit: string;
	features: PricingFeature[];
	buttonLabel: string;
	highlighted?: boolean;
	badge?: string;
	onSelect?: () => void;
}

function PricingTier({ name, price, priceUnit, features, buttonLabel, highlighted = false, badge, onSelect }: PricingTierProps) {
	const FeatureIcon = highlighted ? CheckCircle : Check;

	return (
		<div className={`border-[3px] border-border-stark flex flex-col relative ${highlighted ? "bg-primary-container terminal-shadow md:-translate-y-4" : "bg-surface hard-shadow"}`}>
			{/* Badge */}
			{badge && (
				<div
					className="absolute -top-4 -right-4 bg-tertiary-container border-[3px] border-border-stark px-3 py-1 font-bold text-[12px] uppercase rotate-6"
					style={{ fontFamily: "var(--font-jetbrains-mono)" }}
				>
					{badge}
				</div>
			)}

			{/* Header */}
			<div className={`p-4 border-b-[3px] border-border-stark text-center ${highlighted ? "bg-primary text-on-primary" : "bg-surface-container-highest"}`}>
				<h3 className="text-2xl font-bold uppercase" style={{ fontFamily: "var(--font-space-grotesk)" }}>
					{name}
				</h3>
			</div>

			{/* Body */}
			<div className={`p-6 grow flex flex-col ${highlighted ? "bg-surface" : ""}`}>
				{/* Price */}
				<div className="text-center mb-6">
					<span className="text-5xl font-bold leading-tight tracking-tight" style={{ fontFamily: "var(--font-space-grotesk)" }}>
						{price}
					</span>
					<span className="text-sm text-on-surface-variant ml-1" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
						{priceUnit}
					</span>
				</div>

				{/* Features */}
				<ul className="mb-8 space-y-3 grow" style={{ fontFamily: "var(--font-jetbrains-mono)" }}>
					{features.map((f) => (
						<li key={f.text} className="flex items-center gap-2 text-sm">
							<FeatureIcon size={16} className="text-primary shrink-0" />
							{f.text}
						</li>
					))}
				</ul>

				{/* CTA */}
				<button
					onClick={onSelect}
					className={`w-full py-3 border-[3px] border-border-stark hard-shadow font-bold text-[12px] uppercase tracking-wider cursor-pointer ${
						highlighted ? "bg-primary text-on-primary hover:bg-surface-tint" : "bg-surface text-on-surface hover:bg-surface-container-highest"
					}`}
					style={{ fontFamily: "var(--font-jetbrains-mono)" }}
				>
					{buttonLabel}
				</button>
			</div>
		</div>
	);
}

const tiers: PricingTierProps[] = [
	{
		name: "DEV_TIER",
		price: "₹0",
		priceUnit: "/mo",
		features: [{ text: "Up to 10 PRs / month" }, { text: "The Senior Dev personality" }, { text: "GitHub Integration" }],
		buttonLabel: "Select Dev",
	},
	{
		name: "PRO_TIER",
		price: "₹499",
		priceUnit: "/mo",
		features: [
			{ text: "Unlimited PRs", pro: true },
			{ text: "All Personalities (Incl. Toxic)", pro: true },
			{ text: "Architecture Mapping & Context", pro: true },
			{ text: "Priority Support", pro: true },
		],
		buttonLabel: "Select Pro",
		highlighted: true,
		badge: "Popular",
	},
];

export default function PricingSection() {
	const navigate = useNavigate();

	return (
		<section className="max-w-4xl mx-auto mb-24">
			<h2 className="text-3xl font-bold mb-8 uppercase text-center border-b-[3px] border-border-stark pb-2 mx-auto w-max" style={{ fontFamily: "var(--font-space-grotesk)" }}>
				Pricing
			</h2>
			<div className="grid grid-cols-1 md:grid-cols-2 gap-8">
				{tiers.map((tier) => (
					<PricingTier key={tier.name} {...tier} onSelect={() => navigate("/login")} />
				))}
			</div>
		</section>
	);
}
