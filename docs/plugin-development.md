# NMP 插件开发指南

## 概述

NMP 平台支持通过插件扩展功能。插件可以添加新的 API 路由、菜单项和权限控制。

## 插件结构

```
plugins/
└── your-plugin/
    ├── plugin.json      # 插件清单文件（必需）
    ├── main.go          # Go 插件入口（可选）
    ├── assets/          # 前端静态资源（可选）
    │   ├── index.js
    │   └── style.css
    └── README.md        # 插件说明
```

## plugin.json 清单文件

```json
{
  "name": "your-plugin",
  "version": "1.0.0",
  "description": "插件描述",
  "author": "作者",
  "dependencies": [],
  "entry_point": "main.go",
  "permissions": [...],
  "routes": [...],
  "menus": [...],
  "config": {...},
  "frontend": {...}
}
```

### 权限定义

```json
{
  "permissions": [
    {
      "resource": "dashboard",
      "action": "read",
      "scope": "all",
      "description": "查看仪表板"
    }
  ]
}
```

权限会自动注册到 RBAC 系统，格式为 `plugin.{plugin-name}.{resource}:{action}`

### 路由定义

```json
{
  "routes": [
    {
      "method": "GET",
      "path": "/data",
      "handler": "default",
      "permission": "plugin.your-plugin.data:read",
      "description": "获取数据"
    }
  ]
}
```

路由会自动注册到 `/api/plugins/{plugin-name}/` 路径下。

### 菜单定义

```json
{
  "menus": [
    {
      "key": "your-plugin",
      "label": "插件名称",
      "icon": "icon-name",
      "path": "/plugins/your-plugin",
      "permission": "plugin.your-plugin.dashboard:read",
      "order": 100,
      "visible": true,
      "children": []
    }
  ]
}
```

菜单会根据用户权限自动过滤显示。

### 配置模式

```json
{
  "config": {
    "type": "object",
    "required": ["enabled"],
    "properties": {
      "enabled": {
        "type": "boolean"
      },
      "interval": {
        "type": "integer",
        "minimum": 1,
        "maximum": 3600
      }
    }
  }
}
```

支持 JSON Schema 格式的配置验证。

## 权限系统

### 权限检查流程

1. 用户请求插件 API
2. 系统从 JWT 中获取用户 ID
3. 查询用户角色
4. 检查角色是否有对应权限
5. 允许或拒绝访问

### 自动创建的角色

注册插件时，系统会自动创建 `plugin_{name}_user` 角色，包含该插件的所有权限。

### 为用户分配插件权限

```bash
# 通过 API 为用户分配插件角色
POST /api/admin/users/{user_id}/roles
{
  "role_name": "plugin_example-plugin_user"
}
```

## 处理器类型

### default - 默认处理器
返回插件和路由信息。

### script:{path} - 脚本处理器
执行指定脚本文件。

### http://{url} - HTTP 代理
代理请求到外部 HTTP 服务。

### static:{dir} - 静态文件
提供静态文件服务。

## 安装插件

1. 将插件目录放到 `plugins/` 下
2. 重启服务或调用热加载 API

```bash
POST /api/plugins/reload
```

## 示例

参考 `plugins/example-plugin/` 目录。
