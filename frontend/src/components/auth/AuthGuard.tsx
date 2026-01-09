"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { getToken } from "@/lib/api/client";

// 不需要认证的路径
const publicPaths = ["/login"];

export default function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, fetchUser, isLoading } = useAuthStore();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    const checkAuth = async () => {
      const token = getToken();
      const isPublicPath = publicPaths.includes(pathname);

      if (!token) {
        if (!isPublicPath) {
          router.push("/login");
        }
        setChecking(false);
        return;
      }

      // 有 token，尝试获取用户信息
      if (!isAuthenticated) {
        await fetchUser();
      }
      
      setChecking(false);
    };

    checkAuth();
  }, [pathname, isAuthenticated, fetchUser, router]);

  // 检查中显示加载状态
  if (checking || isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center" style={{ background: "linear-gradient(135deg, #0f1729 0%, #1a2744 50%, #0f1729 100%)" }}>
        <div className="text-center">
          <div className="w-12 h-12 border-4 border-cyan-500/30 border-t-cyan-500 rounded-full animate-spin mx-auto mb-4" />
          <p className="text-slate-400">加载中...</p>
        </div>
      </div>
    );
  }

  // 公开页面直接显示
  if (publicPaths.includes(pathname)) {
    return <>{children}</>;
  }

  // 未认证跳转登录
  if (!isAuthenticated) {
    return null;
  }

  return <>{children}</>;
}
