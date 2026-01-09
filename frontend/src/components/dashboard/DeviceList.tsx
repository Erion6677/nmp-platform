"use client";

import { Card, CardHeader, CardBody, Button, Chip } from "@heroui/react";
import { ArrowPathIcon, ChevronRightIcon } from "@heroicons/react/24/outline";
import Link from "next/link";

interface Device {
  id: number;
  name: string;
  host: string;
  type: string;
  status: "online" | "offline" | "warning";
}

const mockDevices: Device[] = [
  { id: 1, name: "核心路由器-01", host: "192.168.1.1", type: "mikrotik", status: "online" },
  { id: 2, name: "接入交换机-A1", host: "192.168.1.10", type: "switch", status: "online" },
  { id: 3, name: "Web服务器-01", host: "192.168.2.100", type: "linux", status: "offline" },
  { id: 4, name: "边界防火墙", host: "10.0.0.1", type: "firewall", status: "warning" },
  { id: 5, name: "数据库服务器", host: "192.168.2.50", type: "linux", status: "online" },
];

const typeLabels: Record<string, { label: string; color: "primary" | "success" | "warning" | "danger" }> = {
  mikrotik: { label: "MikroTik", color: "primary" },
  linux: { label: "Linux", color: "success" },
  switch: { label: "交换机", color: "warning" },
  firewall: { label: "防火墙", color: "danger" },
};

const statusColors = {
  online: "bg-emerald-400 shadow-emerald-400/50",
  offline: "bg-rose-400 shadow-rose-400/50",
  warning: "bg-amber-400 shadow-amber-400/50 animate-pulse",
};

export default function DeviceList() {
  return (
    <Card className="bg-white dark:bg-[#1a2744]/80 border border-slate-200 dark:border-white/10">
      <CardHeader className="flex justify-between items-center px-5 py-4 border-b border-slate-200 dark:border-white/10">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-cyan-400" />
          <span className="font-medium text-slate-800 dark:text-white">设备状态</span>
        </div>
        <Button isIconOnly size="sm" variant="flat" className="bg-slate-100 dark:bg-white/5">
          <ArrowPathIcon className="w-4 h-4" />
        </Button>
      </CardHeader>
      <CardBody className="p-3 space-y-2 max-h-80 overflow-y-auto">
        {mockDevices.map((device) => (
          <Link
            key={device.id}
            href={`/devices/${device.id}`}
            className="flex items-center gap-3 p-3 rounded-xl hover:bg-slate-100 dark:hover:bg-white/5 border border-transparent hover:border-slate-200 dark:hover:border-white/10 transition-all group cursor-pointer"
          >
            <div className={`w-2.5 h-2.5 rounded-full ${statusColors[device.status]} shadow-lg`} />
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-slate-800 dark:text-slate-200">
                {device.name}
              </div>
              <div className="text-xs text-slate-500 font-mono">{device.host}</div>
            </div>
            <Chip
              size="sm"
              variant="flat"
              color={typeLabels[device.type]?.color || "default"}
              classNames={{
                base: "h-6",
                content: "text-xs font-medium",
              }}
            >
              {typeLabels[device.type]?.label || device.type}
            </Chip>
            <ChevronRightIcon className="w-4 h-4 text-slate-400 group-hover:text-cyan-500 group-hover:translate-x-1 transition-all" />
          </Link>
        ))}
      </CardBody>
    </Card>
  );
}
