// Package collector 提供设备数据采集功能
package collector

import (
"fmt"
"strings"
)

func sanitizeVarName(name string) string {
result := strings.ReplaceAll(name, "-", "")
result = strings.ReplaceAll(result, "_", "")
result = strings.ReplaceAll(result, ".", "")
result = strings.ReplaceAll(result, " ", "")
return result
}

type ScriptConfig struct {
DeviceID      uint
DeviceIP      string
ServerURL     string
IntervalMs    int
PushBatchSize int
MaxQueueSize  int
ScriptName    string
LauncherName  string
SchedulerName string
Interfaces    []string
PingTargets   []PingTargetConfig
}

type PingTargetConfig struct {
TargetAddress   string
TargetName      string
SourceInterface string
}

type ScriptGenerator struct {
defaultServerURL string
}

func NewScriptGenerator(serverURL string) *ScriptGenerator {
return &ScriptGenerator{defaultServerURL: serverURL}
}

func (g *ScriptGenerator) GenerateMikroTikScript(config *ScriptConfig) string {
g.applyDefaults(config)
var sb strings.Builder
sb.WriteString(fmt.Sprintf("# NMP Collector Daemon - %s\n", config.DeviceIP))
sb.WriteString("# Long-running script\n")
sb.WriteString(":local queue \"\";\n")
sb.WriteString(":local cnt 0;\n")
sb.WriteString(fmt.Sprintf(":local maxQ %d;\n", config.MaxQueueSize))
sb.WriteString(fmt.Sprintf(":local batch %d;\n", config.PushBatchSize))
sb.WriteString(fmt.Sprintf(":local intv %d;\n", config.IntervalMs/1000))
sb.WriteString(fmt.Sprintf(":local url \"%s/api/push/metrics\";\n", config.ServerURL))
sb.WriteString(fmt.Sprintf(":local key \"%s\";\n", config.DeviceIP))
sb.WriteString(":while (true) do={\n")
sb.WriteString(":local ifData \"\";\n")
for _, iface := range config.Interfaces {
varName := sanitizeVarName(iface)
sb.WriteString(fmt.Sprintf(`:global nmpRx%s;:global nmpTx%s;:set nmpRx%s 0;:set nmpTx%s 0;`, varName, varName, varName, varName))
sb.WriteString(fmt.Sprintf(`:do {/interface monitor-traffic "%s" once do={:global nmpRx%s;:set nmpRx%s $"rx-bits-per-second";:global nmpTx%s;:set nmpTx%s $"tx-bits-per-second"}} on-error={};`, iface, varName, varName, varName, varName))
sb.WriteString(fmt.Sprintf(`:set ifData ($ifData . "\"%s\":{\"rx_rate\":" . $nmpRx%s . ",\"tx_rate\":" . $nmpTx%s . "},");`, iface, varName, varName))
sb.WriteString("\n")
}
sb.WriteString(":if ([:len $ifData] > 0) do={:set ifData [:pick $ifData 0 ([:len $ifData]-1)]};\n")
sb.WriteString(":local pingData \"\";\n")
for _, target := range config.PingTargets {
srcParam := ""
srcJson := ""
if target.SourceInterface != "" {
srcParam = fmt.Sprintf(` interface="%s"`, target.SourceInterface)
srcJson = target.SourceInterface
}
sb.WriteString(":do {\n")
sb.WriteString(fmt.Sprintf(`:local r [/ping %s count=1%s as-value];`+"\n", target.TargetAddress, srcParam))
sb.WriteString(`:local t ($r->"time");` + "\n")
sb.WriteString(`:if ([:typeof $t]="time") do={` + "\n")
sb.WriteString(":local ts [:tostr $t];\n")
sb.WriteString(":local dotPos [:find $ts \".\" -1];\n")
sb.WriteString(":local usStr [:pick $ts ($dotPos+1) [:len $ts]];\n")
sb.WriteString(":local us [:tonum $usStr];\n")
sb.WriteString(fmt.Sprintf(`:set pingData ($pingData . "{\"target\":\"%s\",\"src_iface\":\"%s\",\"latency\":" . $us . ",\"status\":\"up\"},");`+"\n", target.TargetAddress, srcJson))
sb.WriteString("} else={\n")
sb.WriteString(fmt.Sprintf(`:set pingData ($pingData . "{\"target\":\"%s\",\"src_iface\":\"%s\",\"latency\":0,\"status\":\"down\"},");`+"\n", target.TargetAddress, srcJson))
sb.WriteString("};\n")
sb.WriteString("} on-error={\n")
sb.WriteString(fmt.Sprintf(`:set pingData ($pingData . "{\"target\":\"%s\",\"src_iface\":\"%s\",\"latency\":0,\"status\":\"down\"},");`+"\n", target.TargetAddress, srcJson))
sb.WriteString("};\n")
}
sb.WriteString(":if ([:len $pingData] > 0) do={:set pingData [:pick $pingData 0 ([:len $pingData]-1)]};\n")
sb.WriteString(`:local pt ("{\"ts\":0,\"interfaces\":{" . $ifData . "},\"pings\":[" . $pingData . "]}");` + "\n")
sb.WriteString(":if ([:len $queue] > 0) do={\n")
sb.WriteString(`:set queue ($queue . "," . $pt);` + "\n")
sb.WriteString("} else={\n")
sb.WriteString(":set queue $pt;\n")
sb.WriteString("};\n")
sb.WriteString(":set cnt ($cnt + 1);\n")
sb.WriteString(":if ($cnt >= $batch) do={\n")
sb.WriteString(`:local pl ("{\"device_key\":\"" . $key . "\",\"metrics\":[" . $queue . "]}");` + "\n")
sb.WriteString(":do {\n")
sb.WriteString(`/tool fetch url=$url http-method=post http-data=$pl http-header-field="Content-Type:application/json" output=none;` + "\n")
sb.WriteString(":set queue \"\";\n")
sb.WriteString(":set cnt 0;\n")
sb.WriteString("} on-error={\n")
sb.WriteString(":if ($cnt > $maxQ) do={\n")
sb.WriteString(":set queue \"\";\n")
sb.WriteString(":set cnt 0;\n")
sb.WriteString("};\n")
sb.WriteString("};\n")
sb.WriteString("};\n")
sb.WriteString(":delay ($intv . \"s\");\n")
sb.WriteString("};\n")
return sb.String()
}

