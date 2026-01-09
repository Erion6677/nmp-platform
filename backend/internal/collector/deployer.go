// Package collector 提供设备数据采集功能
package collector

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-routeros/routeros/v3"
	"golang.org/x/crypto/ssh"
)

// DeployResult 部署结果
type DeployResult struct {
	Success      bool   `json:"success"`
	Method       string `json:"method"` // api, ssh
	Message      string `json:"message"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Deployer 脚本部署器
type Deployer struct {
	rosCollector *RouterOSCollector
	sshCollector *SSHCollector
	generator    *ScriptGenerator
}

// NewDeployer 创建新的部署器
func NewDeployer(serverURL string) *Deployer {
	return &Deployer{
		rosCollector: NewRouterOSCollector(30 * time.Second),
		sshCollector: NewSSHCollector(30 * time.Second),
		generator:    NewScriptGenerator(serverURL),
	}
}

// DeployToMikroTik 部署脚本到 MikroTik 设备
// 优先使用 API（更可靠），失败则尝试 SSH
func (d *Deployer) DeployToMikroTik(config *ScriptConfig, ip string, apiPort, sshPort int, username, password string) *DeployResult {
	// 先尝试 API 部署（更可靠，不需要复杂转义）
	result := d.deployViaAPI(config, ip, apiPort, username, password)
	if result.Success {
		return result
	}

	// API 失败，尝试 SSH 部署
	sshResult := d.deployViaSSH(config, ip, sshPort, username, password)
	if sshResult.Success {
		return sshResult
	}

	// 两种方式都失败
	return &DeployResult{
		Success:      false,
		Method:       "none",
		Message:      "部署失败",
		ErrorMessage: fmt.Sprintf("API 错误: %s; SSH 错误: %s", result.ErrorMessage, sshResult.ErrorMessage),
	}
}

// deployViaAPI 通过 RouterOS API 部署脚本
func (d *Deployer) deployViaAPI(config *ScriptConfig, ip string, port int, username, password string) *DeployResult {
	client, err := d.rosCollector.Connect(ip, port, username, password)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "api",
			ErrorMessage: err.Error(),
		}
	}
	defer client.Close()

	// 先移除旧的脚本和调度器
	d.removeScriptViaAPI(client, config.ScriptName, config.SchedulerName)

	// 生成脚本内容
	mainScript := d.generator.GenerateMikroTikScript(config)
	launcherScript := d.generator.GenerateMikroTikLauncher(config)

	// 添加主脚本
	_, err = client.Run("/system/script/add",
		fmt.Sprintf("=name=%s", config.ScriptName),
		fmt.Sprintf("=source=%s", mainScript),
		"=policy=read,write,test",
	)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "api",
			ErrorMessage: fmt.Sprintf("添加主脚本失败: %s", err.Error()),
		}
	}

	// 添加启动器脚本
	launcherName := config.ScriptName + "_launcher"
	if config.LauncherName != "" {
		launcherName = config.LauncherName
	}
	_, err = client.Run("/system/script/add",
		fmt.Sprintf("=name=%s", launcherName),
		fmt.Sprintf("=source=%s", launcherScript),
		"=policy=read,write,test",
	)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "api",
			ErrorMessage: fmt.Sprintf("添加启动器脚本失败: %s", err.Error()),
		}
	}

	// 添加调度器（每秒检查守护进程是否运行）
	schedulerName := config.SchedulerName
	if schedulerName == "" {
		schedulerName = "nmp-scheduler"
	}
	_, err = client.Run("/system/scheduler/add",
		fmt.Sprintf("=name=%s", schedulerName),
		fmt.Sprintf("=on-event=/system script run %s", launcherName),
		"=interval=00:00:01",
		"=policy=read,write,test",
		"=start-time=startup",
	)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "api",
			ErrorMessage: fmt.Sprintf("添加调度器失败: %s", err.Error()),
		}
	}

	// 启动脚本
	_, _ = client.Run("/system/script/run", fmt.Sprintf("=number=%s", launcherName))

	return &DeployResult{
		Success: true,
		Method:  "api",
		Message: "通过 API 部署成功",
	}
}

// deployViaSSH 通过 SSH 部署脚本
func (d *Deployer) deployViaSSH(config *ScriptConfig, ip string, port int, username, password string) *DeployResult {
	client, err := d.sshCollector.Connect(ip, port, username, password)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "ssh",
			ErrorMessage: err.Error(),
		}
	}
	defer client.Close()

	// 生成部署命令
	commands := d.generator.GenerateDeployCommands(config)

	// 执行部署命令
	for _, cmd := range commands {
		_, err := d.runSSHCommand(client, cmd)
		// 忽略移除命令的错误（可能不存在）
		if err != nil && !strings.Contains(cmd, "remove") {
			return &DeployResult{
				Success:      false,
				Method:       "ssh",
				ErrorMessage: fmt.Sprintf("执行命令失败: %s", err.Error()),
			}
		}
	}

	// 启动脚本
	startCmd := d.generator.GenerateStartCommand(config)
	_, _ = d.runSSHCommand(client, startCmd)

	return &DeployResult{
		Success: true,
		Method:  "ssh",
		Message: "通过 SSH 部署成功",
	}
}

// RemoveFromMikroTik 从 MikroTik 设备移除脚本
func (d *Deployer) RemoveFromMikroTik(scriptName, schedulerName, ip string, apiPort, sshPort int, username, password string) *DeployResult {
	// 先尝试 API
	result := d.removeViaAPI(scriptName, schedulerName, ip, apiPort, username, password)
	if result.Success {
		return result
	}

	// API 失败，尝试 SSH
	return d.removeViaSSH(scriptName, schedulerName, ip, sshPort, username, password)
}

// removeViaAPI 通过 API 移除脚本
func (d *Deployer) removeViaAPI(scriptName, schedulerName, ip string, port int, username, password string) *DeployResult {
	client, err := d.rosCollector.Connect(ip, port, username, password)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "api",
			ErrorMessage: err.Error(),
		}
	}
	defer client.Close()

	d.removeScriptViaAPI(client, scriptName, schedulerName)

	return &DeployResult{
		Success: true,
		Method:  "api",
		Message: "通过 API 移除成功",
	}
}

// removeViaSSH 通过 SSH 移除脚本
func (d *Deployer) removeViaSSH(scriptName, schedulerName, ip string, port int, username, password string) *DeployResult {
	client, err := d.sshCollector.Connect(ip, port, username, password)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "ssh",
			ErrorMessage: err.Error(),
		}
	}
	defer client.Close()

	d.removeScriptViaSSH(client, scriptName, schedulerName)

	return &DeployResult{
		Success: true,
		Method:  "ssh",
		Message: "通过 SSH 移除成功",
	}
}


// EnableScheduler 启用调度器
func (d *Deployer) EnableScheduler(schedulerName, ip string, apiPort, sshPort int, username, password string) *DeployResult {
	// 先尝试 API
	client, err := d.rosCollector.Connect(ip, apiPort, username, password)
	if err == nil {
		defer client.Close()
		_, err = client.Run("/system/scheduler/enable", fmt.Sprintf("=numbers=%s", schedulerName))
		if err == nil {
			return &DeployResult{
				Success: true,
				Method:  "api",
				Message: "调度器已启用",
			}
		}
	}

	// API 失败，尝试 SSH
	sshClient, err := d.sshCollector.Connect(ip, sshPort, username, password)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "none",
			ErrorMessage: err.Error(),
		}
	}
	defer sshClient.Close()

	cmd := fmt.Sprintf(`/system scheduler enable [find name="%s"]`, schedulerName)
	_, err = d.runSSHCommand(sshClient, cmd)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "ssh",
			ErrorMessage: err.Error(),
		}
	}

	return &DeployResult{
		Success: true,
		Method:  "ssh",
		Message: "调度器已启用",
	}
}

// DisableScheduler 禁用调度器
func (d *Deployer) DisableScheduler(schedulerName, ip string, apiPort, sshPort int, username, password string) *DeployResult {
	// 先尝试 API
	client, err := d.rosCollector.Connect(ip, apiPort, username, password)
	if err == nil {
		defer client.Close()
		_, err = client.Run("/system/scheduler/disable", fmt.Sprintf("=numbers=%s", schedulerName))
		if err == nil {
			return &DeployResult{
				Success: true,
				Method:  "api",
				Message: "调度器已禁用",
			}
		}
	}

	// API 失败，尝试 SSH
	sshClient, err := d.sshCollector.Connect(ip, sshPort, username, password)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "none",
			ErrorMessage: err.Error(),
		}
	}
	defer sshClient.Close()

	cmd := fmt.Sprintf(`/system scheduler disable [find name="%s"]`, schedulerName)
	_, err = d.runSSHCommand(sshClient, cmd)
	if err != nil {
		return &DeployResult{
			Success:      false,
			Method:       "ssh",
			ErrorMessage: err.Error(),
		}
	}

	return &DeployResult{
		Success: true,
		Method:  "ssh",
		Message: "调度器已禁用",
	}
}

// CheckScriptExists 检查脚本是否存在
func (d *Deployer) CheckScriptExists(scriptName, ip string, apiPort, sshPort int, username, password string) (bool, error) {
	// 先尝试 API
	client, err := d.rosCollector.Connect(ip, apiPort, username, password)
	if err == nil {
		defer client.Close()
		reply, err := client.Run("/system/script/print", fmt.Sprintf("?name=%s", scriptName))
		if err == nil {
			return len(reply.Re) > 0, nil
		}
	}

	// API 失败，尝试 SSH
	sshClient, err := d.sshCollector.Connect(ip, sshPort, username, password)
	if err != nil {
		return false, err
	}
	defer sshClient.Close()

	cmd := fmt.Sprintf(`/system script print where name="%s"`, scriptName)
	output, err := d.runSSHCommand(sshClient, cmd)
	if err != nil {
		return false, err
	}

	// 如果输出包含脚本名称，说明存在
	return strings.Contains(output, scriptName), nil
}

// API 辅助方法

func (d *Deployer) removeScriptViaAPI(client *routeros.Client, scriptName, schedulerName string) {
	// 先停止运行中的脚本
	reply, _ := client.Run("/system/script/job/print", fmt.Sprintf("?script=%s", scriptName))
	if len(reply.Re) > 0 {
		if id, ok := reply.Re[0].Map[".id"]; ok {
			client.Run("/system/script/job/remove", fmt.Sprintf("=.id=%s", id))
		}
	}

	// 移除调度器
	reply, err := client.Run("/system/scheduler/print", fmt.Sprintf("?name=%s", schedulerName))
	if err == nil && len(reply.Re) > 0 {
		if id, ok := reply.Re[0].Map[".id"]; ok {
			client.Run("/system/scheduler/remove", fmt.Sprintf("=.id=%s", id))
		}
	}

	// 移除启动器脚本
	launcherName := scriptName + "_launcher"
	reply, err = client.Run("/system/script/print", fmt.Sprintf("?name=%s", launcherName))
	if err == nil && len(reply.Re) > 0 {
		if id, ok := reply.Re[0].Map[".id"]; ok {
			client.Run("/system/script/remove", fmt.Sprintf("=.id=%s", id))
		}
	}

	// 移除主脚本
	reply, err = client.Run("/system/script/print", fmt.Sprintf("?name=%s", scriptName))
	if err == nil && len(reply.Re) > 0 {
		if id, ok := reply.Re[0].Map[".id"]; ok {
			client.Run("/system/script/remove", fmt.Sprintf("=.id=%s", id))
		}
	}
}

// SSH 辅助方法

func (d *Deployer) removeScriptViaSSH(client *ssh.Client, scriptName, schedulerName string) {
	// 先停止运行中的脚本
	cmd := fmt.Sprintf(`/system script job remove [find script="%s"]`, scriptName)
	d.runSSHCommand(client, cmd)

	// 移除调度器
	cmd = fmt.Sprintf(`/system scheduler remove [find name="%s"]`, schedulerName)
	d.runSSHCommand(client, cmd)

	// 移除启动器脚本
	launcherName := scriptName + "_launcher"
	cmd = fmt.Sprintf(`/system script remove [find name="%s"]`, launcherName)
	d.runSSHCommand(client, cmd)

	// 移除主脚本
	cmd = fmt.Sprintf(`/system script remove [find name="%s"]`, scriptName)
	d.runSSHCommand(client, cmd)
}

func (d *Deployer) runSSHCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return string(output), err
}

// UpdateScript 更新脚本（重新部署）
func (d *Deployer) UpdateScript(config *ScriptConfig, ip string, apiPort, sshPort int, username, password string) *DeployResult {
	// 更新脚本就是重新部署
	return d.DeployToMikroTik(config, ip, apiPort, sshPort, username, password)
}

// GetScriptStatus 获取脚本状态
func (d *Deployer) GetScriptStatus(scriptName, schedulerName, ip string, apiPort, sshPort int, username, password string) (*ScriptStatus, error) {
	status := &ScriptStatus{
		ScriptExists:    false,
		SchedulerExists: false,
		SchedulerEnabled: false,
	}

	// 先尝试 API
	client, err := d.rosCollector.Connect(ip, apiPort, username, password)
	if err == nil {
		defer client.Close()
		
		// 检查脚本
		reply, err := client.Run("/system/script/print", fmt.Sprintf("?name=%s", scriptName))
		if err == nil && len(reply.Re) > 0 {
			status.ScriptExists = true
		}

		// 检查调度器
		reply, err = client.Run("/system/scheduler/print", fmt.Sprintf("?name=%s", schedulerName))
		if err == nil && len(reply.Re) > 0 {
			status.SchedulerExists = true
			if disabled, ok := reply.Re[0].Map["disabled"]; ok {
				status.SchedulerEnabled = disabled != "true"
			} else {
				status.SchedulerEnabled = true
			}
		}

		return status, nil
	}

	// API 失败，尝试 SSH
	sshClient, err := d.sshCollector.Connect(ip, sshPort, username, password)
	if err != nil {
		return nil, err
	}
	defer sshClient.Close()

	// 检查脚本
	cmd := fmt.Sprintf(`/system script print where name="%s"`, scriptName)
	output, _ := d.runSSHCommand(sshClient, cmd)
	status.ScriptExists = strings.Contains(output, scriptName)

	// 检查调度器
	cmd = fmt.Sprintf(`/system scheduler print where name="%s"`, schedulerName)
	output, _ = d.runSSHCommand(sshClient, cmd)
	status.SchedulerExists = strings.Contains(output, schedulerName)
	status.SchedulerEnabled = !strings.Contains(output, "disabled=yes")

	return status, nil
}

// ScriptStatus 脚本状态
type ScriptStatus struct {
	ScriptExists     bool `json:"script_exists"`
	SchedulerExists  bool `json:"scheduler_exists"`
	SchedulerEnabled bool `json:"scheduler_enabled"`
}
