package main

import (
	"fmt"
	"nmp-platform/internal/collector"
)

func main() {
	gen := collector.NewScriptGenerator("http://10.10.10.231:8080")
	config := &collector.ScriptConfig{
		DeviceID:      3,
		DeviceIP:      "10.10.10.254",
		IntervalMs:    1000,
		PushBatchSize: 10,
		Interfaces:    []string{"bridge1", "pppoe-CT"},
		PingTargets: []collector.PingTargetConfig{
			{TargetAddress: "223.5.5.5", SourceInterface: "pppoe-CM"},
			{TargetAddress: "223.5.5.5", SourceInterface: "pppoe-CT"},
		},
	}
	script := gen.GenerateMikroTikScript(config)
	// 打印带行号
	lines := []byte(script)
	lineNum := 1
	fmt.Printf("%3d: ", lineNum)
	for _, c := range lines {
		fmt.Printf("%c", c)
		if c == '\n' {
			lineNum++
			fmt.Printf("%3d: ", lineNum)
		}
	}
}
