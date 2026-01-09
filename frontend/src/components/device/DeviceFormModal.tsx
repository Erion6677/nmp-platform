"use client";

import { useState, useEffect } from "react";
import Modal from "@/components/ui/Modal";

interface Device {
  id?: number;
  name: string;
  type: string;
  host: string;
  port: number;
  apiPort?: number;
  username: string;
  password?: string;
  description?: string;
}

interface DeviceFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  device?: Device | null;
  onSave: (device: Device) => void;
}

const deviceTypes = [
  { value: "mikrotik", label: "MikroTik" },
  { value: "linux", label: "Linux" },
  { value: "switch", label: "交换机" },
  { value: "firewall", label: "防火墙" },
];

export default function DeviceFormModal({ isOpen, onClose, device, onSave }: DeviceFormModalProps) {
  const [formData, setFormData] = useState<Device>({
    name: "",
    type: "mikrotik",
    host: "",
    port: 22,
    apiPort: 8728,
    username: "",
    password: "",
    description: "",
  });
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  useEffect(() => {
    if (device) {
      setFormData({
        ...device,
        password: "",
      });
    } else {
      setFormData({
        name: "",
        type: "mikrotik",
        host: "",
        port: 22,
        apiPort: 8728,
        username: "",
        password: "",
        description: "",
      });
    }
    setTestResult(null);
  }, [device, isOpen]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: name === "port" || name === "apiPort" ? parseInt(value) || 0 : value,
    }));
  };

  const handleTestConnection = async () => {
    setTesting(true);
    setTestResult(null);
    // 模拟测试连接
    await new Promise((resolve) => setTimeout(resolve, 1500));
    setTestResult({
      success: Math.random() > 0.3,
      message: Math.random() > 0.3 ? "连接成功，延迟 12ms" : "连接失败：认证错误",
    });
    setTesting(false);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(formData);
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={device ? "编辑设备" : "添加设备"}
      size="lg"
    >
      <form onSubmit={handleSubmit} className="space-y-5">
        <div className="grid grid-cols-2 gap-5">
          {/* 设备名称 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              设备名称 <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              name="name"
              value={formData.name}
              onChange={handleChange}
              required
              placeholder="例如：核心路由器-01"
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all"
            />
          </div>

          {/* 设备类型 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              设备类型 <span className="text-rose-500">*</span>
            </label>
            <select
              name="type"
              value={formData.type}
              onChange={handleChange}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all"
            >
              {deviceTypes.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </div>

          {/* 主机地址 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              主机地址 <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              name="host"
              value={formData.host}
              onChange={handleChange}
              required
              placeholder="IP 地址或域名"
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all font-mono"
            />
          </div>

          {/* SSH 端口 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              SSH 端口 <span className="text-rose-500">*</span>
            </label>
            <input
              type="number"
              name="port"
              value={formData.port}
              onChange={handleChange}
              required
              min={1}
              max={65535}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all font-mono"
            />
          </div>

          {/* API 端口 (MikroTik) */}
          {formData.type === "mikrotik" && (
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                API 端口
              </label>
              <input
                type="number"
                name="apiPort"
                value={formData.apiPort}
                onChange={handleChange}
                min={1}
                max={65535}
                className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all font-mono"
              />
            </div>
          )}

          {/* 用户名 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              用户名 <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              name="username"
              value={formData.username}
              onChange={handleChange}
              required
              placeholder="SSH 用户名"
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all"
            />
          </div>

          {/* 密码 */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              密码 {!device && <span className="text-rose-500">*</span>}
            </label>
            <input
              type="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              required={!device}
              placeholder={device ? "留空则不修改" : "SSH 密码"}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all"
            />
          </div>
        </div>

        {/* 描述 */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            描述
          </label>
          <textarea
            name="description"
            value={formData.description}
            onChange={handleChange}
            rows={3}
            placeholder="设备描述信息（可选）"
            className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all resize-none"
          />
        </div>

        {/* 测试结果 */}
        {testResult && (
          <div className={`p-3 rounded-xl text-sm ${
            testResult.success
              ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20"
              : "bg-rose-500/10 text-rose-600 dark:text-rose-400 border border-rose-500/20"
          }`}>
            {testResult.message}
          </div>
        )}

        {/* 按钮 */}
        <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-200 dark:border-white/10">
          <button
            type="button"
            onClick={handleTestConnection}
            disabled={testing || !formData.host || !formData.username}
            className="px-4 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {testing ? "测试中..." : "测试连接"}
          </button>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all"
          >
            取消
          </button>
          <button
            type="submit"
            className="px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-xl shadow-lg shadow-cyan-500/30 transition-all"
          >
            {device ? "保存" : "添加"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
