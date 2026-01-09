"use client";

import { Card, CardHeader, CardBody, Chip } from "@heroui/react";
import { ExclamationCircleIcon, ExclamationTriangleIcon } from "@heroicons/react/24/outline";

interface Alert {
  id: number;
  message: string;
  time: string;
  level: "error" | "warning";
}

const mockAlerts: Alert[] = [
  { id: 1, message: "Web服务器-01 连接超时", time: "2 分钟前", level: "error" },
  { id: 2, message: "边界防火墙 CPU 使用率过高", time: "15 分钟前", level: "warning" },
  { id: 3, message: "核心路由器-01 内存使用率 85%", time: "1 小时前", level: "warning" },
];

export default function AlertList() {
  return (
    <Card className="bg-white dark:bg-[#1a2744]/80 border border-slate-200 dark:border-white/10">
      <CardHeader className="flex justify-between items-center px-5 py-4 border-b border-slate-200 dark:border-white/10">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-cyan-400" />
          <span className="font-medium text-slate-800 dark:text-white">最近告警</span>
        </div>
        <Chip size="sm" color="danger" variant="flat">
          {mockAlerts.length}
        </Chip>
      </CardHeader>
      <CardBody className="p-3 space-y-2 max-h-52 overflow-y-auto">
        {mockAlerts.map((alert) => (
          <div
            key={alert.id}
            className={`flex items-start gap-3 p-3 rounded-xl border-l-2 ${
              alert.level === "error"
                ? "bg-rose-500/5 dark:bg-rose-500/10 border-rose-500"
                : "bg-amber-500/5 dark:bg-amber-500/10 border-amber-500"
            }`}
          >
            {alert.level === "error" ? (
              <ExclamationCircleIcon className="w-4 h-4 text-rose-500 mt-0.5 flex-shrink-0" />
            ) : (
              <ExclamationTriangleIcon className="w-4 h-4 text-amber-500 mt-0.5 flex-shrink-0" />
            )}
            <div className="flex-1 min-w-0">
              <div className="text-sm text-slate-700 dark:text-slate-200">{alert.message}</div>
              <div className="text-xs text-slate-500 mt-1">{alert.time}</div>
            </div>
          </div>
        ))}
      </CardBody>
    </Card>
  );
}
