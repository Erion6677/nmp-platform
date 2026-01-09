// +build integration

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nmp-platform/internal/collector"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 真实测试设备配置
var TestMikroTikDevice = struct {
	IP       string
	APIPort  int
	SSHPort  int
	Username string
	Password string
}{
	IP:       "10.10.10.254",
	APIPort:  8827,
	SSHPort:  3399,
	Username: "test",
	Password: "tset@123",
}

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(
		&models.Device{},
		&models.Interface{},
		&models.Tag{},
		&models.DeviceGroup{},
		&models.DeviceTag{},
		&models.DeviceGroupMember{},
	)
	require.NoError(t, err)

	return db
}

// setupTestHandler 创建测试处理器
func setupTestHandler(t *testing.T, db *gorm.DB) *DeviceHandler {
	deviceRepo := repository.NewDeviceRepository(db)
	interfaceRepo := repository.NewInterfaceRepository(db)
	tagRepo := repository.NewTagRepository(db)
	groupRepo := repository.NewDeviceGroupRepository(db)

	deviceService := service.NewDeviceService(deviceRepo, interfaceRepo, tagRepo, groupRepo)

	return NewDeviceHandler(deviceService, nil, nil)
}

// TestIntegration_TestConnectionWithRealDevice 使用真实设备测试连接
// 此测试需要真实的 MikroTik 设备
func TestIntegration_TestConnectionWithRealDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	gin.SetMode(gin.TestMode)

	// 先直接测试采集器连接
	rosCollector := collector.NewRouterOSCollector(10 * time.Second)
	sshCollector := collector.NewSSHCollector(10 * time.Second)

	// 测试 API 连接
	t.Run("API连接测试", func(t *testing.T) {
		err := rosCollector.TestConnection(
			TestMikroTikDevice.IP,
			TestMikroTikDevice.APIPort,
			TestMikroTikDevice.Username,
			TestMikroTikDevice.Password,
		)
		assert.NoError(t, err, "API 连接测试应该成功")
	})

	// 测试 SSH 连接
	t.Run("SSH连接测试", func(t *testing.T) {
		err := sshCollector.TestConnection(
			TestMikroTikDevice.IP,
			TestMikroTikDevice.SSHPort,
			TestMikroTikDevice.Username,
			TestMikroTikDevice.Password,
		)
		assert.NoError(t, err, "SSH 连接测试应该成功")
	})
}

// TestIntegration_GetSystemInfoWithRealDevice 使用真实设备测试获取系统信息
func TestIntegration_GetSystemInfoWithRealDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	gin.SetMode(gin.TestMode)

	// 测试通过 API 获取系统信息
	t.Run("通过API获取系统信息", func(t *testing.T) {
		rosCollector := collector.NewRouterOSCollector(10 * time.Second)
		
		client, err := rosCollector.Connect(
			TestMikroTikDevice.IP,
			TestMikroTikDevice.APIPort,
			TestMikroTikDevice.Username,
			TestMikroTikDevice.Password,
		)
		require.NoError(t, err, "连接应该成功")
		defer client.Close()

		info, err := rosCollector.GetSystemInfo(client)
		require.NoError(t, err, "获取系统信息应该成功")

		// 验证系统信息完整性 (Property 6)
		assert.NotEmpty(t, info.DeviceName, "设备名称不应为空")
		assert.Greater(t, info.CPUCount, 0, "CPU核心数应大于0")
		assert.NotEmpty(t, info.Version, "系统版本不应为空")
		assert.GreaterOrEqual(t, info.Uptime, int64(0), "运行时间应大于等于0")
		assert.GreaterOrEqual(t, info.CPUUsage, float64(0), "CPU使用率应大于等于0")
		assert.GreaterOrEqual(t, info.MemoryUsage, float64(0), "内存使用率应大于等于0")
		assert.Greater(t, info.MemoryTotal, int64(0), "总内存应大于0")
		assert.GreaterOrEqual(t, info.MemoryFree, int64(0), "可用内存应大于等于0")

		t.Logf("设备名称: %s", info.DeviceName)
		t.Logf("CPU核心数: %d", info.CPUCount)
		t.Logf("系统版本: %s", info.Version)
		t.Logf("授权等级: %s", info.License)
		t.Logf("运行时间: %d秒", info.Uptime)
		t.Logf("CPU使用率: %.2f%%", info.CPUUsage)
		t.Logf("内存使用率: %.2f%%", info.MemoryUsage)
	})

	// 测试通过 SSH 获取系统信息
	t.Run("通过SSH获取系统信息", func(t *testing.T) {
		sshCollector := collector.NewSSHCollector(10 * time.Second)
		
		client, err := sshCollector.Connect(
			TestMikroTikDevice.IP,
			TestMikroTikDevice.SSHPort,
			TestMikroTikDevice.Username,
			TestMikroTikDevice.Password,
		)
		require.NoError(t, err, "SSH连接应该成功")
		defer client.Close()

		info, err := sshCollector.GetMikroTikSystemInfo(client)
		require.NoError(t, err, "获取系统信息应该成功")

		// 验证系统信息完整性
		assert.NotEmpty(t, info.DeviceName, "设备名称不应为空")
		assert.Greater(t, info.CPUCount, 0, "CPU核心数应大于0")
		assert.NotEmpty(t, info.Version, "系统版本不应为空")

		t.Logf("SSH - 设备名称: %s", info.DeviceName)
		t.Logf("SSH - CPU核心数: %d", info.CPUCount)
		t.Logf("SSH - 系统版本: %s", info.Version)
	})
}

