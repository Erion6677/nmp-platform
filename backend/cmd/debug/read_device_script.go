package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-routeros/routeros/v3"
)

func main() {
	// 连接设备
	client, err := routeros.DialTimeout("10.10.10.254:8827", "admin", "927528", 30*time.Second)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer client.Close()

	// 读取脚本内容
	reply, err := client.Run("/system/script/print", "?name=nmp-collector")
	if err != nil {
		log.Fatal("读取脚本失败:", err)
	}

	if len(reply.Re) == 0 {
		fmt.Println("脚本不存在")
		return
	}

	// 打印脚本源码
	source := reply.Re[0].Map["source"]
	fmt.Println("=== 设备上的脚本内容 ===")
	fmt.Println(source)
	fmt.Println()
	fmt.Println("=== 脚本长度 ===")
	fmt.Println(len(source))
	fmt.Println()
	
	// 打印前200个字符的十六进制
	fmt.Println("=== 前200字符的十六进制 ===")
	if len(source) > 200 {
		source = source[:200]
	}
	for i, c := range source {
		fmt.Printf("%d: %c (0x%02x)\n", i, c, c)
	}
}
