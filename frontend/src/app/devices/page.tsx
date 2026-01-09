"use client";

import { useState, useEffect, useCallback } from "react";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import DeviceFormModal from "@/components/device/DeviceFormModal";
import DeleteConfirmModal from "@/components/device/DeleteConfirmModal";
import { deviceApi } from "@/lib/api/device";
import type { Device, DeviceStatus } from "@/lib/api/types";
import {
  PlusIcon,
  MagnifyingGlassIcon,
  ArrowPathIcon,
  EyeIcon,
  PencilSquareIcon,
  TrashIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "@heroicons/react/24/outline";
import Link from "next/link";

const typeLabels: Record<string, { label: string; color: string }> = {
  mikrotik: { label: "MikroTik", color: "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20" },
  linux: { label: "Linux", color: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20" },
  switch: { label: "交换机", color: "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-500/20" },
  firewall: { label: "防火墙", color: "bg-rose-500/10 text-rose-600 dark:text-rose-400 border-rose-500/20" },
};

const statusConfig: Record<string, { label: string; dotClass: string; textClass: string }> = {
  online: { label: "在线", dotClass: "status-dot-online", textClass: "text-emerald-600 dark:text-emerald-400" },
  offline: { label: "离线", dotClass: "status-dot-offline", textClass: "text-rose-600 dark:text-rose-400" },
  warning: { label: "告警", dotClass: "status-dot-warning", textClass: "text-amber-600 dark:text-amber-400" },
  unknown: { label: "未知", dotClass: "bg-slate-400", textClass: "text-slate-600 dark:text-slate-400" },
  error: { label: "错误", dotClass: "status-dot-offline", textClass: "text-rose-600 dark:text-rose-400" },
};

export default function DevicesPage() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");

  // 加载设备列表
  const loadDevices = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string | number | undefined> = {
        page,
        page_size: pageSize,
      };
      if (searchQuery) params.search = searchQuery;
      if (typeFilter !== "all") params.type = typeFilter;
      if (statusFilter !== "all") params.status = statusFilter as DeviceStatus;

      const response = await deviceApi.listDevices(params);
      if (response.success && response.data) {
        setDevices(response.data.devices || []);
        setTotal(response.data.total || 0);
      }
    } catch (error) {
      console.error("加载设备列表失败:", error);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, searchQuery, typeFilter, statusFilter]);

  useEffect(() => {
    loadDevices();
  }, [loadDevices]);

  const handleAdd = () => {
    setSelectedDevice(null);
    setShowAddModal(true);
  };

  const handleEdit = (device: Device) => {
    setSelectedDevice(device);
    setShowEditModal(true);
  };

  const handleDelete = (device: Device) => {
    setSelectedDevice(device);
    setShowDeleteModal(true);
  };

  const handleSaveDevice = async (deviceData: any) => {
    try {
      if (selectedDevice) {
        // 编辑
        await deviceApi.updateDevice(selectedDevice.id, deviceData);
      } else {
        // 添加
        await deviceApi.createDevice(deviceData);
      }
      loadDevices();
    } catch (error) {
      console.error("保存设备失败:", error);
    }
  };

  const handleConfirmDelete = async () => {
    if (selectedDevice) {
      try {
        await deviceApi.deleteDevice(selectedDevice.id);
        loadDevices();
      } catch (error) {
        console.error("删除设备失败:", error);
      }
    }
  };

  // 统计
  const stats = {
    online: devices.filter((d) => d.status === "online").length,
    offline: devices.filter((d) => d.status === "offline").length,
    warning: devices.filter((d) => d.status === "error" || d.status === "unknown").length,
    total: total,
  };

  const totalPages = Math.ceil(total / pageSize);

  // 格式化最后在线时间
  const formatLastSeen = (lastSeen?: string) => {
    if (!lastSeen) return "-";
    const date = new Date(lastSeen);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    if (diff < 60000) return "刚刚";
    if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`;
    return `${Math.floor(diff / 86400000)} 天前`;
  };

  return (
    <MainLayout>
      <Header title="设备管理" breadcrumb={["设备管理"]} />
      <div className="flex-1 overflow-y-auto p-6">
        {/* 页面头部 */}
        <div className="flex items-start justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-slate-800 dark:text-white">设备管理</h2>
            <p className="text-sm text-slate-500 mt-1">管理网络设备，包括路由器、交换机、服务器等</p>
          </div>
          <button
            onClick={handleAdd}
            className="flex items-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-xl shadow-lg shadow-cyan-500/30 transition-all"
          >
            <PlusIcon className="w-5 h-5" />
            添加设备
          </button>
        </div>

        {/* 统计条 */}
        <div className="flex items-center gap-3 mb-5">
          <span className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
            <span className="w-2 h-2 rounded-full status-dot-online" />
            在线 {stats.online}
          </span>
          <span className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20">
            <span className="w-2 h-2 rounded-full status-dot-offline" />
            离线 {stats.offline}
          </span>
          <span className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-full bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20">
            <span className="w-2 h-2 rounded-full status-dot-warning" />
            异常 {stats.warning}
          </span>
          <span className="px-3 py-1.5 text-xs font-medium rounded-full bg-slate-100 dark:bg-white/5 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-white/10">
            总计 {stats.total}
          </span>
        </div>

        {/* 筛选栏 */}
        <div className="flex items-center justify-between gap-4 mb-5 p-4 glass-card rounded-2xl">
          <div className="relative flex-1 max-w-md">
            <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="搜索设备名称、IP地址..."
              className="w-full pl-10 pr-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all"
            />
          </div>
          <div className="flex items-center gap-3">
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-600 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            >
              <option value="all">所有类型</option>
              <option value="mikrotik">MikroTik</option>
              <option value="linux">Linux</option>
              <option value="switch">交换机</option>
              <option value="firewall">防火墙</option>
            </select>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-600 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            >
              <option value="all">所有状态</option>
              <option value="online">在线</option>
              <option value="offline">离线</option>
            </select>
            <button
              onClick={loadDevices}
              className="w-10 h-10 rounded-xl bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
            >
              <ArrowPathIcon className={`w-5 h-5 text-slate-500 dark:text-slate-400 ${loading ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        {/* 数据表格 */}
        <div className="glass-card rounded-2xl overflow-hidden">
          {loading ? (
            <div className="py-20 text-center">
              <div className="w-8 h-8 border-4 border-cyan-500/30 border-t-cyan-500 rounded-full animate-spin mx-auto mb-4" />
              <p className="text-slate-500">加载中...</p>
            </div>
          ) : (
            <>
              <table className="w-full">
                <thead>
                  <tr className="bg-slate-100 dark:bg-[#0f1729]/50">
                    <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">状态</th>
                    <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">设备名称</th>
                    <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">类型</th>
                    <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">主机地址</th>
                    <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">端口</th>
                    <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">最后在线</th>
                    <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-white/5">
                  {devices.map((device) => (
                    <tr key={device.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02] transition-colors">
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-2">
                          <span className={`w-2 h-2 rounded-full ${statusConfig[device.status]?.dotClass || "bg-slate-400"}`} />
                          <span className={`text-sm ${statusConfig[device.status]?.textClass || "text-slate-500"}`}>
                            {statusConfig[device.status]?.label || device.status}
                          </span>
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <Link href={`/devices/${device.id}`} className="text-sm font-medium text-cyan-600 dark:text-cyan-400 hover:underline">
                          {device.name}
                        </Link>
                      </td>
                      <td className="px-5 py-4">
                        <span className={`px-2 py-0.5 text-xs font-medium rounded-full border ${typeLabels[device.type]?.color || "bg-slate-100 text-slate-600"}`}>
                          {typeLabels[device.type]?.label || device.type}
                        </span>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-sm text-slate-600 dark:text-slate-300 font-mono">{device.host}</span>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-sm text-slate-500 font-mono">{device.port}</span>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-sm text-slate-500">{formatLastSeen(device.last_seen)}</span>
                      </td>
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-2">
                          <Link
                            href={`/devices/${device.id}`}
                            className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
                            title="查看"
                          >
                            <EyeIcon className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                          </Link>
                          <button
                            onClick={() => handleEdit(device)}
                            className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
                            title="编辑"
                          >
                            <PencilSquareIcon className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                          </button>
                          <button
                            onClick={() => handleDelete(device)}
                            className="w-8 h-8 rounded-lg bg-rose-100 dark:bg-rose-500/10 hover:bg-rose-200 dark:hover:bg-rose-500/20 border border-rose-200 dark:border-rose-500/20 flex items-center justify-center transition-all"
                            title="删除"
                          >
                            <TrashIcon className="w-4 h-4 text-rose-500 dark:text-rose-400" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {devices.length === 0 && (
                <div className="py-12 text-center text-slate-500">
                  没有找到设备
                </div>
              )}

              {/* 分页 */}
              <div className="px-5 py-4 border-t border-slate-200 dark:border-white/10 flex items-center justify-between">
                <span className="text-sm text-slate-500">共 {total} 条记录</span>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all disabled:opacity-50"
                  >
                    <ChevronLeftIcon className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                  </button>
                  {Array.from({ length: Math.min(5, totalPages) }, (_, i) => i + 1).map((p) => (
                    <button
                      key={p}
                      onClick={() => setPage(p)}
                      className={`w-8 h-8 rounded-lg flex items-center justify-center text-sm font-medium transition-all ${
                        page === p
                          ? "bg-gradient-to-r from-cyan-500/20 to-blue-500/20 text-cyan-600 dark:text-cyan-400 border border-cyan-500/20"
                          : "bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 text-slate-600 dark:text-slate-400"
                      }`}
                    >
                      {p}
                    </button>
                  ))}
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages || totalPages === 0}
                    className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all disabled:opacity-50"
                  >
                    <ChevronRightIcon className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* 添加设备弹窗 */}
      <DeviceFormModal
        isOpen={showAddModal}
        onClose={() => setShowAddModal(false)}
        onSave={handleSaveDevice}
      />

      {/* 编辑设备弹窗 */}
      <DeviceFormModal
        isOpen={showEditModal}
        onClose={() => setShowEditModal(false)}
        device={selectedDevice ? {
          id: selectedDevice.id,
          name: selectedDevice.name,
          type: selectedDevice.type,
          host: selectedDevice.host,
          port: selectedDevice.port,
          apiPort: selectedDevice.api_port,
          username: selectedDevice.username,
          description: selectedDevice.description,
        } : null}
        onSave={handleSaveDevice}
      />

      {/* 删除确认弹窗 */}
      <DeleteConfirmModal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleConfirmDelete}
        deviceName={selectedDevice?.name || ""}
      />
    </MainLayout>
  );
}
