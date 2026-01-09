"use client";

import { useEffect, useRef } from "react";
import * as echarts from "echarts";
import { useTheme } from "next-themes";
import type { PingPoint } from "@/lib/api/types";

interface PingChartProps {
  data: PingPoint[];
  height?: number;
  timeRange?: string;
}

// 智能格式化延迟单位
// 输入：微秒（μs）
function formatLatency(us: number): string {
  if (us >= 1000000) return `${(us / 1000000).toFixed(2)} s`;
  if (us >= 1000) return `${(us / 1000).toFixed(2)} ms`;
  if (us > 0) return `${us.toFixed(0)} μs`;
  return "0 ms";
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

export default function PingChart({ data, height = 160, timeRange }: PingChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);
  const { theme } = useTheme();

  useEffect(() => {
    if (!chartRef.current) return;

    if (!chartInstance.current) {
      chartInstance.current = echarts.init(chartRef.current);
    }

    const isDark = theme === "dark";
    
    const times = data.map(d => formatTimeLabel(d.timestamp, timeRange));
    const latencyData = data.map(d => d.latency);
    const lossData = data.map(d => d.is_loss ? 100 : 0);

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
            const color = item.seriesName === "延迟" ? "#34d399" : "#f87171";
            let value = "";
            if (item.seriesName === "延迟") {
              value = formatLatency(item.value || 0);
            } else {
              value = `${(item.value || 0).toFixed(1)}%`;
            }
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
        right: 50,
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
        },
        axisTick: { show: false },
        splitLine: { show: false },
      },
      yAxis: [
        {
          type: "value",
          name: "μs",
          nameTextStyle: {
            color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
            fontSize: 10,
          },
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
            formatter: (v: number) => {
              if (v >= 1000000) return `${(v / 1000000).toFixed(0)}s`;
              if (v >= 1000) return `${(v / 1000).toFixed(0)}ms`;
              return `${v}μs`;
            },
          },
          splitNumber: 4,
        },
        {
          type: "value",
          name: "%",
          nameTextStyle: {
            color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
            fontSize: 10,
          },
          min: 0,
          max: 100,
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: {
            color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
            fontSize: 10,
          },
        },
      ],
      series: [
        {
          name: "延迟",
          type: "line",
          smooth: true,
          symbol: "none",
          data: latencyData,
          yAxisIndex: 0,
          lineStyle: {
            width: 2,
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: "#34d399" },
              { offset: 1, color: "#10b981" },
            ]),
          },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: "rgba(52, 211, 153, 0.3)" },
              { offset: 1, color: "rgba(52, 211, 153, 0)" },
            ]),
          },
        },
        {
          name: "丢包",
          type: "bar",
          data: lossData,
          yAxisIndex: 1,
          barWidth: 4,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: "#f87171" },
              { offset: 1, color: "#ef4444" },
            ]),
            borderRadius: [2, 2, 0, 0],
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
