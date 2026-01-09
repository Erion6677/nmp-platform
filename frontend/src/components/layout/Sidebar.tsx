"use client";

import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { useState, useEffect } from "react";
import { useAuthStore } from "@/stores/auth";
import { useSidebarStore } from "@/stores/sidebar";
import {
  Squares2X2Icon,
  ServerStackIcon,
  UsersIcon,
  Cog6ToothIcon,
  ArrowRightStartOnRectangleIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  PuzzlePieceIcon,
  ArchiveBoxIcon,
  DocumentTextIcon,
  BellIcon,
  ChartBarIcon,
  ShareIcon,
} from "@heroicons/react/24/outline";
import api from "@/lib/api/client";

// 图标映射
const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  archive: ArchiveBoxIcon,
  "document-text": DocumentTextIcon,
  bell: BellIcon,
  "chart-bar": ChartBarIcon,
  share: ShareIcon,
  puzzle: PuzzlePieceIcon,
};

interface MenuItem {
  key: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  path?: string;
}

interface PluginMenu {
  key: string;
  label: string;
  icon: string;
  path: string;
  order: number;
}

// 基础菜单项
const baseMenuItems: MenuItem[] = [
  { key: "/", label: "设备概览", icon: Squares2X2Icon },
  { key: "/devices", label: "设备管理", icon: ServerStackIcon },
  { key: "/users", label: "用户管理", icon: UsersIcon },
];

// 底部固定菜单
const bottomMenuItems: MenuItem[] = [
  { key: "/plugins", label: "插件中心", icon: PuzzlePieceIcon },
  { key: "/settings", label: "系统设置", icon: Cog6ToothIcon },
];

export default function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user, logout, isAuthenticated } = useAuthStore();
  const { isCollapsed, toggle } = useSidebarStore();
  const [pluginMenus, setPluginMenus] = useState<MenuItem[]>([]);

  // 获取插件菜单
  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchPluginMenus = async () => {
      try {
        const response = await api.get<{ success: boolean; data: { menus: PluginMenu[] | null } }>(
          "/api/v1/marketplace/menus"
        );
        if (response.success && response.data.menus && Array.isArray(response.data.menus)) {
          const menus = response.data.menus
            .sort((a, b) => a.order - b.order)
            .map((menu) => ({
              key: menu.path,
              label: menu.label,
              icon: iconMap[menu.icon] || PuzzlePieceIcon,
            }));
          setPluginMenus(menus);
        }
      } catch (error) {
        // 静默处理错误，不影响页面加载
        console.error("Failed to fetch plugin menus:", error);
      }
    };

    fetchPluginMenus();
  }, [isAuthenticated]);

  // 合并所有菜单
  const allMenuItems = [...baseMenuItems, ...pluginMenus, ...bottomMenuItems];

  const isActive = (key: string) => {
    if (key === "/") return pathname === "/";
    return pathname.startsWith(key);
  };

  const handleLogout = async () => {
    await logout();
    router.push("/login");
  };

  const handleUserClick = () => {
    router.push("/profile");
  };

  return (
    <aside 
      className={`h-screen bg-white/90 dark:bg-[#1a2744]/90 backdrop-blur-xl border-r border-slate-200 dark:border-white/10 flex flex-col transition-all duration-300 ${
        isCollapsed ? "w-16" : "w-64"
      }`}
    >
      {/* Logo */}
      <div className="h-16 flex items-center px-3 border-b border-slate-200 dark:border-white/10 justify-between">
        <div className="flex items-center">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-cyan-400 to-blue-500 flex items-center justify-center shadow-lg shadow-cyan-500/30 flex-shrink-0">
            <ServerStackIcon className="w-5 h-5 text-white" />
          </div>
          {!isCollapsed && (
            <span className="ml-3 text-lg font-semibold bg-gradient-to-r from-cyan-500 to-blue-500 bg-clip-text text-transparent">
              NMP
            </span>
          )}
        </div>
        <button
          onClick={toggle}
          className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-white/10 transition-all"
          title={isCollapsed ? "展开侧边栏" : "收起侧边栏"}
        >
          {isCollapsed ? (
            <ChevronRightIcon className="w-4 h-4 text-slate-400" />
          ) : (
            <ChevronLeftIcon className="w-4 h-4 text-slate-400" />
          )}
        </button>
      </div>

      {/* 导航菜单 */}
      <nav className="flex-1 p-2 space-y-1 overflow-y-auto">
        {allMenuItems.map((item) => {
          const Icon = item.icon;
          const active = isActive(item.key);
          return (
            <Link
              key={item.key}
              href={item.key}
              className={`flex items-center gap-3 px-3 py-3 rounded-xl transition-all relative ${
                isCollapsed ? "justify-center" : ""
              } ${
                active
                  ? "bg-gradient-to-r from-cyan-500/10 to-blue-500/10 dark:from-cyan-500/20 dark:to-blue-500/20 text-cyan-600 dark:text-cyan-400 border border-cyan-500/20"
                  : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-white/5"
              }`}
              title={isCollapsed ? item.label : undefined}
            >
              {active && !isCollapsed && (
                <div className="sidebar-active-indicator" />
              )}
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!isCollapsed && <span className="font-medium">{item.label}</span>}
            </Link>
          );
        })}
      </nav>

      {/* 底部用户信息 */}
      <div className="p-2 border-t border-slate-200 dark:border-white/10">
        <div 
          className={`flex items-center gap-3 px-3 py-3 rounded-xl hover:bg-slate-100 dark:hover:bg-white/5 transition-all cursor-pointer ${
            isCollapsed ? "justify-center" : ""
          }`}
          onClick={handleUserClick}
          title={isCollapsed ? "个人中心" : undefined}
        >
          <div className="w-9 h-9 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-sm font-medium text-white shadow-lg shadow-violet-500/30 flex-shrink-0">
            {user?.username?.charAt(0).toUpperCase() || "A"}
          </div>
          {!isCollapsed && (
            <>
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium text-slate-800 dark:text-slate-200 truncate">
                  {user?.full_name || user?.username || "Admin"}
                </div>
                <div className="text-xs text-slate-500">
                  {user?.roles?.[0] || "管理员"}
                </div>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleLogout();
                }}
                className="p-1.5 rounded-lg hover:bg-slate-200 dark:hover:bg-white/10 transition-all"
                title="退出登录"
              >
                <ArrowRightStartOnRectangleIcon className="w-4 h-4 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300" />
              </button>
            </>
          )}
        </div>
      </div>
    </aside>
  );
}
