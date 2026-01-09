"use client";

import { useState, useEffect } from "react";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import {
  Cog6ToothIcon,
  BellIcon,
  ClockIcon,
  ShieldCheckIcon,
  PaintBrushIcon,
} from "@heroicons/react/24/outline";
import { settingsApi } from "@/lib/api/settings";
import type { CollectionSettings } from "@/lib/api/types";

type TabKey = "general" | "collection" | "notification" | "security" | "appearance";

const tabs: { key: TabKey; label: string; icon: typeof Cog6ToothIcon }[] = [
  { key: "general", label: "基本设置", icon: Cog6ToothIcon },
  { key: "collection", label: "采集设置", icon: ClockIcon },
  { key: "notification", label: "通知设置", icon: BellIcon },
  { key: "security", label: "安全设置", icon: ShieldCheckIcon },
  { key: "appearance", label: "外观设置", icon: PaintBrushIcon },
];

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<TabKey>("general");
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);

  // 基本设置
  const [generalSettings, setGeneralSettings] = useState({
    siteName: "NMP 网络监控平台",
    siteDescription: "专业的网络设备监控管理系统",
    timezone: "Asia/Shanghai",
    language: "zh-CN",
  });

  // 采集设置
  const [collectionSettings, setCollectionSettings] = useState<CollectionSettings>({
    default_push_interval: 5000,
    data_retention_days: 30,
    frontend_refresh_interval: 10,
    device_offline_timeout: 60,
    follow_push_interval: true,
  });

  // 通知设置
  const [notificationSettings, setNotificationSettings] = useState({
    emailEnabled: false,
    emailServer: "",
    emailPort: 587,
    emailUsername: "",
    webhookEnabled: false,
    webhookUrl: "",
  });

  // 安全设置
  const [securitySettings, setSecuritySettings] = useState({
    sessionTimeout: 30,
    maxLoginAttempts: 5,
    passwordMinLength: 8,
    requireSpecialChar: true,
    twoFactorEnabled: false,
  });

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    setLoading(true);
    try {
      const res = await settingsApi.getCollectionSettings();
      if (res.success) {
        setCollectionSettings(res.data);
      }
    } catch (error) {
      console.error("Failed to fetch settings:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      if (activeTab === "collection") {
        await settingsApi.updateCollectionSettings(collectionSettings);
      }
      // 其他设置暂时只保存到本地状态
    } catch (error) {
      console.error("Failed to save settings:", error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <MainLayout>
      <Header title="系统设置" breadcrumb={["系统设置"]} />
      <div className="flex-1 overflow-y-auto p-6">
        <div className="flex gap-6">
          {/* 左侧标签栏 */}
          <div className="w-56 flex-shrink-0">
            <div className="glass-card rounded-2xl p-3 space-y-1">
              {tabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <button
                    key={tab.key}
                    onClick={() => setActiveTab(tab.key)}
                    className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all text-left ${
                      activeTab === tab.key
                        ? "bg-gradient-to-r from-cyan-500/10 to-blue-500/10 dark:from-cyan-500/20 dark:to-blue-500/20 text-cyan-600 dark:text-cyan-400 border border-cyan-500/20"
                        : "text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-white/5"
                    }`}
                  >
                    <Icon className="w-5 h-5" />
                    <span className="font-medium">{tab.label}</span>
                  </button>
                );
              })}
            </div>
          </div>

          {/* 右侧内容区 */}
          <div className="flex-1">
            <div className="glass-card rounded-2xl p-6">
              {loading ? (
                <div className="p-8 text-center text-slate-500">加载中...</div>
              ) : (
                <>
                  {/* 基本设置 */}
                  {activeTab === "general" && (
                    <div className="space-y-6">
                      <div>
                        <h3 className="text-lg font-semibold text-slate-800 dark:text-white mb-4">基本设置</h3>
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">站点名称</label>
                            <input
                              type="text"
                              value={generalSettings.siteName}
                              onChange={(e) => setGeneralSettings(prev => ({ ...prev, siteName: e.target.value }))}
                              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                            />
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">站点描述</label>
                            <textarea
                              value={generalSettings.siteDescription}
                              onChange={(e) => setGeneralSettings(prev => ({ ...prev, siteDescription: e.target.value }))}
                              rows={3}
                              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 resize-none"
                            />
                          </div>
                          <div className="grid grid-cols-2 gap-4">
                            <div>
                              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">时区</label>
                              <select
                                value={generalSettings.timezone}
                                onChange={(e) => setGeneralSettings(prev => ({ ...prev, timezone: e.target.value }))}
                                className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                              >
                                <option value="Asia/Shanghai">Asia/Shanghai (UTC+8)</option>
                                <option value="Asia/Tokyo">Asia/Tokyo (UTC+9)</option>
                                <option value="UTC">UTC</option>
                              </select>
                            </div>
                            <div>
                              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">语言</label>
                              <select
                                value={generalSettings.language}
                                onChange={(e) => setGeneralSettings(prev => ({ ...prev, language: e.target.value }))}
                                className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                              >
                                <option value="zh-CN">简体中文</option>
                                <option value="en-US">English</option>
                              </select>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* 采集设置 */}
                  {activeTab === "collection" && (
                    <div className="space-y-6">
                      <div>
                        <h3 className="text-lg font-semibold text-slate-800 dark:text-white mb-4">采集设置</h3>
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">默认采集间隔</label>
                            <select
                              value={collectionSettings.default_push_interval}
                              onChange={(e) => setCollectionSettings(prev => ({ ...prev, default_push_interval: Number(e.target.value) }))}
                              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                            >
                              <option value={1000}>1 秒</option>
                              <option value={2000}>2 秒</option>
                              <option value={5000}>5 秒</option>
                              <option value={10000}>10 秒</option>
                              <option value={30000}>30 秒</option>
                            </select>
                            <p className="mt-1 text-xs text-slate-500">新设备的默认数据采集间隔</p>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">前端刷新间隔（秒）</label>
                            <input
                              type="number"
                              value={collectionSettings.frontend_refresh_interval}
                              onChange={(e) => setCollectionSettings(prev => ({ ...prev, frontend_refresh_interval: Number(e.target.value) }))}
                              min={3}
                              max={60}
                              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                            />
                          </div>
                          <div className="flex items-center justify-between p-4 inner-card rounded-xl">
                            <div>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">跟随推送间隔</div>
                              <div className="text-xs text-slate-500">前端刷新间隔自动跟随设备采集间隔</div>
                            </div>
                            <button
                              onClick={() => setCollectionSettings(prev => ({ ...prev, follow_push_interval: !prev.follow_push_interval }))}
                              className={`w-12 h-6 rounded-full transition-all relative ${
                                collectionSettings.follow_push_interval
                                  ? "bg-gradient-to-r from-cyan-500 to-blue-600"
                                  : "bg-slate-300 dark:bg-white/20"
                              }`}
                            >
                              <span className={`absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-all ${collectionSettings.follow_push_interval ? "left-7" : "left-1"}`} />
                            </button>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">数据保留天数</label>
                            <input
                              type="number"
                              value={collectionSettings.data_retention_days}
                              onChange={(e) => setCollectionSettings(prev => ({ ...prev, data_retention_days: Number(e.target.value) }))}
                              min={7}
                              max={365}
                              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                            />
                            <p className="mt-1 text-xs text-slate-500">超过此天数的历史数据将被自动清理</p>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">设备离线超时（秒）</label>
                            <input
                              type="number"
                              value={collectionSettings.device_offline_timeout}
                              onChange={(e) => setCollectionSettings(prev => ({ ...prev, device_offline_timeout: Number(e.target.value) }))}
                              min={30}
                              max={600}
                              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                            />
                            <p className="mt-1 text-xs text-slate-500">超过此时间未收到数据则标记设备为离线</p>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* 通知设置 */}
                  {activeTab === "notification" && (
                    <div className="space-y-6">
                      <div>
                        <h3 className="text-lg font-semibold text-slate-800 dark:text-white mb-4">邮件通知</h3>
                        <div className="space-y-4">
                          <div className="flex items-center justify-between p-4 inner-card rounded-xl">
                            <div>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">启用邮件通知</div>
                              <div className="text-xs text-slate-500">设备离线、告警时发送邮件通知</div>
                            </div>
                            <button
                              onClick={() => setNotificationSettings(prev => ({ ...prev, emailEnabled: !prev.emailEnabled }))}
                              className={`w-12 h-6 rounded-full transition-all relative ${
                                notificationSettings.emailEnabled
                                  ? "bg-gradient-to-r from-cyan-500 to-blue-600"
                                  : "bg-slate-300 dark:bg-white/20"
                              }`}
                            >
                              <span className={`absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-all ${notificationSettings.emailEnabled ? "left-7" : "left-1"}`} />
                            </button>
                          </div>
                          {notificationSettings.emailEnabled && (
                            <>
                              <div className="grid grid-cols-2 gap-4">
                                <div>
                                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">SMTP 服务器</label>
                                  <input
                                    type="text"
                                    value={notificationSettings.emailServer}
                                    onChange={(e) => setNotificationSettings(prev => ({ ...prev, emailServer: e.target.value }))}
                                    className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                                  />
                                </div>
                                <div>
                                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">端口</label>
                                  <input
                                    type="number"
                                    value={notificationSettings.emailPort}
                                    onChange={(e) => setNotificationSettings(prev => ({ ...prev, emailPort: Number(e.target.value) }))}
                                    className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                                  />
                                </div>
                              </div>
                              <div>
                                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">发件人邮箱</label>
                                <input
                                  type="email"
                                  value={notificationSettings.emailUsername}
                                  onChange={(e) => setNotificationSettings(prev => ({ ...prev, emailUsername: e.target.value }))}
                                  className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                                />
                              </div>
                            </>
                          )}
                        </div>
                      </div>

                      <div>
                        <h3 className="text-lg font-semibold text-slate-800 dark:text-white mb-4">Webhook 通知</h3>
                        <div className="space-y-4">
                          <div className="flex items-center justify-between p-4 inner-card rounded-xl">
                            <div>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">启用 Webhook</div>
                              <div className="text-xs text-slate-500">通过 Webhook 推送告警信息</div>
                            </div>
                            <button
                              onClick={() => setNotificationSettings(prev => ({ ...prev, webhookEnabled: !prev.webhookEnabled }))}
                              className={`w-12 h-6 rounded-full transition-all relative ${
                                notificationSettings.webhookEnabled
                                  ? "bg-gradient-to-r from-cyan-500 to-blue-600"
                                  : "bg-slate-300 dark:bg-white/20"
                              }`}
                            >
                              <span className={`absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-all ${notificationSettings.webhookEnabled ? "left-7" : "left-1"}`} />
                            </button>
                          </div>
                          {notificationSettings.webhookEnabled && (
                            <div>
                              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Webhook URL</label>
                              <input
                                type="url"
                                value={notificationSettings.webhookUrl}
                                onChange={(e) => setNotificationSettings(prev => ({ ...prev, webhookUrl: e.target.value }))}
                                placeholder="https://example.com/webhook"
                                className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                              />
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  )}

                  {/* 安全设置 */}
                  {activeTab === "security" && (
                    <div className="space-y-6">
                      <div>
                        <h3 className="text-lg font-semibold text-slate-800 dark:text-white mb-4">安全设置</h3>
                        <div className="space-y-4">
                          <div className="grid grid-cols-2 gap-4">
                            <div>
                              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">会话超时（分钟）</label>
                              <input
                                type="number"
                                value={securitySettings.sessionTimeout}
                                onChange={(e) => setSecuritySettings(prev => ({ ...prev, sessionTimeout: Number(e.target.value) }))}
                                min={5}
                                max={1440}
                                className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                              />
                            </div>
                            <div>
                              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">最大登录尝试次数</label>
                              <input
                                type="number"
                                value={securitySettings.maxLoginAttempts}
                                onChange={(e) => setSecuritySettings(prev => ({ ...prev, maxLoginAttempts: Number(e.target.value) }))}
                                min={3}
                                max={10}
                                className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                              />
                            </div>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">密码最小长度</label>
                            <input
                              type="number"
                              value={securitySettings.passwordMinLength}
                              onChange={(e) => setSecuritySettings(prev => ({ ...prev, passwordMinLength: Number(e.target.value) }))}
                              min={6}
                              max={32}
                              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
                            />
                          </div>
                          <div className="flex items-center justify-between p-4 inner-card rounded-xl">
                            <div>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">要求特殊字符</div>
                              <div className="text-xs text-slate-500">密码必须包含特殊字符</div>
                            </div>
                            <button
                              onClick={() => setSecuritySettings(prev => ({ ...prev, requireSpecialChar: !prev.requireSpecialChar }))}
                              className={`w-12 h-6 rounded-full transition-all relative ${
                                securitySettings.requireSpecialChar
                                  ? "bg-gradient-to-r from-cyan-500 to-blue-600"
                                  : "bg-slate-300 dark:bg-white/20"
                              }`}
                            >
                              <span className={`absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-all ${securitySettings.requireSpecialChar ? "left-7" : "left-1"}`} />
                            </button>
                          </div>
                          <div className="flex items-center justify-between p-4 inner-card rounded-xl">
                            <div>
                              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">双因素认证</div>
                              <div className="text-xs text-slate-500">登录时需要验证码</div>
                            </div>
                            <button
                              onClick={() => setSecuritySettings(prev => ({ ...prev, twoFactorEnabled: !prev.twoFactorEnabled }))}
                              className={`w-12 h-6 rounded-full transition-all relative ${
                                securitySettings.twoFactorEnabled
                                  ? "bg-gradient-to-r from-cyan-500 to-blue-600"
                                  : "bg-slate-300 dark:bg-white/20"
                              }`}
                            >
                              <span className={`absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-all ${securitySettings.twoFactorEnabled ? "left-7" : "left-1"}`} />
                            </button>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* 外观设置 */}
                  {activeTab === "appearance" && (
                    <div className="space-y-6">
                      <div>
                        <h3 className="text-lg font-semibold text-slate-800 dark:text-white mb-4">外观设置</h3>
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">主题模式</label>
                            <div className="grid grid-cols-3 gap-4">
                              {[
                                { value: "light", label: "亮色", bg: "bg-white" },
                                { value: "dark", label: "暗色", bg: "bg-[#0f1729]" },
                                { value: "system", label: "跟随系统", bg: "bg-gradient-to-r from-white to-[#0f1729]" },
                              ].map((theme) => (
                                <button
                                  key={theme.value}
                                  className="p-4 inner-card rounded-xl border-2 border-transparent hover:border-cyan-500/50 transition-all"
                                >
                                  <div className={`w-full h-16 rounded-lg ${theme.bg} border border-slate-200 dark:border-white/10 mb-2`} />
                                  <span className="text-sm text-slate-700 dark:text-slate-300">{theme.label}</span>
                                </button>
                              ))}
                            </div>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">主题色</label>
                            <div className="flex items-center gap-3">
                              {["cyan", "blue", "violet", "emerald", "rose"].map((color) => (
                                <button
                                  key={color}
                                  className={`w-10 h-10 rounded-xl hover:scale-110 transition-transform ring-2 ring-offset-2 ring-offset-white dark:ring-offset-[#0f1729] ${color === "cyan" ? "ring-cyan-500" : "ring-transparent"}`}
                                  style={{ backgroundColor: color === "cyan" ? "#06b6d4" : color === "blue" ? "#3b82f6" : color === "violet" ? "#8b5cf6" : color === "emerald" ? "#10b981" : "#f43f5e" }}
                                />
                              ))}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* 保存按钮 */}
                  <div className="flex justify-end pt-6 mt-6 border-t border-slate-200 dark:border-white/10">
                    <button
                      onClick={handleSave}
                      disabled={saving}
                      className="px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-xl shadow-lg shadow-cyan-500/30 transition-all disabled:opacity-50"
                    >
                      {saving ? "保存中..." : "保存设置"}
                    </button>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
