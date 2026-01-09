"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { ServerStackIcon, EyeIcon, EyeSlashIcon } from "@heroicons/react/24/outline";

export default function LoginPage() {
  const router = useRouter();
  const { login, isAuthenticated, isLoading, error, clearError } = useAuthStore();
  
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [localError, setLocalError] = useState("");

  useEffect(() => {
    if (isAuthenticated) {
      router.push("/");
    }
  }, [isAuthenticated, router]);

  useEffect(() => {
    if (error) {
      setLocalError(error);
      clearError();
    }
  }, [error, clearError]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLocalError("");

    if (!username.trim()) {
      setLocalError("请输入用户名");
      return;
    }
    if (!password) {
      setLocalError("请输入密码");
      return;
    }

    try {
      const success = await login(username, password);
      if (success) {
        router.push("/");
      }
    } catch (err) {
      console.error("Login error:", err);
      setLocalError("登录失败，请检查后端服务是否启动");
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4" style={{ background: "linear-gradient(135deg, #0f1729 0%, #1a2744 50%, #0f1729 100%)" }}>
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-cyan-400 to-blue-500 shadow-lg shadow-cyan-500/30 mb-4">
            <ServerStackIcon className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-cyan-400 to-blue-400 bg-clip-text text-transparent">
            NMP
          </h1>
          <p className="text-slate-400 mt-2">网络监控平台</p>
        </div>

        {/* 登录表单 */}
        <div className="glass-card rounded-2xl p-8">
          <h2 className="text-xl font-semibold text-white mb-6 text-center">登录</h2>

          <form onSubmit={handleSubmit} className="space-y-5">
            {/* 错误提示 */}
            {localError && (
              <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-sm">
                {localError}
              </div>
            )}

            {/* 用户名 */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                用户名
              </label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="请输入用户名"
                autoComplete="username"
                className="w-full px-4 py-3 text-sm bg-[#0f1729]/50 border border-white/10 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all"
              />
            </div>

            {/* 密码 */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                密码
              </label>
              <div className="relative">
                <input
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="请输入密码"
                  autoComplete="current-password"
                  className="w-full px-4 py-3 pr-12 text-sm bg-[#0f1729]/50 border border-white/10 rounded-xl text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50 transition-all"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-300"
                >
                  {showPassword ? (
                    <EyeSlashIcon className="w-5 h-5" />
                  ) : (
                    <EyeIcon className="w-5 h-5" />
                  )}
                </button>
              </div>
            </div>

            {/* 登录按钮 */}
            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-3 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-xl shadow-lg shadow-cyan-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="animate-spin w-5 h-5" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  登录中...
                </span>
              ) : (
                "登录"
              )}
            </button>
          </form>

          {/* 提示信息 */}
          <div className="mt-6 text-center text-xs text-slate-500">
            默认账号: admin / admin
          </div>
        </div>

        {/* 版权信息 */}
        <div className="mt-8 text-center text-xs text-slate-500">
          © 2024 NMP Network Monitoring Platform
        </div>
      </div>
    </div>
  );
}
