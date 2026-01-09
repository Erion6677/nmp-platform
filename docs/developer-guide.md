# NMP Platform 开发者文档

## 目录
- [项目概述](#项目概述)
- [开发环境搭建](#开发环境搭建)
- [项目结构](#项目结构)
- [后端开发](#后端开发)
- [前端开发](#前端开发)
- [插件开发](#插件开发)
- [API文档](#api文档)
- [测试指南](#测试指南)
- [部署指南](#部署指南)
- [贡献指南](#贡献指南)

## 项目概述

NMP Platform是一个基于Go和Vue.js的现代化网络监控平台，采用插件化架构设计，支持大规模设备监控。

### 技术栈

**后端**
- Go 1.21+
- Gin Web框架
- GORM ORM
- Casbin权限管理
- PostgreSQL数据库
- Redis缓存
- InfluxDB时序数据库

**前端**
- Vue 3 + TypeScript
- Vite构建工具
- NaiveUI组件库
- Pinia状态管理
- Vue Router路由
- ECharts图表库

**开发工具**
- Air热重载
- ESLint代码检查
- Prettier代码格式化
- Husky Git钩子

## 开发环境搭建

### 系统要求

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Redis 7+
- InfluxDB 2.7+
- Git

### 快速开始

1. **克隆项目**
```bash
git clone https://github.com/your-org/nmp-platform.git
cd nmp-platform
```

2. **运行开发环境脚本**
```bash
# 一键搭建开发环境
bash scripts/dev-setup.sh
```

3. **手动搭建（可选）**

   **后端环境**
   ```bash
   cd backend
   
   # 安装依赖
   go mod download
   
   # 复制配置文件
   cp configs/config.yaml.example configs/config.yaml
   
   # 运行数据库迁移
   make migrate
   
   # 启动开发服务器
   make dev
   ```

   **前端环境**
   ```bash
   cd frontend
   
   # 安装依赖
   npm install
   
   # 启动开发服务器
   npm run dev
   ```

### 开发工具配置

**VS Code推荐插件**
- Go (官方Go插件)
- Vetur (Vue开发)
- ESLint (代码检查)
- Prettier (代码格式化)
- GitLens (Git增强)

**配置文件**
```json
// .vscode/settings.json
{
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "editor.formatOnSave": true,
  "eslint.validate": ["javascript", "typescript", "vue"]
}
```

## 项目结构

```
nmp-platform/
├── backend/                 # 后端代码
│   ├── cmd/                # 命令行入口
│   │   └── server/         # 服务器启动代码
│   ├── internal/           # 内部包
│   │   ├── api/           # API处理器
│   │   ├── auth/          # 认证服务
│   │   ├── config/        # 配置管理
│   │   ├── database/      # 数据库操作
│   │   ├── middleware/    # 中间件
│   │   ├── models/        # 数据模型
│   │   ├── plugin/        # 插件系统
│   │   └── services/      # 业务服务
│   ├── plugins/           # 插件目录
│   ├── configs/           # 配置文件
│   ├── tests/             # 测试文件
│   └── Makefile          # 构建脚本
├── frontend/              # 前端代码
│   ├── src/              # 源代码
│   │   ├── components/   # 组件
│   │   ├── views/        # 页面
│   │   ├── stores/       # 状态管理
│   │   ├── router/       # 路由配置
│   │   ├── api/          # API调用
│   │   └── utils/        # 工具函数
│   ├── public/           # 静态资源
│   └── package.json      # 依赖配置
├── deployments/          # 部署文件
├── docs/                 # 文档
└── scripts/              # 脚本文件
```

## 后端开发

### 代码规范

**包命名**
- 使用小写字母
- 避免下划线和驼峰
- 包名应该简洁明了

**函数命名**
- 公开函数使用大写字母开头
- 私有函数使用小写字母开头
- 使用驼峰命名法

**错误处理**
```go
// 好的错误处理
func GetUser(id uint) (*User, error) {
    var user User
    if err := db.First(&user, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrUserNotFound
        }
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return &user, nil
}
```

### 数据库操作

**模型定义**
```go
type Device struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Name      string    `gorm:"not null" json:"name"`
    Host      string    `gorm:"not null" json:"host"`
    Type      string    `gorm:"not null" json:"type"`
    Status    string    `gorm:"default:offline" json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

**数据库迁移**
```go
// migrations/001_create_devices.go
func CreateDevicesTable(db *gorm.DB) error {
    return db.AutoMigrate(&Device{})
}
```

**仓储模式**
```go
type DeviceRepository interface {
    Create(device *Device) error
    GetByID(id uint) (*Device, error)
    Update(device *Device) error
    Delete(id uint) error
    List(offset, limit int) ([]*Device, error)
}

type deviceRepository struct {
    db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) DeviceRepository {
    return &deviceRepository{db: db}
}
```

### API开发

**路由定义**
```go
func SetupRoutes(r *gin.Engine) {
    api := r.Group("/api/v1")
    {
        devices := api.Group("/devices")
        {
            devices.GET("", deviceHandler.List)
            devices.POST("", deviceHandler.Create)
            devices.GET("/:id", deviceHandler.GetByID)
            devices.PUT("/:id", deviceHandler.Update)
            devices.DELETE("/:id", deviceHandler.Delete)
        }
    }
}
```

**处理器实现**
```go
type DeviceHandler struct {
    service DeviceService
}

func (h *DeviceHandler) Create(c *gin.Context) {
    var req CreateDeviceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    device, err := h.service.Create(&req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, device)
}
```

### 中间件开发

**认证中间件**
```go
func AuthMiddleware(authService AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "missing token"})
            c.Abort()
            return
        }
        
        claims, err := authService.ValidateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }
        
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

### 配置管理

**配置结构**
```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    InfluxDB InfluxConfig   `yaml:"influxdb"`
}

func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

## 前端开发

### 项目结构

```
src/
├── api/              # API接口
├── components/       # 通用组件
├── composables/      # 组合式函数
├── layouts/          # 布局组件
├── router/           # 路由配置
├── stores/           # 状态管理
├── styles/           # 样式文件
├── types/            # TypeScript类型
├── utils/            # 工具函数
└── views/            # 页面组件
```

### 组件开发

**基础组件**
```vue
<template>
  <div class="device-card">
    <n-card :title="device.name">
      <template #header-extra>
        <n-tag :type="statusType">{{ device.status }}</n-tag>
      </template>
      
      <n-descriptions :column="2">
        <n-descriptions-item label="IP地址">
          {{ device.host }}
        </n-descriptions-item>
        <n-descriptions-item label="类型">
          {{ device.type }}
        </n-descriptions-item>
      </n-descriptions>
      
      <template #action>
        <n-space>
          <n-button @click="$emit('edit', device)">编辑</n-button>
          <n-button @click="$emit('delete', device)" type="error">删除</n-button>
        </n-space>
      </template>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Device } from '@/types/device'

interface Props {
  device: Device
}

const props = defineProps<Props>()

const emit = defineEmits<{
  edit: [device: Device]
  delete: [device: Device]
}>()

const statusType = computed(() => {
  return props.device.status === 'online' ? 'success' : 'error'
})
</script>
```

### 状态管理

**Pinia Store**
```typescript
// stores/device.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Device } from '@/types/device'
import { deviceApi } from '@/api/device'

export const useDeviceStore = defineStore('device', () => {
  const devices = ref<Device[]>([])
  const loading = ref(false)
  
  const onlineDevices = computed(() => 
    devices.value.filter(d => d.status === 'online')
  )
  
  async function fetchDevices() {
    loading.value = true
    try {
      const response = await deviceApi.list()
      devices.value = response.data
    } finally {
      loading.value = false
    }
  }
  
  async function createDevice(device: CreateDeviceRequest) {
    const response = await deviceApi.create(device)
    devices.value.push(response.data)
    return response.data
  }
  
  return {
    devices,
    loading,
    onlineDevices,
    fetchDevices,
    createDevice
  }
})
```

### API封装

**HTTP客户端**
```typescript
// api/http.ts
import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 10000
})

// 请求拦截器
http.interceptors.request.use(config => {
  const authStore = useAuthStore()
  if (authStore.token) {
    config.headers.Authorization = `Bearer ${authStore.token}`
  }
  return config
})

// 响应拦截器
http.interceptors.response.use(
  response => response,
  error => {
    if (error.response?.status === 401) {
      const authStore = useAuthStore()
      authStore.logout()
    }
    return Promise.reject(error)
  }
)

export default http
```

**API接口**
```typescript
// api/device.ts
import http from './http'
import type { Device, CreateDeviceRequest } from '@/types/device'

export const deviceApi = {
  list: () => http.get<Device[]>('/devices'),
  
  getById: (id: number) => http.get<Device>(`/devices/${id}`),
  
  create: (data: CreateDeviceRequest) => 
    http.post<Device>('/devices', data),
  
  update: (id: number, data: Partial<Device>) => 
    http.put<Device>(`/devices/${id}`, data),
  
  delete: (id: number) => http.delete(`/devices/${id}`)
}
```

### 路由配置

```typescript
// router/index.ts
import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue')
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue')
      },
      {
        path: 'devices',
        name: 'Devices',
        component: () => import('@/views/Devices.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  
  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next('/login')
  } else {
    next()
  }
})

export default router
```

## 插件开发

### 插件架构

NMP Platform采用插件化架构，支持功能模块的动态加载和管理。

### 插件接口

```go
// internal/plugin/interface.go
type Plugin interface {
    // 基础信息
    Name() string
    Version() string
    Description() string
    Dependencies() []string
    
    // 生命周期
    Initialize(ctx context.Context, config interface{}) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() error
    
    // 功能接口
    GetRoutes() []Route
    GetMenus() []MenuItem
    GetPermissions() []Permission
    GetConfigSchema() interface{}
}
```

### 插件示例

**监控插件**
```go
// plugins/monitoring/plugin.go
package monitoring

import (
    "context"
    "github.com/gin-gonic/gin"
    "nmp-platform/internal/plugin"
)

type MonitoringPlugin struct {
    config *Config
    service *MonitoringService
}

func NewMonitoringPlugin() plugin.Plugin {
    return &MonitoringPlugin{}
}

func (p *MonitoringPlugin) Name() string {
    return "monitoring"
}

func (p *MonitoringPlugin) Version() string {
    return "1.0.0"
}

func (p *MonitoringPlugin) Description() string {
    return "设备监控插件"
}

func (p *MonitoringPlugin) Initialize(ctx context.Context, config interface{}) error {
    cfg, ok := config.(*Config)
    if !ok {
        return errors.New("invalid config type")
    }
    
    p.config = cfg
    p.service = NewMonitoringService(cfg)
    return nil
}

func (p *MonitoringPlugin) GetRoutes() []plugin.Route {
    return []plugin.Route{
        {
            Method: "GET",
            Path: "/monitoring/devices",
            Handler: p.handleGetDevices,
        },
        {
            Method: "POST",
            Path: "/monitoring/data",
            Handler: p.handlePushData,
        },
    }
}

func (p *MonitoringPlugin) GetMenus() []plugin.MenuItem {
    return []plugin.MenuItem{
        {
            Key: "monitoring",
            Label: "监控中心",
            Icon: "monitor",
            Path: "/monitoring",
            Children: []plugin.MenuItem{
                {
                    Key: "realtime",
                    Label: "实时监控",
                    Path: "/monitoring/realtime",
                },
            },
        },
    }
}
```

### 前端插件

**插件注册**
```typescript
// plugins/monitoring/index.ts
import type { PluginModule } from '@/types/plugin'

const monitoringPlugin: PluginModule = {
  name: 'monitoring',
  version: '1.0.0',
  
  install(app) {
    // 注册组件
    app.component('MonitoringChart', () => import('./components/MonitoringChart.vue'))
  },
  
  routes: [
    {
      path: '/monitoring',
      component: () => import('./views/Monitoring.vue'),
      children: [
        {
          path: 'realtime',
          component: () => import('./views/Realtime.vue')
        }
      ]
    }
  ],
  
  menus: [
    {
      key: 'monitoring',
      label: '监控中心',
      icon: 'monitor',
      path: '/monitoring'
    }
  ]
}

export default monitoringPlugin
```

## API文档

### RESTful API设计

**URL设计**
- 使用名词而不是动词
- 使用复数形式
- 使用层级结构表示资源关系

```
GET    /api/v1/devices          # 获取设备列表
POST   /api/v1/devices          # 创建设备
GET    /api/v1/devices/:id      # 获取设备详情
PUT    /api/v1/devices/:id      # 更新设备
DELETE /api/v1/devices/:id      # 删除设备

GET    /api/v1/devices/:id/interfaces  # 获取设备接口
POST   /api/v1/devices/:id/test        # 测试设备连接
```

**HTTP状态码**
- 200: 成功
- 201: 创建成功
- 400: 请求错误
- 401: 未认证
- 403: 无权限
- 404: 资源不存在
- 500: 服务器错误

**响应格式**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "name": "Router-01",
    "host": "192.168.1.1"
  }
}
```

### API文档生成

使用Swagger生成API文档：

```go
// @title NMP Platform API
// @version 1.0
// @description 网络监控平台API文档
// @host localhost:8080
// @BasePath /api/v1

// @Summary 获取设备列表
// @Description 获取所有设备的列表
// @Tags devices
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Success 200 {array} Device
// @Router /devices [get]
func (h *DeviceHandler) List(c *gin.Context) {
    // 实现代码
}
```

## 测试指南

### 单元测试

**后端测试**
```go
// internal/services/device_test.go
func TestDeviceService_Create(t *testing.T) {
    // 设置测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    repo := NewDeviceRepository(db)
    service := NewDeviceService(repo)
    
    // 测试用例
    tests := []struct {
        name    string
        request *CreateDeviceRequest
        wantErr bool
    }{
        {
            name: "valid device",
            request: &CreateDeviceRequest{
                Name: "Test Device",
                Host: "192.168.1.1",
                Type: "router",
            },
            wantErr: false,
        },
        {
            name: "invalid host",
            request: &CreateDeviceRequest{
                Name: "Test Device",
                Host: "invalid-host",
                Type: "router",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            device, err := service.Create(tt.request)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, device)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, device)
                assert.Equal(t, tt.request.Name, device.Name)
            }
        })
    }
}
```

**前端测试**
```typescript
// tests/components/DeviceCard.test.ts
import { mount } from '@vue/test-utils'
import DeviceCard from '@/components/DeviceCard.vue'

describe('DeviceCard', () => {
  const mockDevice = {
    id: 1,
    name: 'Test Device',
    host: '192.168.1.1',
    status: 'online'
  }
  
  it('renders device information correctly', () => {
    const wrapper = mount(DeviceCard, {
      props: { device: mockDevice }
    })
    
    expect(wrapper.text()).toContain('Test Device')
    expect(wrapper.text()).toContain('192.168.1.1')
  })
  
  it('emits edit event when edit button clicked', async () => {
    const wrapper = mount(DeviceCard, {
      props: { device: mockDevice }
    })
    
    await wrapper.find('[data-test="edit-button"]').trigger('click')
    
    expect(wrapper.emitted('edit')).toBeTruthy()
    expect(wrapper.emitted('edit')[0]).toEqual([mockDevice])
  })
})
```

### 集成测试

```go
// tests/integration/device_api_test.go
func TestDeviceAPI(t *testing.T) {
    // 设置测试服务器
    router := setupTestRouter(t)
    server := httptest.NewServer(router)
    defer server.Close()
    
    client := &http.Client{}
    
    t.Run("create device", func(t *testing.T) {
        payload := `{"name":"Test Device","host":"192.168.1.1","type":"router"}`
        req, _ := http.NewRequest("POST", server.URL+"/api/v1/devices", strings.NewReader(payload))
        req.Header.Set("Content-Type", "application/json")
        
        resp, err := client.Do(req)
        assert.NoError(t, err)
        assert.Equal(t, http.StatusCreated, resp.StatusCode)
        
        var device Device
        json.NewDecoder(resp.Body).Decode(&device)
        assert.Equal(t, "Test Device", device.Name)
    })
}
```

### 性能测试

```go
// tests/performance/load_test.go
func BenchmarkDeviceList(b *testing.B) {
    router := setupBenchmarkRouter(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        req := httptest.NewRequest("GET", "/api/v1/devices", nil)
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        if w.Code != http.StatusOK {
            b.Fatalf("Expected status 200, got %d", w.Code)
        }
    }
}
```

## 部署指南

### 构建

**后端构建**
```bash
cd backend
make build
```

**前端构建**
```bash
cd frontend
npm run build
```

### 生产部署

参考 [部署指南](DEPLOYMENT.md) 进行生产环境部署。

## 贡献指南

### 开发流程

1. Fork项目到个人仓库
2. 创建功能分支：`git checkout -b feature/new-feature`
3. 提交代码：`git commit -am 'Add new feature'`
4. 推送分支：`git push origin feature/new-feature`
5. 创建Pull Request

### 代码规范

**提交信息格式**
```
type(scope): description

[optional body]

[optional footer]
```

类型：
- feat: 新功能
- fix: 修复bug
- docs: 文档更新
- style: 代码格式化
- refactor: 重构
- test: 测试相关
- chore: 构建过程或辅助工具的变动

**示例**
```
feat(device): add device monitoring functionality

- Add real-time monitoring for network devices
- Implement data collection from RouterOS devices
- Add monitoring dashboard with charts

Closes #123
```

### 代码审查

Pull Request需要通过以下检查：
1. 代码格式化检查
2. 单元测试通过
3. 集成测试通过
4. 代码审查通过
5. 文档更新

### 发布流程

1. 更新版本号
2. 更新CHANGELOG
3. 创建Release Tag
4. 构建发布包
5. 发布到GitHub Releases

## 常见问题

### 开发环境问题

**Q: Go模块下载失败**
```bash
# 设置Go代理
go env -w GOPROXY=https://goproxy.cn,direct
```

**Q: 前端依赖安装失败**
```bash
# 清除缓存重新安装
rm -rf node_modules package-lock.json
npm install
```

**Q: 数据库连接失败**
```bash
# 检查数据库服务状态
systemctl status postgresql

# 检查配置文件
cat configs/config.yaml
```

### 构建问题

**Q: 后端构建失败**
```bash
# 清理构建缓存
go clean -cache
go mod tidy
make build
```

**Q: 前端构建失败**
```bash
# 检查Node.js版本
node --version

# 重新安装依赖
npm ci
npm run build
```

## 参考资源

- [Go官方文档](https://golang.org/doc/)
- [Vue.js官方文档](https://vuejs.org/)
- [Gin框架文档](https://gin-gonic.com/)
- [NaiveUI组件库](https://www.naiveui.com/)
- [PostgreSQL文档](https://www.postgresql.org/docs/)
- [InfluxDB文档](https://docs.influxdata.com/)

## 联系方式

- 项目仓库: https://github.com/your-org/nmp-platform
- 问题反馈: https://github.com/your-org/nmp-platform/issues
- 技术讨论: https://github.com/your-org/nmp-platform/discussions