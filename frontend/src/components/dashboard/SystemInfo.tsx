"use client";

import { Card, CardHeader, CardBody, Progress } from "@heroui/react";

export default function SystemInfo() {
  return (
    <Card className="bg-white dark:bg-[#1a2744]/80 border border-slate-200 dark:border-white/10">
      <CardHeader className="px-5 py-4 border-b border-slate-200 dark:border-white/10">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-cyan-400" />
          <span className="font-medium text-slate-800 dark:text-white">系统信息</span>
        </div>
      </CardHeader>
      <CardBody className="p-5 space-y-4">
        <div className="flex items-center gap-3">
          <span className="text-sm text-slate-500 w-24">系统版本</span>
          <span className="text-sm text-slate-800 dark:text-slate-200 font-medium">v1.0.0</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-slate-500 w-24">运行时间</span>
          <span className="text-sm text-slate-800 dark:text-slate-200 font-medium">15 天 8 小时</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-slate-500 w-24">CPU 使用率</span>
          <Progress
            size="sm"
            value={35}
            color="primary"
            className="flex-1"
            classNames={{
              track: "bg-slate-200 dark:bg-white/10",
              indicator: "bg-gradient-to-r from-cyan-400 to-blue-500",
            }}
          />
          <span className="text-sm text-slate-600 dark:text-slate-300 font-mono w-12 text-right">35%</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-slate-500 w-24">内存使用率</span>
          <Progress
            size="sm"
            value={62}
            color="secondary"
            className="flex-1"
            classNames={{
              track: "bg-slate-200 dark:bg-white/10",
              indicator: "bg-gradient-to-r from-violet-400 to-purple-500",
            }}
          />
          <span className="text-sm text-slate-600 dark:text-slate-300 font-mono w-12 text-right">62%</span>
        </div>
      </CardBody>
    </Card>
  );
}
