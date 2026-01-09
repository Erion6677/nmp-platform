package marketplace

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RegistryPlugin 插件注册表中的插件信息
type RegistryPlugin struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Icon        string   `json:"icon"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	DownloadURL string   `json:"download_url"`
	Homepage    string   `json:"homepage"`
	License     string   `json:"license"`
	MinVersion  string   `json:"min_version"` // 最低 NMP 版本要求
	Size        int64    `json:"size"`
	Downloads   int      `json:"downloads"`
	UpdatedAt   string   `json:"updated_at"`
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	Version   string            `json:"version"`
	UpdatedAt string            `json:"updated_at"`
	Plugins   []*RegistryPlugin `json:"plugins"`
}

// InstalledPlugin 已安装插件信息
type InstalledPlugin struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Description    string `json:"description"`
	Author         string `json:"author"`
	Enabled        bool   `json:"enabled"`
	InstalledAt    string `json:"installed_at"`
	HasUpdate      bool   `json:"has_update"`
	LatestVersion  string `json:"latest_version,omitempty"`
}

// MarketplaceConfig 市场配置
type MarketplaceConfig struct {
	RegistryURL string // 插件注册表 URL
	PluginsDir  string // 本地插件目录
	CacheDir    string // 缓存目录
}

// Marketplace 插件市场服务
type Marketplace struct {
	config       *MarketplaceConfig
	logger       *zap.Logger
	registry     *PluginRegistry
	registryLock sync.RWMutex
	httpClient   *http.Client
}

// NewMarketplace 创建插件市场服务
func NewMarketplace(config *MarketplaceConfig, logger *zap.Logger) *Marketplace {
	// 确保目录存在
	os.MkdirAll(config.PluginsDir, 0755)
	os.MkdirAll(config.CacheDir, 0755)

	return &Marketplace{
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // 5分钟超时，GitHub下载可能较慢
		},
	}
}

// FetchRegistry 从远程或本地获取插件注册表
func (m *Marketplace) FetchRegistry() (*PluginRegistry, error) {
	m.logger.Info("Fetching plugin registry", zap.String("url", m.config.RegistryURL))

	var data []byte
	var err error

	// 支持本地文件 (file://) 和远程 URL (http/https)
	if strings.HasPrefix(m.config.RegistryURL, "file://") {
		// 本地文件
		filePath := strings.TrimPrefix(m.config.RegistryURL, "file://")
		data, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("读取本地注册表失败: %w", err)
		}
	} else {
		// 远程 URL
		resp, err := m.httpClient.Get(m.config.RegistryURL)
		if err != nil {
			return nil, fmt.Errorf("获取插件注册表失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("获取插件注册表失败: HTTP %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %w", err)
		}
	}

	var registry PluginRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("解析插件注册表失败: %w", err)
	}

	m.registryLock.Lock()
	m.registry = &registry
	m.registryLock.Unlock()

	m.logger.Info("Plugin registry fetched", zap.Int("plugins", len(registry.Plugins)))
	return &registry, nil
}

// GetRegistry 获取缓存的注册表
func (m *Marketplace) GetRegistry() *PluginRegistry {
	m.registryLock.RLock()
	defer m.registryLock.RUnlock()
	return m.registry
}

// GetAvailablePlugins 获取可用插件列表（带安装状态）
func (m *Marketplace) GetAvailablePlugins() ([]*RegistryPlugin, error) {
	registry, err := m.FetchRegistry()
	if err != nil {
		return nil, err
	}
	return registry.Plugins, nil
}

// GetInstalledPlugins 获取已安装的插件列表
func (m *Marketplace) GetInstalledPlugins() ([]*InstalledPlugin, error) {
	var installed []*InstalledPlugin

	entries, err := os.ReadDir(m.config.PluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return installed, nil
		}
		return nil, fmt.Errorf("读取插件目录失败: %w", err)
	}

	// 获取远程注册表用于版本对比
	m.registryLock.RLock()
	registry := m.registry
	m.registryLock.RUnlock()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(m.config.PluginsDir, entry.Name(), "plugin.json")
		data, err := os.ReadFile(pluginPath)
		if err != nil {
			continue
		}

		var manifest struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Author      string `json:"author"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		plugin := &InstalledPlugin{
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			Author:      manifest.Author,
			Enabled:     true, // TODO: 从配置读取
			InstalledAt: entry.Name(),
		}

		// 检查是否有更新
		if registry != nil {
			for _, rp := range registry.Plugins {
				if rp.Name == manifest.Name && rp.Version != manifest.Version {
					plugin.HasUpdate = true
					plugin.LatestVersion = rp.Version
					break
				}
			}
		}

		installed = append(installed, plugin)
	}

	return installed, nil
}

