package main

import (
	"flag"
	"fmt"
	"log"
	"nmp-platform/internal/config"
	"nmp-platform/internal/database"
	"os"

	"gorm.io/gorm"
)

func main() {
	var (
		seedData   = flag.Bool("seed", false, "是否初始化基础数据")
		dropTables = flag.Bool("drop", false, "是否删除所有表（危险操作）")
	)
	flag.Parse()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// 如果指定了删除表，先删除所有表
	if *dropTables {
		fmt.Print("警告：这将删除所有数据表和数据！确认继续吗？(y/N): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("操作已取消")
			os.Exit(0)
		}

		if err := dropAllTables(db); err != nil {
			log.Fatalf("Failed to drop tables: %v", err)
		}
		fmt.Println("所有表已删除")
	}

	// 执行迁移
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 如果指定了初始化数据，执行数据初始化
	if *seedData {
		if err := database.SeedData(db); err != nil {
			log.Fatalf("Failed to seed data: %v", err)
		}
	}

	fmt.Println("数据库迁移完成！")
}

// dropAllTables 删除所有表
func dropAllTables(db *gorm.DB) error {
	// 获取所有表名
	var tables []string
	if err := db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables).Error; err != nil {
		return err
	}

	// 删除所有表
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			return err
		}
	}

	return nil
}