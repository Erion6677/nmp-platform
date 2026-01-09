"use client";

import { useEffect, useRef } from "react";
import * as echarts from "echarts";
import { useTheme } from "next-themes";
import type { BandwidthPoint } from "@/lib/api/types";

interface BandwidthChartProps {
  data: BandwidthPoint[];
  height?: number;
  timeRange?: string; // "10m" | "30m" | "1h" | "3h" | "6h" | "24h"
}

// 智能格式化带宽单位
function formatBandwidth(bps: number): string {
  if (bps >= 1000000000) return `${(bps / 1000000000).toFixed(2)} Gbps`;
  if (bps >= 1000000) return `${(bps / 1000000).toFixed(2)} Mbps`;
  if (bps >= 1000) return `${(bps / 1000).toFixed(2)} Kbps`;
  return `${bps.toFixed(0)} bps`;
}

// Y轴标签格式化
function formatYAxisLabel(bps: number): string {
  if (bps >= 1000000000) return `${(bps / 1000000000).toFixed(0)}G`;
  if (bps >= 1000000) return `${(bps / 1000000).toFixed(0)}M`;
  if (bps >= 1000) return `${(bps / 1000).toFixed(0)}K`;
  return `${bps.toFixed(0)}`;
}

// 根据时间范围格式化X轴时间
function formatTimeLabel(timestamp: string, timeRange?: string): string {
  const date = new Date(timestamp);
  // 12小时以内显示 HH:mm:ss
  if (!timeRange || ["10m", "30m", "1h", "3h", "6h"].includes(timeRange)) {
    return date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  }
  // 24小时显示 HH:mm
  if (timeRange === "24h") {
    return date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
  }
  // 更长时间显示日期+时间
  return date.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export default function BandwidthChart({ data, height = 160, timeRange }: BandwidthChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);
  const { theme } = useTheme();

  useEffect(() => {
    if (!chartRef.current) return;

    if (!chartInstance.current) {
      chartInstance.current = echarts.init(chartRef.current);
    }

    const isDark = theme === "dark";
    
    // 格式化时间标签
    const times = data.map(d => formatTimeLabel(d.timestamp, timeRange));
    
    // 保持原始 bps 单位用于显示
    const rxData = data.map(d => d.rx_rate);
    const txData = data.map(d => d.tx_rate);

    const option: echarts.EChartsOption = {
      backgroundColor: "transparent",
      tooltip: {
        trigger: "axis",
        backgroundColor: isDark ? "rgba(26, 39, 68, 0.95)" : "rgba(255, 255, 255, 0.95)",
        borderColor: isDark ? "rgba(255, 255, 255, 0.1)" : "rgba(0, 0, 0, 0.1)",
        borderWidth: 1,
        textStyle: {
          color: isDark ? "#fff" : "#334155",
          fontSize: 12,
        },
        formatter: (params: unknown) => {
          const p = params as { axisValue?: string; seriesName?: string; value?: number }[];
          const time = p[0]?.axisValue || "";
          let html = `<div style="font-weight: 500; margin-bottom: 8px;">${time}</div>`;
          p.forEach((item) => {
            const color = item.seriesName === "接收" ? "#22d3ee" : "#a78bfa";
            const value = formatBandwidth(item.value || 0);
            html += `<div style="display: flex; align-items: center; gap: 8px; margin: 4px 0;">
              <span style="width: 8px; height: 8px; border-radius: 50%; background: ${color};"></span>
              <span>${item.seriesName}: ${value}</span>
            </div>`;
          });
          return html;
        },
      },
      grid: {
        left: 50,
        right: 15,
        top: 15,
        bottom: 30,
      },
      xAxis: {
        type: "category",
        data: times,
        axisLine: {
          lineStyle: { color: isDark ? "rgba(255, 255, 255, 0.1)" : "rgba(0, 0, 0, 0.1)" },
        },
        axisLabel: {
          color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
          fontSize: 10,
          interval: "auto",
          rotate: 0,
        },
        axisTick: { show: false },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: {
          lineStyle: { 
            color: isDark ? "rgba(255, 255, 255, 0.06)" : "rgba(0, 0, 0, 0.06)",
            type: "dashed"
          },
        },
        axisLabel: {
          color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
          fontSize: 10,
          formatter: (v: number) => formatYAxisLabel(v),
        },
        splitNumber: 4,
      },
      series: [
        {
          name: "接收",
          type: "line",
          smooth: true,
          symbol: "none",
          data: rxData,
          lineStyle: {
            width: 2,
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: "#22d3ee" },
              { offset: 1, color: "#3b82f6" },
            ]),
          },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: "rgba(34, 211, 238, 0.3)" },
              { offset: 1, color: "rgba(34, 211, 238, 0)" },
            ]),
          },
        },
        {
          name: "发送",
          type: "line",
          smooth: true,
          symbol: "none",
          data: txData,
          lineStyle: {
            width: 2,
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: "#a78bfa" },
              { offset: 1, color: "#8b5cf6" },
            ]),
          },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: "rgba(167, 139, 250, 0.3)" },
              { offset: 1, color: "rgba(167, 139, 250, 0)" },
            ]),
          },
        },
      ],
    };

    chartInstance.current.setOption(option, true);

    const handleResize = () => chartInstance.current?.resize();
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [data, theme, timeRange]);

  useEffect(() => {
    return () => { chartInstance.current?.dispose(); };
  }, []);

  if (!data || data.length === 0) {
    return (
      <div className="w-full flex items-center justify-center text-slate-400 dark:text-slate-500 text-xs" style={{ height }}>
        暂无数据
      </div>
    );
  }

  return <div ref={chartRef} className="w-full" style={{ height }} />;
}
