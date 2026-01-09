package main

import (
	"fmt"
	"nmp-platform/internal/collector"
)

func main() {
	gen := collector.NewScriptGenerator("http://45.41.53.254:8080")
	
	config := &collector.ScriptConfig{
		DeviceID:      3,
		DeviceIP:      "45.192.246.241",
		ServerURL:     "http://45.41.53.254:8080",
		IntervalMs:    1000,
		PushBatchSize: 60,
		MaxQueueSize:  300,
		ScriptName:    "nmp-collector",
		LauncherName:  "nmp-collector_launcher",
		SchedulerName: "nmp-scheduler",
		Interfaces:    []string{"bridge1"},
		PingTargets: []collector.PingTargetConfig{
			{TargetAddress: "223.5.5.5", SourceInterface: "l2tp-hkg101"},
			{TargetAddress: "223.5.5.5", SourceInterface: "l2tp-hkg201"},
			{TargetAddress: "8.8.8.8", SourceInterface: "ether1"},
		},
	}
	
	script := gen.GenerateMikroTikScript(config)
	fmt.Println("=== 主脚本 ===")
	fmt.Println(script)
	
	launcher := gen.GenerateMikroTikLauncher(config)
	fmt.Println("\n=== 启动器脚本 ===")
	fmt.Println(launcher)
}
