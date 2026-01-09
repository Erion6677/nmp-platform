"use client";

import { useState, useEffect } from "react";
import Modal from "@/components/ui/Modal";
import { ArrowPathIcon, CheckIcon } from "@heroicons/react/24/outline";
import { deviceApi } from "@/lib/api/device";
import type { DeviceInterface } from "@/lib/api/types";

interface InterfaceModalProps {
  isOpen: boolean;
  onClose: () => void;
  deviceId: number;
  deviceName: string;
  onUpdate?: () => void;
}

export default function InterfaceModal({ isOpen, onClose, deviceId, deviceName, onUpdate }: InterfaceModalProps) {
  const [interfaces, setInterfaces] = useState<DeviceInterface[]>([]);
  const [syncing, setSyncing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [monitoredIds, setMonitoredIds] = useState<Set<number>>(new Set());

  useEffect(() => {
    if (isOpen) {
      fetchInterfaces();
    }
  }, [isOpen, deviceId]);

  const fetchInterfaces = async () => {
    setLoading(true);
    try {
      const res = await deviceApi.getDeviceInterfaces(deviceId);
      if (res.success) {
        setInterfaces(res.data);
        setMonitoredIds(new Set(res.data.filter(i => i.monitored).map(i => i.id)));
      }
    } catch (error) {
      console.error("Failed to fetch interfaces:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      const res = await deviceApi.syncInterfaces(deviceId);
      if (res.success) {
        setInterfaces(res.data.interfaces);
        setMonitoredIds(new Set(res.data.interfaces.filter(i => i.monitored).map(i => i.id)));
      }
    } catch (error) {
      console.error("Failed to sync interfaces:", error);
    } finally {
      setSyncing(false);
    }
  };

  const toggleMonitor = (id: number) => {
    setMonitoredIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const monitoredNames = interfaces
        .filter(i => monitoredIds.has(i.id))
        .map(i => i.name);
      await deviceApi.setMonitoredInterfaces(deviceId, monitoredNames);
      onUpdate?.();
      onClose();
    } catch (error) {
      console.error("Failed to save interfaces:", error);
    } finally {
      setSaving(false);
    }
  };

  const formatSpeed = (speed?: number) => {
    if (!speed) return "-";
    if (speed >= 1000000000) return `${(speed / 1000000000).toFixed(0)}Gbps`;
    if (speed >= 1000000) return `${(speed / 1000000).toFixed(0)}Mbps`;
    return `${speed}bps`;
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`接口管理 - ${deviceName}`} size="xl">
      <div className="space-y-4">
        {/* 操作栏 */}
        <div className="flex items-center justify-between">
          <p className="text-sm text-slate-500">
            选择需要监控的接口，监控的接口将采集带宽数据
          </p>
          <button
            onClick={handleSync}
            disabled={syncing}
            className="flex items-center gap-2 px-4 py-2 text-sm text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all disabled:opacity-50"
          >
            <ArrowPathIcon className={`w-4 h-4 ${syncing ? "animate-spin" : ""}`} />
            {syncing ? "同步中..." : "同步接口"}
          </button>
        </div>

        {/* 接口列表 */}
        <div className="inner-card rounded-xl overflow-hidden">
          {loading ? (
            <div className="p-8 text-center text-slate-500">加载中...</div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="bg-slate-100 dark:bg-[#0f1729]/50">
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">监控</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">接口名称</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">类型</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">速率</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">MAC 地址</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">状态</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-white/5">
                {interfaces.map((iface) => (
                  <tr key={iface.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02]">
                    <td className="px-4 py-3">
                      <button
                        onClick={() => toggleMonitor(iface.id)}
                        className={`w-6 h-6 rounded-lg border flex items-center justify-center transition-all ${
                          monitoredIds.has(iface.id)
                            ? "bg-gradient-to-r from-cyan-500 to-blue-600 border-cyan-500 text-white"
                            : "bg-slate-100 dark:bg-white/5 border-slate-300 dark:border-white/20"
                        }`}
                      >
                        {monitoredIds.has(iface.id) && <CheckIcon className="w-4 h-4" />}
                      </button>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm font-medium text-slate-800 dark:text-slate-200">{iface.name}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-slate-600 dark:text-slate-400">{iface.type || "-"}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-slate-600 dark:text-slate-400 font-mono">{formatSpeed(iface.speed)}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-slate-500 font-mono">{iface.mac_address || "-"}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`flex items-center gap-1.5 text-xs font-medium ${
                        iface.status === "up" ? "text-emerald-600 dark:text-emerald-400" : "text-slate-500"
                      }`}>
                        <span className={`w-1.5 h-1.5 rounded-full ${
                          iface.status === "up" ? "status-dot-online" : "bg-slate-400"
                        }`} />
                        {iface.status === "up" ? "在线" : "离线"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* 统计 */}
        <div className="flex items-center justify-between text-sm text-slate-500">
          <span>共 {interfaces.length} 个接口，已选择 {monitoredIds.size} 个监控</span>
        </div>

        {/* 按钮 */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-200 dark:border-white/10">
          <button
            onClick={onClose}
            className="px-4 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
          >
            取消
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-xl shadow-lg shadow-cyan-500/30 transition-all disabled:opacity-50"
          >
            {saving ? "保存中..." : "保存"}
          </button>
        </div>
      </div>
    </Modal>
  );
}