func (g *ScriptGenerator) GenerateMikroTikLauncher(config *ScriptConfig) string {
g.applyDefaults(config)
var sb strings.Builder
sb.WriteString("# NMP Collector Launcher\n")
sb.WriteString(fmt.Sprintf(`:local sn "%s"`+"\n", config.ScriptName))
sb.WriteString(":local jobs [/system script job print count-only as-value]\n")
sb.WriteString(":local running false\n")
sb.WriteString(":foreach j in=[/system script job find] do={\n")
sb.WriteString(":local jscript [/system script job get $j script]\n")
sb.WriteString(":if ($jscript = $sn) do={:set running true}\n")
sb.WriteString("}\n")
sb.WriteString(":if (!$running) do={\n")
sb.WriteString(":execute script=$sn\n")
sb.WriteString(`:log info "NMP collector daemon started"` + "\n")
sb.WriteString("} else={\n")
sb.WriteString(`:log info "NMP collector daemon already running"` + "\n")
sb.WriteString("}\n")
return sb.String()
}

func (g *ScriptGenerator) GenerateDeployCommands(config *ScriptConfig) []string {
g.applyDefaults(config)
mainScript := g.GenerateMikroTikScript(config)
launcherScript := g.GenerateMikroTikLauncher(config)
escapedMain := g.escapeForSSHCommand(mainScript)
escapedLauncher := g.escapeForSSHCommand(launcherScript)
return []string{
fmt.Sprintf(`/system script job remove [find script="%s"]`, config.ScriptName),
fmt.Sprintf(`/system scheduler remove [find name="%s"]`, config.SchedulerName),
fmt.Sprintf(`/system script remove [find name="%s"]`, config.LauncherName),
fmt.Sprintf(`/system script remove [find name="%s"]`, config.ScriptName),
fmt.Sprintf(`/system script add name="%s" source="%s" policy=read,write,test`, config.ScriptName, escapedMain),
fmt.Sprintf(`/system script add name="%s" source="%s" policy=read,write,test`, config.LauncherName, escapedLauncher),
fmt.Sprintf(`/system scheduler add name="%s" interval=00:00:01 on-event="/system script run %s" policy=read,write,test start-time=startup`, config.SchedulerName, config.LauncherName),
}
}

func (g *ScriptGenerator) GenerateRemoveCommands(config *ScriptConfig) []string {
g.applyDefaults(config)
return []string{
fmt.Sprintf(`/system script job remove [find script="%s"]`, config.ScriptName),
fmt.Sprintf(`/system scheduler remove [find name="%s"]`, config.SchedulerName),
fmt.Sprintf(`/system script remove [find name="%s"]`, config.LauncherName),
fmt.Sprintf(`/system script remove [find name="%s"]`, config.ScriptName),
}
}

func (g *ScriptGenerator) GenerateStartCommand(config *ScriptConfig) string {
g.applyDefaults(config)
return fmt.Sprintf(`/system script run %s`, config.LauncherName)
}

func (g *ScriptGenerator) GenerateStopCommand(config *ScriptConfig) string {
g.applyDefaults(config)
return fmt.Sprintf(`/system script job remove [find script="%s"]`, config.ScriptName)
}

func (g *ScriptGenerator) applyDefaults(config *ScriptConfig) {
if config.ServerURL == "" {
config.ServerURL = g.defaultServerURL
}
if config.ScriptName == "" {
config.ScriptName = "nmp-collector"
}
if config.LauncherName == "" {
config.LauncherName = "nmp-collector_launcher"
}
if config.SchedulerName == "" {
config.SchedulerName = "nmp-scheduler"
}
if config.IntervalMs <= 0 {
config.IntervalMs = 1000
}
if config.PushBatchSize <= 0 {
config.PushBatchSize = 3
}
if config.MaxQueueSize <= 0 {
config.MaxQueueSize = 60
}
}

func (g *ScriptGenerator) escapeForSSHCommand(s string) string {
s = strings.ReplaceAll(s, `"`, `\"`)
s = strings.ReplaceAll(s, "$", `\$`)
s = strings.ReplaceAll(s, "\n", "\r\n")
return s
}

func (g *ScriptGenerator) GetDefaultScriptName() string    { return "nmp-collector" }
func (g *ScriptGenerator) GetDefaultLauncherName() string  { return "nmp-collector_launcher" }
func (g *ScriptGenerator) GetDefaultSchedulerName() string { return "nmp-scheduler" }
