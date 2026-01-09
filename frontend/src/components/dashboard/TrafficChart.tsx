"use client";

import { useEffect, useRef, useState } from "react";
import { Card, CardHeader, CardBody, Select, SelectItem } from "@heroui/react";
import * as echarts from "echarts";
import { useTheme } from "next-themes";
import { metricsApi } from "@/lib/api/metrics";

interface TrafficData {
  time: string;
  inbound: number;
  outbound: number;
}

export default function TrafficChart() {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);
  const { theme } = useTheme();
  const [timeRange, setTimeRange] = useState("1h");
  const [data, setData] = useState<TrafficData[]>([]);
  const [loading, setLoading] = useState(true);

  // 生成模拟数据（当 API 无数据时使用）
  const generateMockData = (range: string): TrafficData[] => {
    const now = new Date();
    const points = range === "1h" ? 60 : range === "6h" ? 72 : range === "24h" ? 96 : 168;
    const interval = range === "1h" ? 1 : range === "6h" ? 5 : range === "24h" ? 15 : 60;
    
    return Array.from({ length: points }, (_, i) => {
      const time = new Date(now.getTime() - (points - i - 1) * interval * 60 * 1000);
      const baseIn = 50 + Math.random() * 30;
      const baseOut = 30 + Math.random() * 20;
      return {
        time: time.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
        inbound: Math.round((baseIn + Math.sin(i / 10) * 20) * 100) / 100,
        outbound: Math.round((baseOut + Math.cos(i / 8) * 15) * 100) / 100,
      };
    });
  };

  // 获取流量数据
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const response = await metricsApi.queryTotalTraffic(timeRange as any);
        if (response?.points?.length > 0) {
          const formatted = response.points.map((item: any) => ({
            time: new Date(item.timestamp).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
            inbound: item.inbound ? item.inbound / 1024 / 1024 : 0,
            outbound: item.outbound ? item.outbound / 1024 / 1024 : 0,
          }));
          setData(formatted);
        } else {
          // 使用模拟数据
          setData(generateMockData(timeRange));
        }
      } catch (error) {
        console.error("Failed to fetch traffic data:", error);
        setData(generateMockData(timeRange));
      } finally {
        setLoading(false);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 30000); // 每30秒刷新
    return () => clearInterval(interval);
  }, [timeRange]);

  // 初始化和更新图表
  useEffect(() => {
    if (!chartRef.current) return;

    if (!chartInstance.current) {
      chartInstance.current = echarts.init(chartRef.current);
    }

    const isDark = theme === "dark";
    
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
        formatter: (params: any) => {
          const time = params[0]?.axisValue || "";
          let html = `<div style="font-weight: 500; margin-bottom: 8px;">${time}</div>`;
          params.forEach((item: any) => {
            const color = item.seriesName === "入站" ? "#22d3ee" : "#a78bfa";
            html += `<div style="display: flex; align-items: center; gap: 8px; margin: 4px 0;">
              <span style="width: 8px; height: 8px; border-radius: 50%; background: ${color};"></span>
              <span>${item.seriesName}: ${item.value.toFixed(2)} Mbps</span>
            </div>`;
          });
          return html;
        },
      },
      legend: {
        show: false,
      },
      grid: {
        left: 50,
        right: 20,
        top: 20,
        bottom: 30,
      },
      xAxis: {
        type: "category",
        data: data.map((d) => d.time),
        axisLine: {
          lineStyle: {
            color: isDark ? "rgba(255, 255, 255, 0.1)" : "rgba(0, 0, 0, 0.1)",
          },
        },
        axisLabel: {
          color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
          fontSize: 10,
          interval: Math.floor(data.length / 6),
        },
        axisTick: { show: false },
      },
      yAxis: {
        type: "value",
        name: "Mbps",
        nameTextStyle: {
          color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
          fontSize: 10,
        },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: {
          lineStyle: {
            color: isDark ? "rgba(255, 255, 255, 0.05)" : "rgba(0, 0, 0, 0.05)",
          },
        },
        axisLabel: {
          color: isDark ? "rgba(255, 255, 255, 0.5)" : "rgba(0, 0, 0, 0.5)",
          fontSize: 10,
        },
      },
      series: [
        {
          name: "入站",
          type: "line",
          smooth: true,
          symbol: "none",
          data: data.map((d) => d.inbound),
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
          name: "出站",
          type: "line",
          smooth: true,
          symbol: "none",
          data: data.map((d) => d.outbound),
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

    chartInstance.current.setOption(option);

    const handleResize = () => {
      chartInstance.current?.resize();
    };
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, [data, theme]);

  // 清理
  useEffect(() => {
    return () => {
      chartInstance.current?.dispose();
    };
  }, []);

  return (
    <Card className="bg-white dark:bg-[#1a2744]/80 border border-slate-200 dark:border-white/10">
      <CardHeader className="flex justify-between items-center px-5 py-4 border-b border-slate-200 dark:border-white/10">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-cyan-400" />
          <span className="font-medium text-slate-800 dark:text-white">网络流量</span>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-4 text-xs">
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-gradient-to-r from-cyan-400 to-blue-400" />
              <span className="text-slate-500">入站</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-gradient-to-r from-violet-400 to-purple-400" />
              <span className="text-slate-500">出站</span>
            </span>
          </div>
          <Select
            size="sm"
            selectedKeys={[timeRange]}
            onChange={(e) => setTimeRange(e.target.value)}
            className="w-24"
            classNames={{
              trigger: "h-7 min-h-7 bg-slate-100 dark:bg-white/5",
              value: "text-xs",
            }}
          >
            <SelectItem key="1h">1小时</SelectItem>
            <SelectItem key="6h">6小时</SelectItem>
            <SelectItem key="24h">24小时</SelectItem>
            <SelectItem key="7d">7天</SelectItem>
          </Select>
        </div>
      </CardHeader>
      <CardBody className="p-5 h-64">
        {loading ? (
          <div className="w-full h-full flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-cyan-400"></div>
          </div>
        ) : (
          <div ref={chartRef} className="w-full h-full" />
        )}
      </CardBody>
    </Card>
  );
}
