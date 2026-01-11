"use client";

import { useState, useEffect } from "react";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import { deviceApi } from "@/lib/api/device";
import type { Device } from "@/lib/api/types";
import {
  CheckCircleIcon,
  XCircleIcon,
  ExclamationTriangleIcon,
  CpuChipIcon,
  ArrowPathIcon,
  ChevronRightIcon,
  ExclamationCircleIcon,
} from "@heroicons/react/24/outline";
import Link from "next/link";

const typeLabels: Record<string, { label: string; color: string }> = {
  router: { label: "路由器", color: "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20" },
  switch: { label: "交换机", color: "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20" },
  firewall: { label: "防火墙", color: "bg-rose-500/10 text-rose-600 dark:text-rose-400 border-rose-500/20" },
  server: { label: "服务器", color: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20" },
  other: { label: "其他", color: "bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20" },
};

const colorConfig: Record<string, { iconBg: string; iconColor: string; valueColor: string }> = {
  emerald: {
    iconBg: "icon-bg-emerald",
    iconColor: "text-emerald-500 dark:text-emerald-400",
    valueColor: "text-emerald-600 dark:text-emerald-400",
  },
  rose: {
    iconBg: "icon-bg-rose",
    iconColor: "text-rose-500 dark:text-rose-400",
    valueColor: "text-rose-600 dark:text-rose-400",
  },
  amber: {
    iconBg: "icon-bg-amber",
    iconColor: "text-amber-500 dark:text-amber-400",
    valueColor: "text-amber-600 dark:text-amber-400",
  },
  cyan: {
    iconBg: "icon-bg-cyan",
    iconColor: "text-cyan-500 dark:text-cyan-400",
    valueColor: "text-cyan-600 dark:text-cyan-400",
  },
};

export default function OverviewPage() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const loadDevices = async () => {
    try {
      const response = await deviceApi.listDevices({ page: 1, page_size: 100 });
      if (response.success && response.data) {
        setDevices(response.data.devices || []);
      }
    } catch (error) {
      console.error("加载设备列表失败:", error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    loadDevices();
    // 每 30 秒刷新一次
    const interval = setInterval(loadDevices, 30000);
    return () => clearInterval(interval);
  }, []);

  const handleRefresh = () => {
    setRefreshing(true);
    loadDevices();
  };

  // 统计数据
  const stats = [
    { title: "在线设备", value: devices.filter((d) => d.status === "online").length, icon: CheckCircleIcon, color: "emerald" },
    { title: "离线设备", value: devices.filter((d) => d.status === "offline").length, icon: XCircleIcon, color: "rose" },
    { title: "异常设备", value: devices.filter((d) => d.status === "error" || d.status === "unknown").length, icon: ExclamationTriangleIcon, color: "amber" },
    { title: "设备总数", value: devices.length, icon: CpuChipIcon, color: "cyan" },
  ];

  // 最近告警（模拟，后续可对接告警 API）
  const alerts = devices
    .filter((d) => d.status === "offline" || d.status === "error")
    .slice(0, 5)
    .map((d, i) => ({
      id: i,
      message: `${d.name} ${d.status === "offline" ? "离线" : "异常"}`,
      time: d.last_seen ? formatTimeAgo(d.last_seen) : "未知",
      level: d.status === "offline" ? "error" : "warning",
    }));

  function formatTimeAgo(dateStr: string) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    if (diff < 60000) return "刚刚";
    if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`;
    return `${Math.floor(diff / 86400000)} 天前`;
  }

  if (loading) {
    return (
      <MainLayout>
        <Header title="设备概览" breadcrumb={["设备概览"]} />
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center">
            <div className="w-12 h-12 border-4 border-cyan-500/30 border-t-cyan-500 rounded-full animate-spin mx-auto mb-4" />
            <p className="text-slate-500">加载中...</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout>
      <Header title="设备概览" breadcrumb={["设备概览"]} />
      <div className="flex-1 overflow-y-auto p-6">
        {/* 统计卡片 */}
        <div className="grid grid-cols-4 gap-5 mb-6">
          {stats.map((stat) => {
            const Icon = stat.icon;
            const config = colorConfig[stat.color];
            return (
              <div
                key={stat.title}
                className="glass-card rounded-2xl p-5 hover:-translate-y-1 transition-all duration-300 cursor-pointer"
              >
                <div className="flex items-center gap-4">
                  <div className={`w-12 h-12 rounded-xl ${config.iconBg} flex items-center justify-center`}>
                    <Icon className={`w-6 h-6 ${config.iconColor}`} />
                  </div>
                  <div>
                    <div className={`text-3xl font-bold ${config.valueColor}`}>{stat.value}</div>
                    <div className="text-sm text-slate-500 dark:text-slate-400">{stat.title}</div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {/* 主内容网格 */}
        <div className="grid grid-cols-2 gap-5">
          {/* 设备状态 */}
          <div className="glass-card rounded-2xl overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-200 dark:border-white/10 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-cyan-400" />
                <span className="font-medium text-slate-800 dark:text-white">设备状态</span>
              </div>
              <button
                onClick={handleRefresh}
                className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
              >
                <ArrowPathIcon className={`w-4 h-4 text-slate-500 dark:text-slate-400 ${refreshing ? "animate-spin" : ""}`} />
              </button>
            </div>
            <div className="p-3 space-y-2 max-h-80 overflow-y-auto">
              {devices.slice(0, 10).map((device) => (
                <Link
                  key={device.id}
                  href={`/devices/${device.id}`}
                  className="flex items-center gap-3 p-3 rounded-xl hover:bg-slate-100 dark:hover:bg-white/5 border border-transparent hover:border-slate-200 dark:hover:border-white/10 transition-all group cursor-pointer"
                >
                  <div className={`w-2.5 h-2.5 rounded-full ${
                    device.status === "online" ? "status-dot-online" :
                    device.status === "offline" ? "status-dot-offline" : "status-dot-warning"
                  }`} />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-slate-800 dark:text-slate-200">{device.name}</div>
                    <div className="text-xs text-slate-500 font-mono">{device.host}</div>
                  </div>
                  <span className={`px-2 py-0.5 text-xs font-medium rounded-full border ${typeLabels[device.type]?.color || "bg-slate-100"}`}>
                    {typeLabels[device.type]?.label || device.type}
                  </span>
                  <ChevronRightIcon className="w-4 h-4 text-slate-400 group-hover:text-cyan-500 group-hover:translate-x-1 transition-all" />
                </Link>
              ))}
              {devices.length === 0 && (
                <div className="py-8 text-center text-slate-500">
                  暂无设备
                </div>
              )}
            </div>
          </div>

          {/* 网络流量 */}
          <div className="glass-card rounded-2xl overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-200 dark:border-white/10 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-cyan-400" />
                <span className="font-medium text-slate-800 dark:text-white">网络流量</span>
              </div>
              <div className="flex items-center gap-4 text-xs">
                <span className="flex items-center gap-1.5">
                  <span className="w-2 h-2 rounded-full bg-gradient-to-r from-cyan-400 to-blue-400" />
                  <span className="text-slate-500">入站</span>
                </span>
                <span className="flex items-center gap-1.5">
                  <span className="w-2 h-2 rounded-full bg-gradient-to-r from-violet-400 to-purple-400" />
                  <span className="text-slate-500">出站</span>
                </span>
              </div>
            </div>
            <div className="p-5 h-64">
              <div className="w-full h-full chart-placeholder">
                <span className="text-slate-400 dark:text-slate-500 text-sm">ECharts 图表区域</span>
              </div>
            </div>
          </div>

          {/* 最近告警 */}
          <div className="glass-card rounded-2xl overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-200 dark:border-white/10 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-cyan-400" />
                <span className="font-medium text-slate-800 dark:text-white">最近告警</span>
              </div>
              <span className="px-2 py-0.5 text-xs font-medium rounded-full bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20">
                {alerts.length}
              </span>
            </div>
            <div className="p-3 space-y-2 max-h-52 overflow-y-auto">
              {alerts.length > 0 ? alerts.map((alert) => (
                <div
                  key={alert.id}
                  className={`flex items-start gap-3 p-3 rounded-xl border-l-2 ${
                    alert.level === "error"
                      ? "bg-rose-500/5 dark:bg-rose-500/10 border-rose-500"
                      : "bg-amber-500/5 dark:bg-amber-500/10 border-amber-500"
                  }`}
                >
                  {alert.level === "error" ? (
                    <ExclamationCircleIcon className="w-4 h-4 text-rose-500 mt-0.5 flex-shrink-0" />
                  ) : (
                    <ExclamationTriangleIcon className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="text-sm text-slate-700 dark:text-slate-200">{alert.message}</div>
                    <div className="text-xs text-slate-500 mt-1">{alert.time}</div>
                  </div>
                </div>
              )) : (
                <div className="py-8 text-center text-slate-500">
                  暂无告警
                </div>
              )}
            </div>
          </div>

          {/* 系统信息 */}
          <div className="glass-card rounded-2xl overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-200 dark:border-white/10">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-cyan-400" />
                <span className="font-medium text-slate-800 dark:text-white">系统信息</span>
              </div>
            </div>
            <div className="p-5 space-y-4">
              <div className="flex items-center gap-3">
                <span className="text-sm text-slate-500 w-24">系统版本</span>
                <span className="text-sm text-slate-800 dark:text-slate-200 font-medium">v1.0.0</span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-sm text-slate-500 w-24">设备总数</span>
                <span className="text-sm text-slate-800 dark:text-slate-200 font-medium">{devices.length}</span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-sm text-slate-500 w-24">在线率</span>
                <div className="flex-1 h-2.5 bg-slate-200 dark:bg-white/10 rounded-full overflow-hidden">
                  <div
                    className="h-full progress-gradient-cyan rounded-full"
                    style={{ width: `${devices.length > 0 ? (devices.filter((d) => d.status === "online").length / devices.length) * 100 : 0}%` }}
                  />
                </div>
                <span className="text-sm text-slate-600 dark:text-slate-300 font-mono w-12 text-right">
                  {devices.length > 0 ? Math.round((devices.filter((d) => d.status === "online").length / devices.length) * 100) : 0}%
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
