import { LayoutDashboard, FolderGit2 } from "lucide-react";
import SidebarLayout from "@/layouts/sidebar-layout";

const links = [
	{ name: "Overview", href: "/dashboard/overview", icon: LayoutDashboard },
	{ name: "Repositories", href: "/dashboard/repositories", icon: FolderGit2 },
];

export default function DashboardLayout() {
	return <SidebarLayout title="Dashboard" description="Overview and repositories." links={links} />;
}