// TestIntegration_SyncInterfacesWithRealDevice 使用真实设备测试同步接口
func TestIntegration_SyncInterfacesWithRealDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	gin.SetMode(gin.TestMode)

	// 测试通过 API 获取接口列表
	t.Run("通过API获取接口列表", func(t *testing.T) {
		rosCollector := collector.NewRouterOSCollector(10 * time.Second)
		
		client, err := rosCollector.Connect(
			TestMikroTikDevice.IP,
			TestMikroTikDevice.APIPort,
			TestMikroTikDevice.Username,
			TestMikroTikDevice.Password,
		)
		require.NoError(t, err, "连接应该成功")
		defer client.Close()

		interfaces, err := rosCollector.GetInterfaces(client)
		require.NoError(t, err, "获取接口列表应该成功")
		require.NotEmpty(t, interfaces, "接口列表不应为空")

		// 验证接口数据只包含名称和状态 (Property 2)
		for _, iface := range interfaces {
			assert.NotEmpty(t, iface.Name, "接口名称不应为空")
			assert.Contains(t, []string{"up", "down"}, iface.Status, "接口状态应为 up 或 down")
		}

		t.Logf("获取到 %d 个接口", len(interfaces))
		for _, iface := range interfaces {
			t.Logf("  - %s: %s", iface.Name, iface.Status)
		}
	})

	// 测试通过 SSH 获取接口列表
	t.Run("通过SSH获取接口列表", func(t *testing.T) {
		sshCollector := collector.NewSSHCollector(10 * time.Second)
		
		client, err := sshCollector.Connect(
			TestMikroTikDevice.IP,
			TestMikroTikDevice.SSHPort,
			TestMikroTikDevice.Username,
			TestMikroTikDevice.Password,
		)
		require.NoError(t, err, "SSH连接应该成功")
		defer client.Close()

		interfaces, err := sshCollector.GetMikroTikInterfaces(client)
		require.NoError(t, err, "获取接口列表应该成功")
		require.NotEmpty(t, interfaces, "接口列表不应为空")

		// 验证接口数据只包含名称和状态
		for _, iface := range interfaces {
			assert.NotEmpty(t, iface.Name, "接口名称不应为空")
			assert.Contains(t, []string{"up", "down"}, iface.Status, "接口状态应为 up 或 down")
		}

		t.Logf("SSH - 获取到 %d 个接口", len(interfaces))
	})
}

// TestIntegration_DeviceAPIWithRealDevice 使用真实设备测试完整 API 流程
func TestIntegration_DeviceAPIWithRealDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	gin.SetMode(gin.TestMode)

	// 设置测试数据库和处理器
	db := setupTestDB(t)
	handler := setupTestHandler(t, db)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 1. 创建设备
	t.Run("创建设备", func(t *testing.T) {
		createReq := CreateDeviceRequest{
			Name:     "Test MikroTik",
			Type:     models.DeviceTypeRouter,
			OSType:   models.DeviceOSTypeMikroTik,
			Host:     TestMikroTikDevice.IP,
			Port:     TestMikroTikDevice.SSHPort,
			APIPort:  TestMikroTikDevice.APIPort,
			Username: TestMikroTikDevice.Username,
			Password: TestMikroTikDevice.Password,
		}

		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/devices", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
	})

	// 2. 测试连接
	t.Run("测试连接", func(t *testing.T) {
		testReq := TestConnectionRequest{
			Host:           TestMikroTikDevice.IP,
			Port:           TestMikroTikDevice.SSHPort,
			APIPort:        TestMikroTikDevice.APIPort,
			Username:       TestMikroTikDevice.Username,
			Password:       TestMikroTikDevice.Password,
			OSType:         models.DeviceOSTypeMikroTik,
			ConnectionType: "all",
		}

		reqBody, _ := json.Marshal(testReq)
		req, _ := http.NewRequest("POST", "/api/devices/test", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		assert.True(t, data["api_success"].(bool), "API 连接应该成功")
		assert.True(t, data["ssh_success"].(bool), "SSH 连接应该成功")
	})

	// 3. 获取系统信息
	t.Run("获取系统信息", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/devices/1/info", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["device_name"])
		assert.NotEmpty(t, data["version"])
	})

	// 4. 同步接口
	t.Run("同步接口", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/devices/1/interfaces/sync", nil)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].([]interface{})
		assert.NotEmpty(t, data, "应该有接口数据")
	})

	// 5. 设置监控接口
	t.Run("设置监控接口", func(t *testing.T) {
		// 先获取接口列表
		req, _ := http.NewRequest("GET", "/api/devices/1/interfaces", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var listResponse map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &listResponse)
		interfaces := listResponse["data"].([]interface{})

		if len(interfaces) > 0 {
			// 选择第一个接口进行监控
			firstInterface := interfaces[0].(map[string]interface{})
			interfaceName := firstInterface["name"].(string)

			setReq := map[string][]string{"interface_names": {interfaceName}}
			reqBody, _ := json.Marshal(setReq)
			req, _ := http.NewRequest("PUT", "/api/devices/1/interfaces/monitored", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.True(t, response["success"].(bool))
		}
	})
}
