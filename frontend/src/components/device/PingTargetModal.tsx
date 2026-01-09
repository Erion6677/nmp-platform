"use client";

import { useState, useEffect } from "react";
import Modal from "@/components/ui/Modal";
import { PlusIcon, TrashIcon, PencilSquareIcon, CheckIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { deviceApi } from "@/lib/api/device";
import type { PingTarget, DeviceInterface } from "@/lib/api/types";

interface PingTargetModalProps {
  isOpen: boolean;
  onClose: () => void;
  deviceId: number;
  deviceName: string;
  interfaces: DeviceInterface[];
  onUpdate?: () => void;
}

export default function PingTargetModal({ isOpen, onClose, deviceId, deviceName, interfaces, onUpdate }: PingTargetModalProps) {
  const [targets, setTargets] = useState<PingTarget[]>([]);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editForm, setEditForm] = useState({ target_name: "", target_address: "", source_interface: "" });
  const [showAddForm, setShowAddForm] = useState(false);
  const [newTarget, setNewTarget] = useState({ target_name: "", target_address: "", source_interface: "" });
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);

  const interfaceNames = interfaces.map(i => i.name);

  useEffect(() => {
    if (isOpen) {
      fetchTargets();
    }
  }, [isOpen, deviceId]);

  const fetchTargets = async () => {
    setLoading(true);
    try {
      const res = await deviceApi.getPingTargets(deviceId);
      if (res.success) {
        setTargets(res.data);
      }
    } catch (error) {
      console.error("Failed to fetch ping targets:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleToggle = async (id: number) => {
    try {
      const res = await deviceApi.togglePingTarget(deviceId, id);
      if (res.success) {
        setTargets(prev => prev.map(t => t.id === id ? { ...t, enabled: res.data.enabled } : t));
      }
    } catch (error) {
      console.error("Failed to toggle ping target:", error);
    }
  };

  const handleDelete = async (id: number) => {
    try {
      const res = await deviceApi.deletePingTarget(deviceId, id);
      if (res.success) {
        setTargets(prev => prev.filter(t => t.id !== id));
        onUpdate?.();
      }
    } catch (error) {
      console.error("Failed to delete ping target:", error);
    }
  };

  const handleEdit = (target: PingTarget) => {
    setEditingId(target.id);
    setEditForm({
      target_name: target.target_name,
      target_address: target.target_address,
      source_interface: target.source_interface,
    });
  };

  const handleSaveEdit = async () => {
    if (!editingId) return;
    try {
      const res = await deviceApi.updatePingTarget(deviceId, editingId, editForm);
      if (res.success) {
        setTargets(prev => prev.map(t => t.id === editingId ? res.data : t));
        setEditingId(null);
        onUpdate?.();
      }
    } catch (error) {
      console.error("Failed to update ping target:", error);
    }
  };

  const handleCancelEdit = () => {
    setEditingId(null);
  };

  const handleAdd = async () => {
    if (!newTarget.target_name || !newTarget.target_address) return;
    setSaving(true);
    try {
      const res = await deviceApi.createPingTarget(deviceId, newTarget);
      if (res.success) {
        setTargets(prev => [...prev, res.data]);
        setNewTarget({ target_name: "", target_address: "", source_interface: "" });
        setShowAddForm(false);
        onUpdate?.();
      }
    } catch (error) {
      console.error("Failed to create ping target:", error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Ping 目标管理 - ${deviceName}`} size="lg">
      <div className="space-y-4">
        {/* 操作栏 */}
        <div className="flex items-center justify-between">
          <p className="text-sm text-slate-500">
            配置 Ping 监控目标，用于检测网络连通性和延迟
          </p>
          <button
            onClick={() => setShowAddForm(true)}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-xl shadow-lg shadow-cyan-500/30 transition-all"
          >
            <PlusIcon className="w-4 h-4" />
            添加目标
          </button>
        </div>

        {/* 添加表单 */}
        {showAddForm && (
          <div className="inner-card rounded-xl p-4 space-y-4">
            <div className="grid grid-cols-3 gap-4">
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">名称</label>
                <input
                  type="text"
                  value={newTarget.target_name}
                  onChange={(e) => setNewTarget(prev => ({ ...prev, target_name: e.target.value }))}
                  placeholder="例如：电信 DNS"
                  className="w-full px-3 py-2 text-sm bg-white dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-lg text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">目标地址</label>
                <input
                  type="text"
                  value={newTarget.target_address}
                  onChange={(e) => setNewTarget(prev => ({ ...prev, target_address: e.target.value }))}
                  placeholder="IP 地址或域名"
                  className="w-full px-3 py-2 text-sm bg-white dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-lg text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 font-mono"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">源接口</label>
                <select
                  value={newTarget.source_interface}
                  onChange={(e) => setNewTarget(prev => ({ ...prev, source_interface: e.target.value }))}
                  className="w-full px-3 py-2 text-sm bg-white dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-lg text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                >
                  <option value="">默认</option>
                  {interfaceNames.map(iface => (
                    <option key={iface} value={iface}>{iface}</option>
                  ))}
                </select>
              </div>
            </div>
            <div className="flex items-center justify-end gap-2">
              <button
                onClick={() => setShowAddForm(false)}
                className="px-3 py-1.5 text-sm text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200"
              >
                取消
              </button>
              <button
                onClick={handleAdd}
                disabled={!newTarget.target_name || !newTarget.target_address || saving}
                className="px-4 py-1.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 rounded-lg disabled:opacity-50"
              >
                {saving ? "添加中..." : "添加"}
              </button>
            </div>
          </div>
        )}

        {/* 目标列表 */}
        <div className="inner-card rounded-xl overflow-hidden">
          {loading ? (
            <div className="p-8 text-center text-slate-500">加载中...</div>
          ) : targets.length === 0 ? (
            <div className="p-8 text-center text-slate-500">暂无 Ping 目标</div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="bg-slate-100 dark:bg-[#0f1729]/50">
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">启用</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">名称</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">目标地址</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">源接口</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-white/5">
                {targets.map(target => (
                  <tr key={target.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02]">
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleToggle(target.id)}
                        className={`w-10 h-5 rounded-full transition-all relative ${
                          target.enabled
                            ? "bg-gradient-to-r from-cyan-500 to-blue-600"
                            : "bg-slate-300 dark:bg-white/20"
                        }`}
                      >
                        <span
                          className={`absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all ${
                            target.enabled ? "left-5" : "left-0.5"
                          }`}
                        />
                      </button>
                    </td>
                    <td className="px-4 py-3">
                      {editingId === target.id ? (
                        <input
                          type="text"
                          value={editForm.target_name}
                          onChange={(e) => setEditForm(prev => ({ ...prev, target_name: e.target.value }))}
                          className="w-full px-2 py-1 text-sm bg-white dark:bg-[#0f1729]/50 border border-cyan-500/50 rounded text-slate-800 dark:text-slate-200 focus:outline-none"
                        />
                      ) : (
                        <span className="text-sm font-medium text-slate-800 dark:text-slate-200">{target.target_name}</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {editingId === target.id ? (
                        <input
                          type="text"
                          value={editForm.target_address}
                          onChange={(e) => setEditForm(prev => ({ ...prev, target_address: e.target.value }))}
                          className="w-full px-2 py-1 text-sm bg-white dark:bg-[#0f1729]/50 border border-cyan-500/50 rounded text-slate-800 dark:text-slate-200 focus:outline-none font-mono"
                        />
                      ) : (
                        <span className="text-sm text-slate-600 dark:text-slate-400 font-mono">{target.target_address}</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {editingId === target.id ? (
                        <select
                          value={editForm.source_interface}
                          onChange={(e) => setEditForm(prev => ({ ...prev, source_interface: e.target.value }))}
                          className="w-full px-2 py-1 text-sm bg-white dark:bg-[#0f1729]/50 border border-cyan-500/50 rounded text-slate-800 dark:text-slate-200 focus:outline-none"
                        >
                          <option value="">默认</option>
                          {interfaceNames.map(iface => (
                            <option key={iface} value={iface}>{iface}</option>
                          ))}
                        </select>
                      ) : (
                        <span className="text-sm text-slate-500">{target.source_interface || "默认"}</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        {editingId === target.id ? (
                          <>
                            <button
                              onClick={handleSaveEdit}
                              className="w-7 h-7 rounded-lg bg-emerald-100 dark:bg-emerald-500/10 hover:bg-emerald-200 dark:hover:bg-emerald-500/20 flex items-center justify-center transition-all"
                            >
                              <CheckIcon className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                            </button>
                            <button
                              onClick={handleCancelEdit}
                              className="w-7 h-7 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 flex items-center justify-center transition-all"
                            >
                              <XMarkIcon className="w-4 h-4 text-slate-500" />
                            </button>
                          </>
                        ) : (
                          <>
                            <button
                              onClick={() => handleEdit(target)}
                              className="w-7 h-7 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 flex items-center justify-center transition-all"
                            >
                              <PencilSquareIcon className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                            </button>
                            <button
                              onClick={() => handleDelete(target.id)}
                              className="w-7 h-7 rounded-lg bg-rose-100 dark:bg-rose-500/10 hover:bg-rose-200 dark:hover:bg-rose-500/20 flex items-center justify-center transition-all"
                            >
                              <TrashIcon className="w-4 h-4 text-rose-500 dark:text-rose-400" />
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* 按钮 */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-200 dark:border-white/10">
          <button
            onClick={onClose}
            className="px-4 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
          >
            关闭
          </button>
        </div>
      </div>
    </Modal>
  );
}
