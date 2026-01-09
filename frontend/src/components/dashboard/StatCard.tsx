"use client";

import { Card, CardBody } from "@heroui/react";

interface StatCardProps {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  color: "cyan" | "emerald" | "rose" | "amber";
}

const colorClasses = {
  cyan: {
    iconBg: "bg-cyan-500/10 dark:bg-cyan-500/20 border-cyan-500/20",
    iconColor: "text-cyan-500 dark:text-cyan-400",
    valueColor: "text-cyan-600 dark:text-cyan-400",
    glow: "hover:shadow-cyan-500/20",
    border: "hover:border-cyan-500/30",
  },
  emerald: {
    iconBg: "bg-emerald-500/10 dark:bg-emerald-500/20 border-emerald-500/20",
    iconColor: "text-emerald-500 dark:text-emerald-400",
    valueColor: "text-emerald-600 dark:text-emerald-400",
    glow: "hover:shadow-emerald-500/20",
    border: "hover:border-emerald-500/30",
  },
  rose: {
    iconBg: "bg-rose-500/10 dark:bg-rose-500/20 border-rose-500/20",
    iconColor: "text-rose-500 dark:text-rose-400",
    valueColor: "text-rose-600 dark:text-rose-400",
    glow: "hover:shadow-rose-500/20",
    border: "hover:border-rose-500/30",
  },
  amber: {
    iconBg: "bg-amber-500/10 dark:bg-amber-500/20 border-amber-500/20",
    iconColor: "text-amber-500 dark:text-amber-400",
    valueColor: "text-amber-600 dark:text-amber-400",
    glow: "hover:shadow-amber-500/20",
    border: "hover:border-amber-500/30",
  },
};

export default function StatCard({ title, value, icon, color }: StatCardProps) {
  const classes = colorClasses[color];

  return (
    <Card
      className={`bg-white dark:bg-[#1a2744]/80 border border-slate-200 dark:border-white/10 hover:-translate-y-1 transition-all duration-300 shadow-sm hover:shadow-lg ${classes.glow} ${classes.border}`}
    >
      <CardBody className="p-5">
        <div className="flex items-center gap-4">
          <div
            className={`w-12 h-12 rounded-xl ${classes.iconBg} border flex items-center justify-center`}
          >
            <div className={classes.iconColor}>{icon}</div>
          </div>
          <div>
            <div className={`text-3xl font-bold ${classes.valueColor}`}>
              {value}
            </div>
            <div className="text-sm text-slate-500 dark:text-slate-400">
              {title}
            </div>
          </div>
        </div>
      </CardBody>
    </Card>
  );
}
