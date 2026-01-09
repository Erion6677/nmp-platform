"use client";

import { useState, useEffect, useCallback } from "react";
import { useParams } from "next/navigation";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import InterfaceModal from "@/components/device/InterfaceModal";
import CollectorModal from "@/components/device/CollectorModal";
import PingTargetModal from "@/components/device/PingTargetModal";
import BandwidthChart from "@/components/charts/BandwidthChart";
import PingChart from "@/components/charts/PingChart";
import {
  ArrowLeftIcon,
  ArrowPathIcon,
  CpuChipIcon,
  CircleStackIcon,
  ClockIcon,
  InformationCircleIcon,
  CommandLineIcon,
  ArrowsRightLeftIcon,
  BoltIcon,
} from "@heroicons/react/24/outline";
import Link from "next/link";
import { deviceApi } from "@/lib/api/device";
import { metricsApi } from "@/lib/api/metrics";
import type { Device, SystemInfoResponse, DeviceInterface, BandwidthQueryResponse, PingQueryResponse, TimeRangeType } from "@/lib/api/types";

const timeRanges: { label: string; value: TimeRangeType }[] = [
  { label: "10分钟", value: "10m" },
  { label: "30分钟", value: "30m" },
  { label: "1小时", value: "1h" },
  { label: "3小时", value: "3h" },
  { label: "6小时", value: "6h" },
  { label: "24小时", value: "24h" },
];

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0) return `${days} 天 ${hours} 小时`;
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours} 小时 ${minutes} 分钟`;
}

function formatBytes(bytes: number): string {
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`;
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(0)} MB`;
  return `${(bytes / 1024).toFixed(0)} KB`;
}

function formatBps(bps: number): string {
  if (bps >= 1000000000) return `${(bps / 1000000000).toFixed(1)} Gbps`;
  if (bps >= 1000000) return `${(bps / 1000000).toFixed(1)} Mbps`;
  if (bps >= 1000) return `${(bps / 1000).toFixed(1)} Kbps`;
  return `${bps.toFixed(0)} bps`;
}

export default function DeviceDetailPage() {
  const params = useParams();
  const deviceId = Number(params.id);
  
  const [activeTimeRange, setActiveTimeRange] = useState<TimeRangeType>("1h");
  const [showInterfaceModal, setShowInterfaceModal] = useState(false);
  const [showCollectorModal, setShowCollectorModal] = useState(false);
  const [showPingTargetModal, setShowPingTargetModal] = useState(false);
  
  const [device, setDevice] = useState<Device | null>(null);
  const [systemInfo, setSystemInfo] = useState<SystemInfoResponse | null>(null);
  const [interfaces, setInterfaces] = useState<DeviceInterface[]>([]);
  const [bandwidthData, setBandwidthData] = useState<BandwidthQueryResponse | null>(null);
  const [pingData, setPingData] = useState<PingQueryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [deviceRes, systemRes, interfacesRes] = await Promise.all([
        deviceApi.getDevice(deviceId),
        deviceApi.getSystemInfo(deviceId).catch(() => null),
        deviceApi.getDeviceInterfaces(deviceId),
      ]);
      
      if (deviceRes.success) setDevice(deviceRes.data);
      if (systemRes?.success) setSystemInfo(systemRes.data);
      if (interfacesRes.success) setInterfaces(interfacesRes.data);
      
      // 获取带宽和 Ping 数据
      const monitoredInterfaces = interfacesRes.success 
        ? interfacesRes.data.filter(i => i.monitored).map(i => i.name)
        : [];
      
      if (monitoredInterfaces.length > 0) {
        const bwRes = await metricsApi.queryBandwidth(deviceId, activeTimeRange, monitoredInterfaces).catch(() => null);
        if (bwRes?.interfaces) setBandwidthData(bwRes);
      }
      
      const pingRes = await metricsApi.queryPing(deviceId, activeTimeRange).catch(() => null);
      if (pingRes?.targets) setPingData(pingRes);
    } catch (error) {
      console.error("Failed to fetch device data:", error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [deviceId, activeTimeRange]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleRefresh = () => {
    setRefreshing(true);
    fetchData();
  };

  const monitoredInterfaces = interfaces.filter(i => i.monitored);
  const deviceName = device?.name || "加载中...";

  if (loading) {
    return (
      <MainLayout>
        <Header title="设备详情" breadcrumb={["设备管理", "设备详情"]} />
        <div className="flex-1 flex items-center justify-center">
          <div className="text-slate-500">加载中...</div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout>
      <Header title="设备详情" breadcrumb={["设备管理", "设备详情"]} />
      <div className="flex-1 overflow-y-auto p-6">
        {/* 页面头部 */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Link
              href="/devices"
              className="flex items-center gap-2 px-4 py-2 text-sm text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
            >
              <ArrowLeftIcon className="w-4 h-4" />
              返回
            </Link>
            <h2 className="text-2xl font-bold text-slate-800 dark:text-white">设备详情</h2>
          </div>
          <button 
            onClick={handleRefresh}
            disabled={refreshing}
            className="flex items-center gap-2 px-4 py-2 text-sm text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all disabled:opacity-50"
          >
            <ArrowPathIcon className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
            刷新数据
          </button>
        </div>

        {/* 设备信息卡片 */}
        <div className="glass-card rounded-2xl p-6 mb-6">
          <div className="flex items-start justify-between mb-6">
            <div>
              <div className="flex items-center gap-3 mb-2">
                <h3 className="text-xl font-semibold text-slate-800 dark:text-white">{deviceName}</h3>
                <span className={`flex items-center gap-1.5 px-3 py-1 text-xs font-medium rounded-full ${
                  device?.status === "online"
                    ? "bg-emerald-500/10 dark:bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 dark:border-emerald-500/25"
                    : "bg-slate-500/10 dark:bg-slate-500/15 text-slate-600 dark:text-slate-400 border border-slate-500/20 dark:border-slate-500/25"
                }`}>
                  <span className={`w-1.5 h-1.5 rounded-full ${device?.status === "online" ? "status-dot-online" : "bg-slate-400"}`} />
                  {device?.status === "online" ? "在线" : "离线"}
                </span>
              </div>
              <div className="text-sm text-slate-500 dark:text-slate-400 font-mono">{device?.host}</div>
            </div>

            <div className="flex items-center gap-3">
              <button
                onClick={() => setShowInterfaceModal(true)}
                className="flex items-center gap-2 px-4 py-2 text-sm text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
              >
                <CommandLineIcon className="w-4 h-4" />
                接口管理
              </button>
              <button
                onClick={() => setShowCollectorModal(true)}
                className="flex items-center gap-2 px-4 py-2 text-sm text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
              >
                <ArrowsRightLeftIcon className="w-4 h-4" />
                采集器
              </button>
              <button
                onClick={() => setShowPingTargetModal(true)}
                className="flex items-center gap-2 px-4 py-2 text-sm text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
              >
                <BoltIcon className="w-4 h-4" />
                Ping 目标
              </button>
            </div>
          </div>

          {/* 系统指标 */}
          <div className="grid grid-cols-4 gap-4">
            {/* CPU */}
            <div className="inner-card rounded-xl p-4">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg icon-bg-cyan flex items-center justify-center">
                  <CpuChipIcon className="w-5 h-5 text-cyan-500 dark:text-cyan-400" />
                </div>
                <div className="text-sm text-slate-500 dark:text-slate-400">CPU 使用率</div>
              </div>
              <div className="flex items-center gap-3">
                <div className="flex-1 h-2.5 bg-slate-200 dark:bg-white/10 rounded-full overflow-hidden">
                  <div className="h-full progress-gradient-cyan rounded-full" style={{ width: `${systemInfo?.cpu_usage || 0}%` }} />
                </div>
                <span className="text-lg font-semibold text-cyan-600 dark:text-cyan-400">{systemInfo?.cpu_usage?.toFixed(0) || 0}%</span>
              </div>
              <div className="text-xs text-slate-500 mt-2">{systemInfo?.cpu_count || 0} 核心</div>
            </div>

            {/* 内存 */}
            <div className="inner-card rounded-xl p-4">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg icon-bg-violet flex items-center justify-center">
                  <CircleStackIcon className="w-5 h-5 text-violet-500 dark:text-violet-400" />
                </div>
                <div className="text-sm text-slate-500 dark:text-slate-400">内存使用率</div>
              </div>
              <div className="flex items-center gap-3">
                <div className="flex-1 h-2.5 bg-slate-200 dark:bg-white/10 rounded-full overflow-hidden">
                  <div className="h-full progress-gradient-violet rounded-full" style={{ width: `${systemInfo?.memory_usage || 0}%` }} />
                </div>
                <span className="text-lg font-semibold text-violet-600 dark:text-violet-400">{systemInfo?.memory_usage?.toFixed(0) || 0}%</span>
              </div>
              <div className="text-xs text-slate-500 mt-2">
                {systemInfo ? `${formatBytes(systemInfo.memory_total - systemInfo.memory_free)} / ${formatBytes(systemInfo.memory_total)}` : "-"}
              </div>
            </div>

            {/* 运行时间 */}
            <div className="inner-card rounded-xl p-4">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg icon-bg-emerald flex items-center justify-center">
                  <ClockIcon className="w-5 h-5 text-emerald-500 dark:text-emerald-400" />
                </div>
                <div className="text-sm text-slate-500 dark:text-slate-400">运行时间</div>
              </div>
              <div className="text-xl font-semibold text-emerald-600 dark:text-emerald-400">
                {systemInfo?.uptime ? formatUptime(systemInfo.uptime) : "-"}
              </div>
            </div>

            {/* 系统版本 */}
            <div className="inner-card rounded-xl p-4">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg icon-bg-amber flex items-center justify-center">
                  <InformationCircleIcon className="w-5 h-5 text-amber-500 dark:text-amber-400" />
                </div>
                <div className="text-sm text-slate-500 dark:text-slate-400">系统版本</div>
              </div>
              <div className="text-xl font-semibold text-amber-600 dark:text-amber-400">{systemInfo?.version || device?.version || "-"}</div>
              <div className="text-xs text-slate-500 mt-2">授权: {systemInfo?.license || "-"}</div>
            </div>
          </div>
        </div>

        {/* 带宽监控区 */}
        <div className="glass-card rounded-2xl overflow-hidden mb-6">
          <div className="px-6 py-4 border-b border-slate-200 dark:border-white/10 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-cyan-400" />
              <span className="font-medium text-slate-800 dark:text-white">接口带宽监控</span>
              <span className="px-2 py-0.5 text-xs text-slate-500 dark:text-slate-400 bg-slate-100 dark:bg-[#0f1729]/50 rounded-full border border-slate-200 dark:border-white/10">
                {monitoredInterfaces.length} 个监控接口
              </span>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex items-center bg-slate-100 dark:bg-[#0f1729]/50 rounded-xl p-1 border border-slate-200 dark:border-white/10">
                {timeRanges.map((range) => (
                  <button
                    key={range.value}
                    onClick={() => setActiveTimeRange(range.value)}
                    className={`px-3 py-1.5 text-xs rounded-lg transition-all ${
                      activeTimeRange === range.value
                        ? "bg-gradient-to-r from-cyan-500/20 to-blue-500/20 text-cyan-600 dark:text-cyan-400 border border-cyan-500/20"
                        : "text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
                    }`}
                  >
                    {range.label}
                  </button>
                ))}
              </div>
              <button 
                onClick={handleRefresh}
                disabled={refreshing}
                className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
              >
                <ArrowPathIcon className={`w-4 h-4 text-slate-500 dark:text-slate-400 ${refreshing ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>
          <div className="p-6 grid grid-cols-2 gap-6">
            {monitoredInterfaces.length > 0 ? monitoredInterfaces.map((iface) => {
              const ifaceData = bandwidthData?.interfaces?.[iface.name];
              const lastPoint = ifaceData?.[ifaceData.length - 1];
              return (
                <div key={iface.name} className="inner-card rounded-xl p-4">
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-2">
                      <CommandLineIcon className="w-4 h-4 text-cyan-500 dark:text-cyan-400" />
                      <span className="text-sm font-medium text-slate-700 dark:text-slate-200">{iface.name}</span>
                      <span className={`flex items-center gap-1 px-2 py-0.5 text-xs rounded-full ${
                        iface.status === "up"
                          ? "bg-emerald-500/10 dark:bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 dark:border-emerald-500/25"
                          : "bg-slate-500/10 text-slate-500 border border-slate-500/20"
                      }`}>
                        <span className={`w-1 h-1 rounded-full ${iface.status === "up" ? "status-dot-online" : "bg-slate-400"}`} />
                        {iface.status === "up" ? "在线" : "离线"}
                      </span>
                    </div>
                    <div className="flex items-center gap-4 text-xs">
                      <span className="text-slate-500">↓ <span className="text-cyan-600 dark:text-cyan-400 font-medium">{lastPoint ? formatBps(lastPoint.rx_rate) : "-"}</span></span>
                      <span className="text-slate-500">↑ <span className="text-violet-600 dark:text-violet-400 font-medium">{lastPoint ? formatBps(lastPoint.tx_rate) : "-"}</span></span>
                    </div>
                  </div>
                  <div className="flex items-center gap-4 text-xs mb-3">
                    <span className="flex items-center gap-1.5">
                      <span className="w-2 h-2 rounded-full bg-gradient-to-r from-cyan-400 to-blue-400" />
                      <span className="text-slate-500">接收</span>
                    </span>
                    <span className="flex items-center gap-1.5">
                      <span className="w-2 h-2 rounded-full bg-gradient-to-r from-violet-400 to-purple-400" />
                      <span className="text-slate-500">发送</span>
                    </span>
                  </div>
                  <div className="h-40">
                    <BandwidthChart data={ifaceData || []} height={160} />
                  </div>
                </div>
              );
            }) : (
              <div className="col-span-2 text-center py-8 text-slate-500">
                暂无监控接口，请在接口管理中选择需要监控的接口
              </div>
            )}
          </div>
        </div>

        {/* Ping 延迟监控区 */}
        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-200 dark:border-white/10 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-cyan-400" />
              <span className="font-medium text-slate-800 dark:text-white">Ping 延迟监控</span>
              <span className="px-2 py-0.5 text-xs text-slate-500 dark:text-slate-400 bg-slate-100 dark:bg-[#0f1729]/50 rounded-full border border-slate-200 dark:border-white/10">
                {pingData?.targets ? Object.keys(pingData.targets).length : 0} 个监控目标
              </span>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex items-center bg-slate-100 dark:bg-[#0f1729]/50 rounded-xl p-1 border border-slate-200 dark:border-white/10">
                {timeRanges.map((range) => (
                  <button
                    key={range.value}
                    onClick={() => setActiveTimeRange(range.value)}
                    className={`px-3 py-1.5 text-xs rounded-lg transition-all ${
                      activeTimeRange === range.value
                        ? "bg-gradient-to-r from-cyan-500/20 to-blue-500/20 text-cyan-600 dark:text-cyan-400 border border-cyan-500/20"
                        : "text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
                    }`}
                  >
                    {range.label}
                  </button>
                ))}
              </div>
              <button 
                onClick={handleRefresh}
                disabled={refreshing}
                className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
              >
                <ArrowPathIcon className={`w-4 h-4 text-slate-500 dark:text-slate-400 ${refreshing ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>
          <div className="p-6 grid grid-cols-2 gap-6">
            {pingData?.targets && Object.keys(pingData.targets).length > 0 ? (
              Object.entries(pingData.targets).map(([targetName, points]) => {
                const stats = pingData.stats?.[targetName];
                return (
                  <div key={targetName} className="inner-card rounded-xl p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <BoltIcon className="w-4 h-4 text-cyan-500 dark:text-cyan-400" />
                        <span className="text-sm font-medium text-slate-700 dark:text-slate-200">{targetName}</span>
                      </div>
                      <div className="flex items-center gap-3 text-xs">
                        <span className="flex items-center gap-1.5">
                          <span className="w-2 h-2 rounded-full bg-gradient-to-r from-emerald-400 to-green-400" />
                          <span className="text-slate-500">延迟</span>
                        </span>
                        <span className="flex items-center gap-1.5">
                          <span className="w-2 h-2 rounded-full bg-gradient-to-r from-rose-400 to-red-400" />
                          <span className="text-slate-500">丢包</span>
                        </span>
                      </div>
                    </div>
                    <div className="flex items-center gap-3 mb-4 text-xs flex-wrap">
                      <span className="px-2 py-1 bg-white dark:bg-white/5 rounded-lg border border-slate-200 dark:border-white/10">
                        <span className="text-slate-500">延迟</span>
                        <span className="text-emerald-600 dark:text-emerald-400 font-medium ml-1">{stats?.avg_latency ? (stats.avg_latency / 1000).toFixed(2) : "-"} ms</span>
                      </span>
                      <span className="px-2 py-1 bg-white dark:bg-white/5 rounded-lg border border-slate-200 dark:border-white/10">
                        <span className="text-slate-500">丢包率</span>
                        <span className={`font-medium ml-1 ${(stats?.loss_rate || 0) === 0 ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600 dark:text-amber-400"}`}>
                          {stats?.loss_rate?.toFixed(1) || "0"}%
                        </span>
                      </span>
                      <span className="px-2 py-1 bg-white dark:bg-white/5 rounded-lg border border-slate-200 dark:border-white/10">
                        <span className="text-slate-500">丢包数</span>
                        <span className={`font-medium ml-1 ${(stats?.loss_count || 0) === 0 ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}`}>
                          {stats?.loss_count || 0}/{stats?.total_count || 0}
                        </span>
                      </span>
                      <span className="px-2 py-1 bg-white dark:bg-white/5 rounded-lg border border-slate-200 dark:border-white/10">
                        <span className="text-slate-500">最小</span>
                        <span className="text-slate-700 dark:text-slate-300 font-medium ml-1">{stats?.min_latency ? (stats.min_latency / 1000).toFixed(2) : "-"} ms</span>
                      </span>
                      <span className="px-2 py-1 bg-white dark:bg-white/5 rounded-lg border border-slate-200 dark:border-white/10">
                        <span className="text-slate-500">最大</span>
                        <span className="text-slate-700 dark:text-slate-300 font-medium ml-1">{stats?.max_latency ? (stats.max_latency / 1000).toFixed(2) : "-"} ms</span>
                      </span>
                    </div>
                    <div className="h-40">
                      <PingChart data={points || []} height={160} />
                    </div>
                  </div>
                );
              })
            ) : (
              <div className="col-span-2 text-center py-8 text-slate-500">
                暂无 Ping 监控目标，请在 Ping 目标管理中添加
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 接口管理弹窗 */}
      <InterfaceModal
        isOpen={showInterfaceModal}
        onClose={() => setShowInterfaceModal(false)}
        deviceId={deviceId}
        deviceName={deviceName}
        onUpdate={fetchData}
      />

      {/* 采集器管理弹窗 */}
      <CollectorModal
        isOpen={showCollectorModal}
        onClose={() => setShowCollectorModal(false)}
        deviceId={deviceId}
        deviceName={deviceName}
      />

      {/* Ping 目标管理弹窗 */}
      <PingTargetModal
        isOpen={showPingTargetModal}
        onClose={() => setShowPingTargetModal(false)}
        deviceId={deviceId}
        deviceName={deviceName}
        interfaces={interfaces}
        onUpdate={fetchData}
      />
    </MainLayout>
  );
}