// InstallPlugin 安装插件
func (m *Marketplace) InstallPlugin(name string) error {
	// 从注册表获取插件信息
	m.registryLock.RLock()
	registry := m.registry
	m.registryLock.RUnlock()

	if registry == nil {
		// 尝试获取注册表
		var err error
		registry, err = m.FetchRegistry()
		if err != nil {
			return err
		}
	}

	var plugin *RegistryPlugin
	for _, p := range registry.Plugins {
		if p.Name == name {
			plugin = p
			break
		}
	}

	if plugin == nil {
		return fmt.Errorf("插件不存在: %s", name)
	}

	m.logger.Info("Installing plugin", zap.String("name", name), zap.String("url", plugin.DownloadURL))

	// 下载插件
	resp, err := m.httpClient.Get(plugin.DownloadURL)
	if err != nil {
		return fmt.Errorf("下载插件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载插件失败: HTTP %d", resp.StatusCode)
	}

	// 保存到临时文件
	tmpFile := filepath.Join(m.config.CacheDir, fmt.Sprintf("%s.tar.gz", name))
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("保存插件失败: %w", err)
	}

	// 解压到插件目录
	pluginDir := filepath.Join(m.config.PluginsDir, name)
	if err := m.extractTarGz(tmpFile, pluginDir); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("解压插件失败: %w", err)
	}

	// 清理临时文件
	os.Remove(tmpFile)

	m.logger.Info("Plugin installed successfully", zap.String("name", name))
	return nil
}

// UninstallPlugin 卸载插件
func (m *Marketplace) UninstallPlugin(name string) error {
	pluginDir := filepath.Join(m.config.PluginsDir, name)

	// 检查插件是否存在
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return fmt.Errorf("插件未安装: %s", name)
	}

	// 删除插件目录
	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("删除插件失败: %w", err)
	}

	m.logger.Info("Plugin uninstalled", zap.String("name", name))
	return nil
}

// UpdatePlugin 更新插件
func (m *Marketplace) UpdatePlugin(name string) error {
	// 先卸载再安装
	if err := m.UninstallPlugin(name); err != nil {
		// 如果插件不存在，直接安装
		if !strings.Contains(err.Error(), "未安装") {
			return err
		}
	}
	return m.InstallPlugin(name)
}

// extractTarGz 解压 tar.gz 文件（去掉第一层目录）
func (m *Marketplace) extractTarGz(src, dst string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 去掉第一层目录（如 system-backup/plugin.json -> plugin.json）
		name := header.Name
		parts := strings.SplitN(name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			// 跳过顶层目录本身
			continue
		}
		name = parts[1]

		// 安全检查：防止路径遍历攻击
		target := filepath.Join(dst, name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("非法路径: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// IsPluginInstalled 检查插件是否已安装
func (m *Marketplace) IsPluginInstalled(name string) bool {
	pluginPath := filepath.Join(m.config.PluginsDir, name, "plugin.json")
	_, err := os.Stat(pluginPath)
	return err == nil
}

// GetPluginInfo 获取插件详情
func (m *Marketplace) GetPluginInfo(name string) (*RegistryPlugin, error) {
	m.registryLock.RLock()
	registry := m.registry
	m.registryLock.RUnlock()

	if registry == nil {
		var err error
		registry, err = m.FetchRegistry()
		if err != nil {
			return nil, err
		}
	}

	for _, p := range registry.Plugins {
		if p.Name == name {
			return p, nil
		}
	}

	return nil, fmt.Errorf("插件不存在: %s", name)
}


// PluginMenu 插件菜单项
type PluginMenu struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Icon       string `json:"icon"`
	Path       string `json:"path"`
	Permission string `json:"permission,omitempty"`
	Order      int    `json:"order"`
}

// GetInstalledPluginMenus 获取已安装插件的菜单
func (m *Marketplace) GetInstalledPluginMenus() ([]PluginMenu, error) {
	var menus []PluginMenu

	entries, err := os.ReadDir(m.config.PluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return menus, nil
		}
		return nil, fmt.Errorf("读取插件目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(m.config.PluginsDir, entry.Name(), "plugin.json")
		data, err := os.ReadFile(pluginPath)
		if err != nil {
			continue
		}

		var manifest struct {
			Name  string `json:"name"`
			Menus []struct {
				Key        string `json:"key"`
				Label      string `json:"label"`
				Icon       string `json:"icon"`
				Path       string `json:"path"`
				Permission string `json:"permission"`
				Order      int    `json:"order"`
				Visible    bool   `json:"visible"`
			} `json:"menus"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		for _, menu := range manifest.Menus {
			if menu.Visible {
				menus = append(menus, PluginMenu{
					Key:        menu.Key,
					Label:      menu.Label,
					Icon:       menu.Icon,
					Path:       menu.Path,
					Permission: menu.Permission,
					Order:      menu.Order,
				})
			}
		}
	}

	return menus, nil
}
