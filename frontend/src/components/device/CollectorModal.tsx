"use client";

import { useState, useEffect, useCallback } from "react";
import Modal from "@/components/ui/Modal";
import { 
  CloudArrowUpIcon, 
  TrashIcon, 
  PauseIcon, 
  PlayIcon,
  ExclamationTriangleIcon
} from "@heroicons/react/24/outline";
import { deviceApi } from "@/lib/api/device";

interface CollectorModalProps {
  isOpen: boolean;
  onClose: () => void;
  deviceId: number;
  deviceName: string;
}

interface CollectorConfig {
  device_id: number;
  interval_ms: number;
  push_batch_size: number;
  enabled: boolean;
  status: string;
  deployed_at: string | null;
  last_push_at: string | null;
  push_count: number;
}

interface DeviceStatus {
  script_exists: boolean;
  scheduler_exists: boolean;
  scheduler_enabled: boolean;
}

export default function CollectorModal({ isOpen, onClose, deviceId }: CollectorModalProps) {
  const [config, setConfig] = useState<CollectorConfig | null>(null);
  const [deviceStatus, setDeviceStatus] = useState<DeviceStatus | null>(null);
  const [intervalSeconds, setIntervalSeconds] = useState(1);
  const [batchSize, setBatchSize] = useState(30);
  const [loading, setLoading] = useState(true);
  const [deploying, setDeploying] = useState(false);
  const [removing, setRemoving] = useState(false);
  const [toggling, setToggling] = useState(false);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const fetchStatus = useCallback(async () => {
    try {
      const res = await deviceApi.getCollectorStatus(deviceId);
      if (res.success && res.data) {
        setConfig(res.data.config);
        setDeviceStatus(res.data.device_status);
        if (res.data.config) {
          setIntervalSeconds(Math.max(1, Math.floor((res.data.config.interval_ms || 1000) / 1000)));
          setBatchSize(res.data.config.push_batch_size || 30);
        }
      }
    } catch (error) {
      console.error("Failed to fetch collector status:", error);
    }
  }, [deviceId]);

  useEffect(() => {
    if (isOpen) {
      setLoading(true);
      fetchStatus().finally(() => setLoading(false));
    }
  }, [isOpen, deviceId, fetchStatus]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchStatus();
    setRefreshing(false);
  };

  const handleDeploy = async () => {
    setDeploying(true);
    try {
      const res = await deviceApi.deployCollector(deviceId, intervalSeconds * 1000);
      if (res.success) { await fetchStatus(); }
    } catch (error) { console.error("Failed to deploy:", error); }
    finally { setDeploying(false); }
  };

  const handleRemove = async () => {
    setRemoving(true);
    try {
      const res = await deviceApi.undeployCollector(deviceId);
      if (res.success) { await fetchStatus(); }
    } catch (error) { console.error("Failed to remove:", error); }
    finally { setRemoving(false); }
  };

  const handleTogglePush = async () => {
    setToggling(true);
    try {
      const newEnabled = !(deviceStatus?.scheduler_enabled ?? false);
      const res = await deviceApi.toggleCollectorPush(deviceId, newEnabled);
      if (res.success) { await fetchStatus(); }
    } catch (error) { console.error("Failed to toggle:", error); }
    finally { setToggling(false); }
  };

  const handleSaveConfig = async () => {
    setSaving(true);
    try {
      const res = await deviceApi.updateCollectorConfig(deviceId, intervalSeconds * 1000, batchSize);
      if (res.success) { await fetchStatus(); }
    } catch (error) { console.error("Failed to save:", error); }
    finally { setSaving(false); }
  };

  const handleClearData = async () => {
    if (!confirm("确定要清除该设备的所有采集数据吗？此操作不可恢复！")) return;
    setClearing(true);
    try {
      const res = await deviceApi.clearDeviceData(deviceId);
      if (res.success) { alert("数据清除成功"); await fetchStatus(); }
    } catch (error) { console.error("Failed to clear:", error); alert("数据清除失败"); }
    finally { setClearing(false); }
  };

  const formatLastPush = (lastPush: string | null) => {
    if (!lastPush) return "从未";
    const diffSeconds = Math.floor((Date.now() - new Date(lastPush).getTime()) / 1000);
    if (diffSeconds < 60) return "刚刚";
    if (diffSeconds < 3600) return `${Math.floor(diffSeconds / 60)} 分钟前`;
    if (diffSeconds < 86400) return `${Math.floor(diffSeconds / 3600)} 小时前`;
    return `${Math.floor(diffSeconds / 86400)} 天前`;
  };

  const isDeployed = deviceStatus?.script_exists ?? false;
  const isPushEnabled = deviceStatus?.scheduler_enabled ?? false;
  const pushCount = config?.push_count ?? 0;
  const lastPush = config?.last_push_at ?? null;
  const pushInterval = intervalSeconds * batchSize;

  if (loading) {
    return (
      <Modal isOpen={isOpen} onClose={onClose} title="采集器管理" size="lg">
        <div className="p-8 text-center text-slate-500">加载中...</div>
      </Modal>
    );
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="采集器管理" size="lg">
      <div className="space-y-6">
        {/* 采集器状态 */}
        <div className="glass-card rounded-xl p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-slate-300">采集器状态</h3>
            <button onClick={handleRefresh} disabled={refreshing}
              className="px-3 py-1.5 text-xs font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-lg transition-all disabled:opacity-50">
              {refreshing ? "刷新中..." : "刷新"}
            </button>
          </div>
          <div className="grid grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-xs text-slate-500 mb-2">部署状态</div>
              <div className={`inline-flex px-3 py-1 rounded-full text-xs font-medium ${isDeployed ? "bg-emerald-500/15 text-emerald-400 border border-emerald-500/25" : "bg-slate-500/15 text-slate-400 border border-slate-500/25"}`}>
                {isDeployed ? "运行中" : "未部署"}
              </div>
            </div>
            <div className="text-center">
              <div className="text-xs text-slate-500 mb-2">推送状态</div>
              <div className={`inline-flex px-3 py-1 rounded-full text-xs font-medium ${isPushEnabled ? "bg-cyan-500/15 text-cyan-400 border border-cyan-500/25" : "bg-amber-500/15 text-amber-400 border border-amber-500/25"}`}>
                {isPushEnabled ? "已开启" : "已关闭"}
              </div>
            </div>
            <div className="text-center">
              <div className="text-xs text-slate-500 mb-2">推送次数</div>
              <div className="text-lg font-semibold text-white">{pushCount}</div>
            </div>
            <div className="text-center">
              <div className="text-xs text-slate-500 mb-2">最后推送</div>
              <div className="text-lg font-semibold text-white">{formatLastPush(lastPush)}</div>
            </div>
          </div>
        </div>

        {/* 采集配置 */}
        <div className="glass-card rounded-xl p-5">
          <h3 className="text-sm font-medium text-slate-300 mb-4">采集配置</h3>
          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <label className="w-20 text-sm text-slate-400">采集间隔</label>
              <div className="flex items-center gap-2">
                <input type="number" min={1} max={60} value={intervalSeconds}
                  onChange={(e) => setIntervalSeconds(Math.max(1, Math.min(60, parseInt(e.target.value) || 1)))}
                  className="w-24 px-3 py-2 text-sm bg-[#0f1729]/50 border border-white/10 rounded-lg text-white text-center focus:outline-none focus:ring-2 focus:ring-cyan-500/50" />
                <span className="text-sm text-slate-400">秒</span>
                <div className="flex items-center gap-1 ml-2">
                  <button onClick={() => setIntervalSeconds(Math.max(1, intervalSeconds - 1))}
                    className="w-7 h-7 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg text-slate-400">−</button>
                  <button onClick={() => setIntervalSeconds(Math.min(60, intervalSeconds + 1))}
                    className="w-7 h-7 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg text-slate-400">+</button>
                </div>
              </div>
              <span className="text-xs text-slate-500">每隔多少秒采集一次数据</span>
            </div>
            <div className="flex items-center gap-4">
              <label className="w-20 text-sm text-slate-400">采集周期</label>
              <div className="flex items-center gap-2">
                <input type="number" min={1} max={100} value={batchSize}
                  onChange={(e) => setBatchSize(Math.max(1, Math.min(100, parseInt(e.target.value) || 1)))}
                  className="w-24 px-3 py-2 text-sm bg-[#0f1729]/50 border border-white/10 rounded-lg text-white text-center focus:outline-none focus:ring-2 focus:ring-cyan-500/50" />
                <span className="text-sm text-slate-400">次</span>
                <div className="flex items-center gap-1 ml-2">
                  <button onClick={() => setBatchSize(Math.max(1, batchSize - 1))}
                    className="w-7 h-7 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg text-slate-400">−</button>
                  <button onClick={() => setBatchSize(Math.min(100, batchSize + 1))}
                    className="w-7 h-7 flex items-center justify-center bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg text-slate-400">+</button>
                </div>
              </div>
              <span className="text-xs text-slate-500">采集多少次后推送一次</span>
            </div>
            <div className="flex items-center gap-4">
              <label className="w-20 text-sm text-slate-400">推送间隔</label>
              <div className="px-3 py-2 bg-[#0f1729]/30 border border-white/5 rounded-lg">
                <span className="text-cyan-400 font-medium">{pushInterval} 秒</span>
              </div>
              <span className="text-xs text-slate-500">= 采集间隔 × 采集周期</span>
            </div>
          </div>
          <div className="mt-4 pt-4 border-t border-white/5">
            <button onClick={handleSaveConfig} disabled={saving || !isDeployed}
              className="px-4 py-2 text-sm font-medium text-cyan-400 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/25 rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed">
              {saving ? "保存中..." : "保存配置"}
            </button>
          </div>
        </div>

        {/* 采集器操作 */}
        <div className="glass-card rounded-xl p-5">
          <h3 className="text-sm font-medium text-slate-300 mb-4">采集器操作</h3>
          <div className="flex items-center gap-3">
            <button onClick={handleDeploy} disabled={deploying || isDeployed}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-lg shadow-lg shadow-cyan-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed">
              <CloudArrowUpIcon className="w-4 h-4" />
              {deploying ? "部署中..." : "部署脚本"}
            </button>
            <button onClick={handleRemove} disabled={removing || !isDeployed}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/25 rounded-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed">
              <TrashIcon className="w-4 h-4" />
              {removing ? "移除中..." : "移除脚本"}
            </button>
            <button onClick={handleTogglePush} disabled={toggling || !isDeployed}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border transition-all disabled:opacity-50 disabled:cursor-not-allowed ${isPushEnabled ? "text-slate-300 bg-white/5 hover:bg-white/10 border-white/10" : "text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 border-emerald-500/25"}`}>
              {isPushEnabled ? <><PauseIcon className="w-4 h-4" />{toggling ? "处理中..." : "关闭推送"}</> : <><PlayIcon className="w-4 h-4" />{toggling ? "处理中..." : "开启推送"}</>}
            </button>
          </div>
        </div>

        {/* 危险操作 */}
        <div className="glass-card rounded-xl p-5 border border-rose-500/20">
          <h3 className="text-sm font-medium text-rose-400 mb-2">危险操作</h3>
          <p className="text-xs text-slate-500 mb-4">清除数据将删除该设备的所有采集数据，此操作不可恢复</p>
          <button onClick={handleClearData} disabled={clearing}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/25 rounded-lg transition-all disabled:opacity-50">
            <ExclamationTriangleIcon className="w-4 h-4" />
            {clearing ? "清除中..." : "清除数据"}
          </button>
        </div>

        {/* 底部按钮 */}
        <div className="flex justify-end pt-4 border-t border-white/10">
          <button onClick={onClose} className="px-4 py-2 text-sm font-medium text-slate-300 bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg transition-all">关闭</button>
        </div>
      </div>
    </Modal>
  );
}
