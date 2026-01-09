package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-routeros/routeros/v3"
)

func main() {
	// 设备信息
	host := "45.192.246.241"
	port := 8827
	username := "admin"
	password := "Aa112211"

	// 连接到 RouterOS
	address := fmt.Sprintf("%s:%d", host, port)
	client, err := routeros.DialTimeout(address, username, password, 30*time.Second)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Close()

	fmt.Println("=== 连接成功 ===")

	// 获取脚本列表
	reply, err := client.Run("/system/script/print")
	if err != nil {
		log.Fatalf("获取脚本失败: %v", err)
	}

	fmt.Println("\n=== 脚本列表 ===")
	for _, re := range reply.Re {
		name := re.Map["name"]
		fmt.Printf("\n--- 脚本: %s ---\n", name)
		if source, ok := re.Map["source"]; ok {
			fmt.Printf("内容:\n%s\n", source)
		}
	}

	// 获取调度器
	reply, err = client.Run("/system/scheduler/print")
	if err != nil {
		log.Fatalf("获取调度器失败: %v", err)
	}

	fmt.Println("\n=== 调度器列表 ===")
	for _, re := range reply.Re {
		name := re.Map["name"]
		interval := re.Map["interval"]
		onEvent := re.Map["on-event"]
		disabled := re.Map["disabled"]
		fmt.Printf("名称: %s, 间隔: %s, 事件: %s, 禁用: %s\n", name, interval, onEvent, disabled)
	}

	// 获取脚本任务
	reply, err = client.Run("/system/script/job/print")
	if err != nil {
		log.Printf("获取脚本任务失败: %v", err)
	} else {
		fmt.Println("\n=== 运行中的脚本任务 ===")
		for _, re := range reply.Re {
			script := re.Map["script"]
			started := re.Map["started"]
			fmt.Printf("脚本: %s, 开始时间: %s\n", script, started)
		}
	}
}
