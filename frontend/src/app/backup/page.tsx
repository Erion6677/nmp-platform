"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import { useAuthStore } from "@/stores/auth";
import {
  ArchiveBoxIcon,
  ArrowDownTrayIcon,
  ArrowPathIcon,
  TrashIcon,
  PlusIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  ClockIcon,
  ServerStackIcon,
  CircleStackIcon,
  Cog6ToothIcon,
} from "@heroicons/react/24/outline";
import api from "@/lib/api/client";

interface BackupInfo {
  id: string;
  name: string;
  description: string;
  size: number;
  size_human: string;
  created_at: string;
  type: string;
  status: string;
  file_path: string;
  components: string[] | null;
}

interface BackupStatus {
  total_backups: number;
  total_size: number;
  total_size_human: string;
  backup_dir: string;
}

export default function BackupPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [status, setStatus] = useState<BackupStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [isRestoring, setIsRestoring] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  
  // 创建备份表单
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createForm, setCreateForm] = useState({
    name: "",
    description: "",
    components: ["postgres", "influxdb", "config"] as string[],
  });

  const fetchBackups = useCallback(async () => {
    try {
      const [listRes, statusRes] = await Promise.all([
        api.get<{ success: boolean; data: { backups: BackupInfo[]; total: number } }>("/api/v1/system-backup/list"),
        api.get<{ success: boolean; data: BackupStatus }>("/api/v1/system-backup/status"),
      ]);
      
      if (listRes.success) {
        setBackups(listRes.data.backups || []);
      }
      if (statusRes.success) {
        setStatus(statusRes.data);
      }
    } catch (error) {
      console.error("Failed to fetch backups:", error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }
    fetchBackups();
  }, [isAuthenticated, router, fetchBackups]);

  const handleCreateBackup = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsCreating(true);
    setMessage(null);

    try {
      const response = await api.post<{ success: boolean; data?: BackupInfo; error?: string; message?: string }>(
        "/api/v1/system-backup/create",
        createForm
      );
      
      if (response.success) {
        setMessage({ type: "success", text: response.message || "备份创建成功" });
        setShowCreateForm(false);
        setCreateForm({ name: "", description: "", components: ["postgres", "influxdb", "config"] });
        fetchBackups();
      } else {
        setMessage({ type: "error", text: response.error || "创建备份失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "创建备份失败" });
    } finally {
      setIsCreating(false);
    }
  };

  const handleDownload = async (backup: BackupInfo) => {
    try {
      const token = localStorage.getItem("token");
      const response = await fetch(`/api/v1/system-backup/download/${backup.id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      
      if (!response.ok) throw new Error("下载失败");
      
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${backup.name}.tar.gz`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "下载失败" });
    }
  };

  const handleRestore = async (backup: BackupInfo) => {
    if (!confirm(`确定要还原备份 "${backup.name}" 吗？这将覆盖当前数据！`)) return;
    
    setIsRestoring(backup.id);
    setMessage(null);

    try {
      const response = await api.post<{ success: boolean; error?: string; message?: string }>(
        `/api/v1/system-backup/restore/${backup.id}`,
        { components: ["postgres", "config"] }
      );
      
      if (response.success) {
        setMessage({ type: "success", text: response.message || "备份还原成功，请重启服务" });
      } else {
        setMessage({ type: "error", text: response.error || "还原失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "还原失败" });
    } finally {
      setIsRestoring(null);
    }
  };

  const handleDelete = async (backup: BackupInfo) => {
    if (!confirm(`确定要删除备份 "${backup.name}" 吗？此操作不可恢复！`)) return;

    try {
      const response = await api.delete<{ success: boolean; error?: string; message?: string }>(
        `/api/v1/system-backup/${backup.id}`
      );
      
      if (response.success) {
        setMessage({ type: "success", text: "备份删除成功" });
        fetchBackups();
      } else {
        setMessage({ type: "error", text: response.error || "删除失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "删除失败" });
    }
  };

  const toggleComponent = (component: string) => {
    setCreateForm(prev => ({
      ...prev,
      components: prev.components.includes(component)
        ? prev.components.filter(c => c !== component)
        : [...prev.components, component],
    }));
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  if (!isAuthenticated) return null;

  return (
    <MainLayout>
      <Header title="系统备份" breadcrumb={["系统管理", "系统备份"]} />
      
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

          {/* 状态卡片 */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-white dark:bg-[#1e2d47] rounded-2xl p-5 border border-slate-200 dark:border-white/10">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-xl bg-cyan-500/10 flex items-center justify-center">
                  <ArchiveBoxIcon className="w-6 h-6 text-cyan-500" />
                </div>
                <div>
                  <p className="text-sm text-slate-500 dark:text-slate-400">备份总数</p>
                  <p className="text-2xl font-bold text-slate-800 dark:text-white">{status?.total_backups || 0}</p>
                </div>
              </div>
            </div>
            <div className="bg-white dark:bg-[#1e2d47] rounded-2xl p-5 border border-slate-200 dark:border-white/10">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-xl bg-violet-500/10 flex items-center justify-center">
                  <ServerStackIcon className="w-6 h-6 text-violet-500" />
                </div>
                <div>
                  <p className="text-sm text-slate-500 dark:text-slate-400">占用空间</p>
                  <p className="text-2xl font-bold text-slate-800 dark:text-white">{status?.total_size_human || "0 B"}</p>
                </div>
              </div>
            </div>
            <div className="bg-white dark:bg-[#1e2d47] rounded-2xl p-5 border border-slate-200 dark:border-white/10">
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 rounded-xl bg-emerald-500/10 flex items-center justify-center">
                  <Cog6ToothIcon className="w-6 h-6 text-emerald-500" />
                </div>
                <div>
                  <p className="text-sm text-slate-500 dark:text-slate-400">备份目录</p>
                  <p className="text-sm font-medium text-slate-800 dark:text-white truncate">{status?.backup_dir || "/opt/nmp/backups"}</p>
                </div>
              </div>
            </div>
          </div>

          {/* 操作栏 */}
          <div className="flex justify-between items-center">
            <h2 className="text-lg font-semibold text-slate-800 dark:text-white">备份列表</h2>
            <div className="flex gap-3">
              <button
                onClick={fetchBackups}
                className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-white/5 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-white/10 transition-all flex items-center gap-2"
              >
                <ArrowPathIcon className="w-4 h-4" />
                刷新
              </button>
              <button
                onClick={() => setShowCreateForm(true)}
                className="px-4 py-2 rounded-xl bg-gradient-to-r from-cyan-500 to-blue-500 text-white hover:from-cyan-600 hover:to-blue-600 transition-all flex items-center gap-2"
              >
                <PlusIcon className="w-4 h-4" />
                创建备份
              </button>
            </div>
          </div>

          {/* 创建备份表单 */}
          {showCreateForm && (
            <div className="bg-white dark:bg-[#1e2d47] rounded-2xl p-6 border border-slate-200 dark:border-white/10">
              <h3 className="text-lg font-semibold text-slate-800 dark:text-white mb-4">创建新备份</h3>
              <form onSubmit={handleCreateBackup} className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">备份名称</label>
                    <input
                      type="text"
                      value={createForm.name}
                      onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                      placeholder="留空则自动生成"
                      className="w-full px-4 py-3 rounded-xl bg-white dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">备份描述</label>
                    <input
                      type="text"
                      value={createForm.description}
                      onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                      placeholder="可选"
                      className="w-full px-4 py-3 rounded-xl bg-white dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">备份组件</label>
                  <div className="flex flex-wrap gap-3">
                    {[
                      { key: "postgres", label: "PostgreSQL", icon: CircleStackIcon },
                      { key: "influxdb", label: "InfluxDB", icon: ServerStackIcon },
                      { key: "config", label: "配置文件", icon: Cog6ToothIcon },
                    ].map(({ key, label, icon: Icon }) => (
                      <button
                        key={key}
                        type="button"
                        onClick={() => toggleComponent(key)}
                        className={`px-4 py-2 rounded-xl flex items-center gap-2 transition-all ${
                          createForm.components.includes(key)
                            ? "bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 border border-cyan-500/30"
                            : "bg-slate-100 dark:bg-white/5 text-slate-600 dark:text-slate-400 border border-transparent"
                        }`}
                      >
                        <Icon className="w-4 h-4" />
                        {label}
                      </button>
                    ))}
                  </div>
                </div>
                <div className="flex gap-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setShowCreateForm(false)}
                    className="px-6 py-2 rounded-xl bg-slate-100 dark:bg-white/5 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-white/10"
                  >
                    取消
                  </button>
                  <button
                    type="submit"
                    disabled={isCreating || createForm.components.length === 0}
                    className="px-6 py-2 rounded-xl bg-gradient-to-r from-cyan-500 to-blue-500 text-white hover:from-cyan-600 hover:to-blue-600 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                  >
                    {isCreating ? (
                      <>
                        <ArrowPathIcon className="w-4 h-4 animate-spin" />
                        创建中...
                      </>
                    ) : (
                      "开始备份"
                    )}
                  </button>
                </div>
              </form>
            </div>
          )}

          {/* 备份列表 */}
          <div className="bg-white dark:bg-[#1e2d47] rounded-2xl border border-slate-200 dark:border-white/10 overflow-hidden">
            {isLoading ? (
              <div className="p-12 text-center">
                <ArrowPathIcon className="w-8 h-8 animate-spin text-cyan-500 mx-auto mb-3" />
                <p className="text-slate-500 dark:text-slate-400">加载中...</p>
              </div>
            ) : backups.length === 0 ? (
              <div className="p-12 text-center">
                <ArchiveBoxIcon className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
                <p className="text-slate-500 dark:text-slate-400">暂无备份</p>
                <p className="text-sm text-slate-400 dark:text-slate-500 mt-1">点击"创建备份"开始第一次备份</p>
              </div>
            ) : (
              <div className="divide-y divide-slate-200 dark:divide-white/10">
                {backups.map((backup) => (
                  <div key={backup.id} className="p-5 hover:bg-slate-50 dark:hover:bg-white/5 transition-all">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-cyan-500/20 to-blue-500/20 flex items-center justify-center">
                          <ArchiveBoxIcon className="w-6 h-6 text-cyan-500" />
                        </div>
                        <div>
                          <h3 className="font-medium text-slate-800 dark:text-white">{backup.name}</h3>
                          <div className="flex items-center gap-4 mt-1 text-sm text-slate-500 dark:text-slate-400">
                            <span className="flex items-center gap-1">
                              <ClockIcon className="w-4 h-4" />
                              {formatDate(backup.created_at)}
                            </span>
                            <span>{backup.size_human}</span>
                            <span className={`px-2 py-0.5 rounded-full text-xs ${
                              backup.status === "completed"
                                ? "bg-green-100 dark:bg-green-500/20 text-green-600 dark:text-green-400"
                                : "bg-yellow-100 dark:bg-yellow-500/20 text-yellow-600 dark:text-yellow-400"
                            }`}>
                              {backup.status === "completed" ? "已完成" : backup.status}
                            </span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleDownload(backup)}
                          className="p-2 rounded-lg text-slate-500 hover:text-cyan-500 hover:bg-cyan-500/10 transition-all"
                          title="下载"
                        >
                          <ArrowDownTrayIcon className="w-5 h-5" />
                        </button>
                        <button
                          onClick={() => handleRestore(backup)}
                          disabled={isRestoring === backup.id}
                          className="p-2 rounded-lg text-slate-500 hover:text-amber-500 hover:bg-amber-500/10 transition-all disabled:opacity-50"
                          title="还原"
                        >
                          {isRestoring === backup.id ? (
                            <ArrowPathIcon className="w-5 h-5 animate-spin" />
                          ) : (
                            <ArrowPathIcon className="w-5 h-5" />
                          )}
                        </button>
                        <button
                          onClick={() => handleDelete(backup)}
                          className="p-2 rounded-lg text-slate-500 hover:text-red-500 hover:bg-red-500/10 transition-all"
                          title="删除"
                        >
                          <TrashIcon className="w-5 h-5" />
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
