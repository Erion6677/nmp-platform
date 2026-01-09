"use client";

import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import { SunIcon, MoonIcon } from "@heroicons/react/24/outline";

interface HeaderProps {
  title: string;
  breadcrumb?: string[];
}

export default function Header({ title, breadcrumb = [] }: HeaderProps) {
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  return (
    <header className="h-16 bg-white/80 dark:bg-[#1a2744]/60 backdrop-blur-xl border-b border-slate-200 dark:border-white/10 flex items-center justify-between px-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-800 dark:text-white">
          {title}
        </h1>
        {breadcrumb.length > 0 && (
          <div className="flex items-center gap-2 text-xs text-slate-500 mt-0.5">
            <span>首页</span>
            {breadcrumb.map((item, index) => (
              <span key={index} className="flex items-center gap-2">
                <span>/</span>
                <span
                  className={
                    index === breadcrumb.length - 1
                      ? "text-slate-700 dark:text-slate-400"
                      : ""
                  }
                >
                  {item}
                </span>
              </span>
            ))}
          </div>
        )}
      </div>

      <div className="flex items-center gap-3">
        {mounted && (
          <button
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            className="w-9 h-9 rounded-xl bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
            title={theme === "dark" ? "切换到亮色模式" : "切换到暗色模式"}
          >
            {theme === "dark" ? (
              <SunIcon className="w-5 h-5 text-amber-400" />
            ) : (
              <MoonIcon className="w-5 h-5 text-slate-600" />
            )}
          </button>
        )}
      </div>
    </header>
  );
}
