"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import { useAuthStore } from "@/stores/auth";
import {
  UserCircleIcon,
  KeyIcon,
  EnvelopeIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
} from "@heroicons/react/24/outline";
import api from "@/lib/api/client";

interface ProfileFormData {
  full_name: string;
  email: string;
}

interface PasswordFormData {
  old_password: string;
  new_password: string;
  confirm_password: string;
}

export default function ProfilePage() {
  const router = useRouter();
  const { user, isAuthenticated, fetchUser } = useAuthStore();
  
  const [activeTab, setActiveTab] = useState<"profile" | "password">("profile");
  const [isLoading, setIsLoading] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  
  const [profileForm, setProfileForm] = useState<ProfileFormData>({
    full_name: "",
    email: "",
  });
  
  const [passwordForm, setPasswordForm] = useState<PasswordFormData>({
    old_password: "",
    new_password: "",
    confirm_password: "",
  });

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }
    
    if (user) {
      setProfileForm({
        full_name: user.full_name || "",
        email: user.email || "",
      });
    }
  }, [user, isAuthenticated, router]);

  const handleProfileSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setMessage(null);

    try {
      const response = await api.put<{ success: boolean; data?: unknown; error?: string }>("/api/v1/auth/profile", profileForm);
      if (response.success) {
        setMessage({ type: "success", text: "个人信息更新成功" });
        await fetchUser();
      } else {
        setMessage({ type: "error", text: response.error || "更新失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "更新失败" });
    } finally {
      setIsLoading(false);
    }
  };

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setMessage(null);

    if (passwordForm.new_password !== passwordForm.confirm_password) {
      setMessage({ type: "error", text: "两次输入的新密码不一致" });
      setIsLoading(false);
      return;
    }

    if (passwordForm.new_password.length < 6) {
      setMessage({ type: "error", text: "新密码长度至少6位" });
      setIsLoading(false);
      return;
    }

    try {
      const response = await api.put<{ success: boolean; data?: unknown; error?: string }>("/api/v1/auth/password", {
        old_password: passwordForm.old_password,
        new_password: passwordForm.new_password,
      });
      
      if (response.success) {
        setMessage({ type: "success", text: "密码修改成功" });
        setPasswordForm({
          old_password: "",
          new_password: "",
          confirm_password: "",
        });
      } else {
        setMessage({ type: "error", text: response.error || "密码修改失败" });
      }
    } catch (error) {
      setMessage({ type: "error", text: error instanceof Error ? error.message : "密码修改失败" });
    } finally {
      setIsLoading(false);
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <MainLayout>
      <Header title="个人中心" breadcrumb={["个人中心"]} />
      
      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-2xl mx-auto">
          {/* 用户头像和基本信息 */}
          <div className="bg-white dark:bg-[#1e2d47] rounded-2xl p-6 mb-6 border border-slate-200 dark:border-white/10">
            <div className="flex items-center gap-4">
              <div className="w-20 h-20 rounded-full bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center text-3xl font-bold text-white shadow-lg shadow-violet-500/30">
                {user?.username?.charAt(0).toUpperCase() || "A"}
              </div>
              <div>
                <h2 className="text-xl font-semibold text-slate-800 dark:text-white">
                  {user?.full_name || user?.username}
                </h2>
                <p className="text-slate-500 dark:text-slate-400">@{user?.username}</p>
                <p className="text-sm text-slate-400 dark:text-slate-500 mt-1">
                  角色: {user?.roles?.join(", ") || "用户"}
                </p>
              </div>
            </div>
          </div>

          {/* 标签页 */}
          <div className="bg-white dark:bg-[#1e2d47] rounded-2xl border border-slate-200 dark:border-white/10 overflow-hidden">
            <div className="flex border-b border-slate-200 dark:border-white/10">
              <button
                onClick={() => { setActiveTab("profile"); setMessage(null); }}
                className={`flex-1 px-6 py-4 text-sm font-medium transition-all flex items-center justify-center gap-2 ${
                  activeTab === "profile"
                    ? "text-cyan-600 dark:text-cyan-400 border-b-2 border-cyan-500 bg-cyan-50/50 dark:bg-cyan-500/10"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200"
                }`}
              >
                <UserCircleIcon className="w-5 h-5" />
                个人信息
              </button>
              <button
                onClick={() => { setActiveTab("password"); setMessage(null); }}
                className={`flex-1 px-6 py-4 text-sm font-medium transition-all flex items-center justify-center gap-2 ${
                  activeTab === "password"
                    ? "text-cyan-600 dark:text-cyan-400 border-b-2 border-cyan-500 bg-cyan-50/50 dark:bg-cyan-500/10"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200"
                }`}
              >
                <KeyIcon className="w-5 h-5" />
                修改密码
              </button>
            </div>

            <div className="p-6">
              {/* 消息提示 */}
              {message && (
                <div
                  className={`mb-6 p-4 rounded-xl flex items-center gap-3 ${
                    message.type === "success"
                      ? "bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 border border-green-200 dark:border-green-500/20"
                      : "bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 border border-red-200 dark:border-red-500/20"
                  }`}
                >
                  {message.type === "success" ? (
                    <CheckCircleIcon className="w-5 h-5" />
                  ) : (
                    <ExclamationCircleIcon className="w-5 h-5" />
                  )}
                  {message.text}
                </div>
              )}

              {/* 个人信息表单 */}
              {activeTab === "profile" && (
                <form onSubmit={handleProfileSubmit} className="space-y-5">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      用户名
                    </label>
                    <input
                      type="text"
                      value={user?.username || ""}
                      disabled
                      className="w-full px-4 py-3 rounded-xl bg-slate-100 dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-500 dark:text-slate-400 cursor-not-allowed"
                    />
                    <p className="mt-1 text-xs text-slate-400">用户名不可修改</p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      <UserCircleIcon className="w-4 h-4 inline mr-1" />
                      显示名称
                    </label>
                    <input
                      type="text"
                      value={profileForm.full_name}
                      onChange={(e) => setProfileForm({ ...profileForm, full_name: e.target.value })}
                      placeholder="请输入显示名称"
                      className="w-full px-4 py-3 rounded-xl bg-white dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      <EnvelopeIcon className="w-4 h-4 inline mr-1" />
                      邮箱地址
                    </label>
                    <input
                      type="email"
                      value={profileForm.email}
                      onChange={(e) => setProfileForm({ ...profileForm, email: e.target.value })}
                      placeholder="请输入邮箱地址"
                      className="w-full px-4 py-3 rounded-xl bg-white dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 transition-all"
                    />
                  </div>

                  <button
                    type="submit"
                    disabled={isLoading}
                    className="w-full py-3 px-4 rounded-xl bg-gradient-to-r from-cyan-500 to-blue-500 text-white font-medium hover:from-cyan-600 hover:to-blue-600 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
                  >
                    {isLoading ? "保存中..." : "保存修改"}
                  </button>
                </form>
              )}

              {/* 修改密码表单 */}
              {activeTab === "password" && (
                <form onSubmit={handlePasswordSubmit} className="space-y-5">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      当前密码
                    </label>
                    <input
                      type="password"
                      value={passwordForm.old_password}
                      onChange={(e) => setPasswordForm({ ...passwordForm, old_password: e.target.value })}
                      placeholder="请输入当前密码"
                      required
                      className="w-full px-4 py-3 rounded-xl bg-white dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      新密码
                    </label>
                    <input
                      type="password"
                      value={passwordForm.new_password}
                      onChange={(e) => setPasswordForm({ ...passwordForm, new_password: e.target.value })}
                      placeholder="请输入新密码（至少6位）"
                      required
                      minLength={6}
                      className="w-full px-4 py-3 rounded-xl bg-white dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 transition-all"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                      确认新密码
                    </label>
                    <input
                      type="password"
                      value={passwordForm.confirm_password}
                      onChange={(e) => setPasswordForm({ ...passwordForm, confirm_password: e.target.value })}
                      placeholder="请再次输入新密码"
                      required
                      minLength={6}
                      className="w-full px-4 py-3 rounded-xl bg-white dark:bg-white/5 border border-slate-200 dark:border-white/10 text-slate-800 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500 transition-all"
                    />
                  </div>

                  <button
                    type="submit"
                    disabled={isLoading}
                    className="w-full py-3 px-4 rounded-xl bg-gradient-to-r from-cyan-500 to-blue-500 text-white font-medium hover:from-cyan-600 hover:to-blue-600 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
                  >
                    {isLoading ? "修改中..." : "修改密码"}
                  </button>
                </form>
              )}
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
