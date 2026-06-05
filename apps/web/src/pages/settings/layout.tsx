import { User, Monitor } from "lucide-react";
import SidebarLayout from "@/layouts/sidebar-layout";

const links = [
	{ name: "Account", href: "/settings/account", icon: User },
	{ name: "Active Sessions", href: "/settings/sessions", icon: Monitor },
];

export default function SettingsLayout() {
	return <SidebarLayout title="Settings" description="Manage your account and active sessions." links={links} />;
}
