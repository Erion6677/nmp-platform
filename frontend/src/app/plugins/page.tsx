"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import { useAuthStore } from "@/stores/auth";
import {
  PuzzlePieceIcon,
  ArrowDownTrayIcon,
  ArrowPathIcon,
  TrashIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  MagnifyingGlassIcon,
  FunnelIcon,
  CloudArrowDownIcon,
  CheckBadgeIcon,
  ArrowUpCircleIcon,
} from "@heroicons/react/24/outline";
import api from "@/lib/api/client";

interface RegistryPlugin {
  name: string;
  version: string;
  description: string;
  author: string;
  icon: string;
  category: string;
  tags: string[];
  download_url: string;
  homepage: string;
  license: string;
  min_version: string;
  size: number;
  downloads: number;
  updated_at: string;
  installed: boolean;
}

interface InstalledPlugin {
  name: string;
  version: string;
  description: string;
  author: string;
  enabled: boolean;
  installed_at: string;
  has_update: boolean;
  latest_version?: string;
}

export default function PluginsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  
  const [activeTab, setActiveTab] = useState<"marketplace" | "installed">("marketplace");
  const [availablePlugins, setAvailablePlugins] = useState<RegistryPlugin[]>([]);
  const [installedPlugins, setInstalledPlugins] = useState<InstalledPlugin[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isInstalling, setIsInstalling] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const categories = ["all", "系统工具", "分析工具", "告警通知", "报表工具", "可视化"];

  const fetchPlugins = useCallback(async () => {
    setIsLoading(true);
    try {
      const [availableRes, installedRes] = await Promise.all([
        api.get<{ success: boolean; data: { plugins: RegistryPlugin[] } }>("/api/v1/marketplace/plugins"),
        api.get<{ success: boolean; data: { plugins: InstalledPlugin[] } }>("/api/v1/marketplace/installed"),
      ]);
      
      if (availableRes.success) {
        setAvailablePlugins(availableRes.data.plugins || []);
      }
      if (installedRes.success) {
        setInstalledPlugins(installedRes.data.plugins || []);
      }
    } catch (error) {
      console.error("Failed to fetch plugins:", error);
      setMessage({ type: "error", text: "获取插件列表失败" });
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }
    fetchPlugins();
  }, [isAuthenticated, router, fetchPlugins]);

  const handleInstall = async (pluginName: string) => {
    setIsInstalling(pluginName);
    setMessage(null);

    try {
      const response = await api.post<{ success: boolean; error?: string; message?: string }>(
        `/api/v1/marketplace/install/${pluginName}`
      );
      
      if (response.success) {
        setMessage({ type: "success", text: response.message || "插件安装成功" });
        fetchPlugins();
      } else {
        setMessage({ type: "error", text: response.error || "安装失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "安装失败" });
    } finally {
      setIsInstalling(null);
    }
  };

  const handleUninstall = async (pluginName: string) => {
    if (!confirm(`确定要卸载插件 "${pluginName}" 吗？`)) return;

    try {
      const response = await api.post<{ success: boolean; error?: string; message?: string }>(
        `/api/v1/marketplace/uninstall/${pluginName}`
      );
      
      if (response.success) {
        setMessage({ type: "success", text: response.message || "插件卸载成功" });
        fetchPlugins();
      } else {
        setMessage({ type: "error", text: response.error || "卸载失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "卸载失败" });
    }
  };

  const handleUpdate = async (pluginName: string) => {
    setIsInstalling(pluginName);
    setMessage(null);

    try {
      const response = await api.post<{ success: boolean; error?: string; message?: string }>(
        `/api/v1/marketplace/update/${pluginName}`
      );
      
      if (response.success) {
        setMessage({ type: "success", text: response.message || "插件更新成功" });
        fetchPlugins();
      } else {
        setMessage({ type: "error", text: response.error || "更新失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "更新失败" });
    } finally {
      setIsInstalling(null);
    }
  };

  const handleRefresh = async () => {
    setIsLoading(true);
    try {
      await api.post("/api/v1/marketplace/refresh");
      await fetchPlugins();
      setMessage({ type: "success", text: "插件列表已刷新" });
    } catch (error) {
      setMessage({ type: "error", text: "刷新失败" });
    } finally {
      setIsLoading(false);
    }
  };

  const filteredPlugins = availablePlugins.filter(plugin => {
    const matchesSearch = plugin.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                         plugin.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = selectedCategory === "all" || plugin.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const getIconComponent = (iconName: string) => {
    // 简单的图标映射
    return PuzzlePieceIcon;
  };

  if (!isAuthenticated) return null;

  return (
    <MainLayout>
      <Header title="插件中心" breadcrumb={["系统管理", "插件中心"]} />
      
      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-6xl mx-auto space-y-6">
          {/* 消息提示 */}
          {message && (
            <div className={`p-4 rounded-xl flex items-center gap-3 ${
              message.type === "success"
                ? "bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 border border-green-200 dark:border-green-500/20"
                : "bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 border border-red-200 dark:border-red-500/20"
            }`}>
              {message.type === "success" ? <CheckCircleIcon className="w-5 h-5" /> : <ExclamationCircleIcon className="w-5 h-5" />}
              {message.text}
              <button onClick={() => setMessage(null)} className="ml-auto text-current opacity-50 hover:opacity-100">×</button>
            </div>
          )}

          {/* 标签页切换 */}
          <div className="flex items-center justify-between">
            <div className="flex bg-slate-100 dark:bg-white/5 rounded-xl p-1">
              <button
                onClick={() => setActiveTab("marketplace")}
                className={`px-6 py-2 rounded-lg text-sm font-medium transition-all ${
                  activeTab === "marketplace"
                    ? "bg-white dark:bg-[#1e2d47] text-cyan-600 dark:text-cyan-400 shadow-sm"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200"
                }`}
              >
                <CloudArrowDownIcon className="w-4 h-4 inline mr-2" />
                插件市场
              </button>
              <button
                onClick={() => setActiveTab("installed")}
                className={`px-6 py-2 rounded-lg text-sm font-medium transition-all ${
                  activeTab === "installed"
                    ? "bg-white dark:bg-[#1e2d47] text-cyan-600 dark:text-cyan-400 shadow-sm"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200"
                }`}
              >
                <CheckBadgeIcon className="w-4 h-4 inline mr-2" />
                已安装 ({installedPlugins.length})
              </button>
            </div>
            <button
              onClick={handleRefresh}
              disabled={isLoading}
              className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-white/5 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-white/10 transition-all flex items-center gap-2 disabled:opacity-50"
            >
              <ArrowPathIcon className={`w-4 h-4 ${isLoading ? "animate-spin" : ""}`} />
              刷新
            </button>
          </div>

          {/* 插件市场 */}
          {activeTab === "marketplace" && (
            <>
              {/* 搜索和筛选 */}
              <div className="flex gap-4">
                <div className="flex-1 relative">
                  <MagnifyingGlassIcon className="w-5 h-5 absolute left-4 top-1/2 -translate-y-1/2 text-slate-400" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="搜索插件..."
                    className="w-full pl-12 pr-4 py-3 rounded-xl bg-white dark:bg-[#1e2d47] border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                  />
                </div>
                <div className="relative">
                  <FunnelIcon className="w-5 h-5 absolute left-4 top-1/2 -translate-y-1/2 text-slate-400" />
                  <select
                    value={selectedCategory}
                    onChange={(e) => setSelectedCategory(e.target.value)}
                    className="pl-12 pr-8 py-3 rounded-xl bg-white dark:bg-[#1e2d47] border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-cyan-500/50 appearance-none cursor-pointer"
                  >
                    {categories.map(cat => (
                      <option key={cat} value={cat}>{cat === "all" ? "全部分类" : cat}</option>
                    ))}
                  </select>
                </div>
              </div>

              {/* 插件列表 */}
              {isLoading ? (
                <div className="p-12 text-center">
                  <ArrowPathIcon className="w-8 h-8 animate-spin text-cyan-500 mx-auto mb-3" />
                  <p className="text-slate-500 dark:text-slate-400">加载中...</p>
                </div>
              ) : filteredPlugins.length === 0 ? (
                <div className="p-12 text-center bg-white dark:bg-[#1e2d47] rounded-2xl border border-slate-200 dark:border-white/10">
                  <PuzzlePieceIcon className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
                  <p className="text-slate-500 dark:text-slate-400">暂无可用插件</p>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {filteredPlugins.map((plugin) => {
                    const IconComponent = getIconComponent(plugin.icon);
                    return (
                      <div
                        key={plugin.name}
                        className="bg-white dark:bg-[#1e2d47] rounded-2xl p-5 border border-slate-200 dark:border-white/10 hover:border-cyan-500/30 transition-all"
                      >
                        <div className="flex items-start gap-4">
                          <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-cyan-500/20 to-blue-500/20 flex items-center justify-center flex-shrink-0">
                            <IconComponent className="w-7 h-7 text-cyan-500" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <h3 className="font-semibold text-slate-800 dark:text-white">{plugin.name}</h3>
                              <span className="text-xs px-2 py-0.5 rounded-full bg-slate-100 dark:bg-white/10 text-slate-500 dark:text-slate-400">
                                v{plugin.version}
                              </span>
                              {plugin.installed && (
                                <span className="text-xs px-2 py-0.5 rounded-full bg-green-100 dark:bg-green-500/20 text-green-600 dark:text-green-400">
                                  已安装
                                </span>
                              )}
                            </div>
                            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1 line-clamp-2">
                              {plugin.description}
                            </p>
                            <div className="flex items-center gap-4 mt-3 text-xs text-slate-400">
                              <span>{plugin.author}</span>
                              <span>{plugin.category}</span>
                              <span>{formatSize(plugin.size)}</span>
                              <span>{plugin.downloads} 次下载</span>
                            </div>
                          </div>
                        </div>
                        <div className="flex items-center justify-between mt-4 pt-4 border-t border-slate-100 dark:border-white/5">
                          <div className="flex flex-wrap gap-1">
                            {plugin.tags?.slice(0, 3).map(tag => (
                              <span key={tag} className="text-xs px-2 py-0.5 rounded-full bg-slate-100 dark:bg-white/5 text-slate-500 dark:text-slate-400">
                                {tag}
                              </span>
                            ))}
                          </div>
                          {plugin.installed ? (
                            <button
                              onClick={() => handleUninstall(plugin.name)}
                              className="px-4 py-1.5 rounded-lg text-sm text-red-500 hover:bg-red-500/10 transition-all flex items-center gap-1"
                            >
                              <TrashIcon className="w-4 h-4" />
                              卸载
                            </button>
                          ) : (
                            <button
                              onClick={() => handleInstall(plugin.name)}
                              disabled={isInstalling === plugin.name}
                              className="px-4 py-1.5 rounded-lg text-sm bg-gradient-to-r from-cyan-500 to-blue-500 text-white hover:from-cyan-600 hover:to-blue-600 transition-all flex items-center gap-1 disabled:opacity-50"
                            >
                              {isInstalling === plugin.name ? (
                                <>
                                  <ArrowPathIcon className="w-4 h-4 animate-spin" />
                                  安装中
                                </>
                              ) : (
                                <>
                                  <ArrowDownTrayIcon className="w-4 h-4" />
                                  安装
                                </>
                              )}
                            </button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </>
          )}

          {/* 已安装插件 */}
          {activeTab === "installed" && (
            <div className="bg-white dark:bg-[#1e2d47] rounded-2xl border border-slate-200 dark:border-white/10 overflow-hidden">
              {installedPlugins.length === 0 ? (
                <div className="p-12 text-center">
                  <PuzzlePieceIcon className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
                  <p className="text-slate-500 dark:text-slate-400">暂无已安装插件</p>
                  <p className="text-sm text-slate-400 dark:text-slate-500 mt-1">前往插件市场安装插件</p>
                </div>
              ) : (
                <div className="divide-y divide-slate-200 dark:divide-white/10">
                  {installedPlugins.map((plugin) => (
                    <div key={plugin.name} className="p-5 hover:bg-slate-50 dark:hover:bg-white/5 transition-all">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-4">
                          <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-emerald-500/20 to-green-500/20 flex items-center justify-center">
                            <PuzzlePieceIcon className="w-6 h-6 text-emerald-500" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <h3 className="font-medium text-slate-800 dark:text-white">{plugin.name}</h3>
                              <span className="text-xs px-2 py-0.5 rounded-full bg-slate-100 dark:bg-white/10 text-slate-500 dark:text-slate-400">
                                v{plugin.version}
                              </span>
                              {plugin.has_update && (
                                <span className="text-xs px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-500/20 text-amber-600 dark:text-amber-400 flex items-center gap-1">
                                  <ArrowUpCircleIcon className="w-3 h-3" />
                                  有更新 v{plugin.latest_version}
                                </span>
                              )}
                            </div>
                            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                              {plugin.description}
                            </p>
                            <p className="text-xs text-slate-400 mt-1">
                              作者: {plugin.author}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          {plugin.has_update && (
                            <button
                              onClick={() => handleUpdate(plugin.name)}
                              disabled={isInstalling === plugin.name}
                              className="px-3 py-1.5 rounded-lg text-sm text-amber-600 hover:bg-amber-500/10 transition-all flex items-center gap-1"
                            >
                              {isInstalling === plugin.name ? (
                                <ArrowPathIcon className="w-4 h-4 animate-spin" />
                              ) : (
                                <ArrowUpCircleIcon className="w-4 h-4" />
                              )}
                              更新
                            </button>
                          )}
                          <button
                            onClick={() => handleUninstall(plugin.name)}
                            className="px-3 py-1.5 rounded-lg text-sm text-red-500 hover:bg-red-500/10 transition-all flex items-center gap-1"
                          >
                            <TrashIcon className="w-4 h-4" />
                            卸载
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </MainLayout>
  );
}
