package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server" validate:"required"`
	Database DatabaseConfig `mapstructure:"database" validate:"required"`
	Redis    RedisConfig    `mapstructure:"redis" validate:"required"`
	InfluxDB InfluxConfig   `mapstructure:"influxdb" validate:"required"`
	Auth     AuthConfig     `mapstructure:"auth" validate:"required"`
	Plugins  PluginConfigs  `mapstructure:"plugins" validate:"required"`
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Host         string        `mapstructure:"host" validate:"required"`
	Port         int           `mapstructure:"port" validate:"required,min=1,max=65535"`
	Mode         string        `mapstructure:"mode" validate:"required,oneof=debug release test"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" validate:"required"`
	PublicURL    string        `mapstructure:"public_url"` // 设备回调使用的公网/内网地址，如 http://10.10.10.231:8080
}

// DatabaseConfig PostgreSQL数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	Database string `mapstructure:"database" validate:"required"`
	Username string `mapstructure:"username" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	SSLMode  string `mapstructure:"ssl_mode" validate:"required,oneof=disable require verify-ca verify-full"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db" validate:"min=0,max=15"`
}

// InfluxConfig InfluxDB配置
type InfluxConfig struct {
	URL    string `mapstructure:"url" validate:"required,url"`
	Token  string `mapstructure:"token"`
	Org    string `mapstructure:"org" validate:"required"`
	Bucket string `mapstructure:"bucket" validate:"required"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret     string        `mapstructure:"jwt_secret" validate:"required,min=32"`
	TokenExpiry   time.Duration `mapstructure:"token_expiry" validate:"required"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry" validate:"required"`
}

// PluginConfigs 插件配置
type PluginConfigs struct {
	Directory string                 `mapstructure:"directory" validate:"required"`
	Configs   map[string]interface{} `mapstructure:"configs"`
}

// Load 加载配置文件
func Load() (*Config, error) {
	return LoadWithPath("./configs")
}

// LoadWithPath 从指定路径加载配置文件
func LoadWithPath(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")

	// 设置默认值
	setDefaults()

	// 配置环境变量支持
	setupEnvironmentVariables()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，使用默认值和环境变量
			fmt.Println("Config file not found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// 应用简写环境变量覆盖
	applyEnvironmentOverrides(&config)

	// 生产环境配置验证
	if config.Server.Mode == "release" {
		if err := validateProductionConfig(&config); err != nil {
			return nil, fmt.Errorf("production config validation failed: %w", err)
		}
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setupEnvironmentVariables 配置环境变量支持
func setupEnvironmentVariables() {
	// 设置环境变量前缀
	viper.SetEnvPrefix("NMP")
	
	// 自动读取环境变量
	viper.AutomaticEnv()
	
	// 设置环境变量键名替换规则（将点号替换为下划线）
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// 绑定特定的环境变量（支持完整前缀和简写两种形式）
	// 完整前缀形式：NMP_DATABASE_PASSWORD
	// 简写形式：DB_PASSWORD
	envBindings := map[string][]string{
		"server.host":           {"NMP_SERVER_HOST"},
		"server.port":           {"NMP_SERVER_PORT"},
		"server.mode":           {"NMP_SERVER_MODE"},
		"database.host":         {"NMP_DATABASE_HOST", "DB_HOST"},
		"database.port":         {"NMP_DATABASE_PORT", "DB_PORT"},
		"database.database":     {"NMP_DATABASE_NAME", "DB_NAME"},
		"database.username":     {"NMP_DATABASE_USERNAME", "DB_USER"},
		"database.password":     {"NMP_DATABASE_PASSWORD", "DB_PASSWORD"},
		"database.ssl_mode":     {"NMP_DATABASE_SSL_MODE", "DB_SSL_MODE"},
		"redis.host":            {"NMP_REDIS_HOST", "REDIS_HOST"},
		"redis.port":            {"NMP_REDIS_PORT", "REDIS_PORT"},
		"redis.password":        {"NMP_REDIS_PASSWORD", "REDIS_PASSWORD"},
		"redis.db":              {"NMP_REDIS_DB", "REDIS_DB"},
		"influxdb.url":          {"NMP_INFLUXDB_URL", "INFLUXDB_URL"},
		"influxdb.token":        {"NMP_INFLUXDB_TOKEN", "INFLUXDB_TOKEN"},
		"influxdb.org":          {"NMP_INFLUXDB_ORG", "INFLUXDB_ORG"},
		"influxdb.bucket":       {"NMP_INFLUXDB_BUCKET", "INFLUXDB_BUCKET"},
		"auth.jwt_secret":       {"NMP_AUTH_JWT_SECRET", "JWT_SECRET"},
		"auth.token_expiry":     {"NMP_AUTH_TOKEN_EXPIRY"},
		"auth.refresh_expiry":   {"NMP_AUTH_REFRESH_EXPIRY"},
		"plugins.directory":     {"NMP_PLUGINS_DIRECTORY"},
	}
	
	for key, envVars := range envBindings {
		// 绑定所有环境变量名称到同一个配置键
		// viper.BindEnv 接受可变参数，第一个是配置键，后面是环境变量名
		args := append([]string{key}, envVars...)
		viper.BindEnv(args...)
	}
}

// applyEnvironmentOverrides 应用环境变量覆盖（用于简写环境变量）
// 这个函数在配置加载后调用，确保简写环境变量能正确覆盖配置
func applyEnvironmentOverrides(config *Config) {
	// 数据库密码
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		config.Database.Password = dbPassword
	}
	
	// InfluxDB Token
	if influxToken := os.Getenv("INFLUXDB_TOKEN"); influxToken != "" {
		config.InfluxDB.Token = influxToken
	}
	
	// JWT Secret
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.Auth.JWTSecret = jwtSecret
	}
	
	// 数据库其他配置
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		config.Database.Username = dbUser
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		config.Database.Database = dbName
	}
	
	// Redis 配置
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Redis.Host = redisHost
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.Redis.Password = redisPassword
	}
	
	// InfluxDB 其他配置
	if influxURL := os.Getenv("INFLUXDB_URL"); influxURL != "" {
		config.InfluxDB.URL = influxURL
	}
}

// validateProductionConfig 生产环境配置验证
// 确保敏感配置项已通过环境变量正确设置
func validateProductionConfig(config *Config) error {
	var errors []string

	// 验证数据库密码不是默认值且足够强
	if config.Database.Password == "" {
		errors = append(errors, "DB_PASSWORD environment variable is required in production")
	} else if len(config.Database.Password) < 12 {
		errors = append(errors, "DB_PASSWORD must be at least 12 characters long")
	} else if isWeakPassword(config.Database.Password) {
		errors = append(errors, "DB_PASSWORD is too weak (avoid common passwords like 'nmp123', 'password', etc.)")
	}

	// 验证 InfluxDB Token
	if config.InfluxDB.Token == "" {
		errors = append(errors, "INFLUXDB_TOKEN environment variable is required in production")
	} else if len(config.InfluxDB.Token) < 32 {
		errors = append(errors, "INFLUXDB_TOKEN must be at least 32 characters long")
	}

	// 验证 JWT Secret 不是默认值且长度足够
	if config.Auth.JWTSecret == "" {
		errors = append(errors, "JWT_SECRET environment variable is required in production")
	} else if isWeakJWTSecret(config.Auth.JWTSecret) {
		errors = append(errors, "JWT_SECRET is too weak or uses a default value")
	} else if len(config.Auth.JWTSecret) < 32 {
		errors = append(errors, "JWT_SECRET must be at least 32 characters long")
	}

	if len(errors) > 0 {
		return fmt.Errorf("production configuration errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// isWeakPassword 检查是否是弱密码
func isWeakPassword(password string) bool {
	weakPasswords := []string{
		"nmp123", "nmp1234", "nmp12345",
		"password", "password123",
		"admin", "admin123", "admin1234",
		"123456", "12345678", "123456789",
		"qwerty", "qwerty123",
		"root", "root123",
	}
	
	lowerPassword := strings.ToLower(password)
	for _, weak := range weakPasswords {
		if lowerPassword == weak {
			return true
		}
	}
	return false
}

// isWeakJWTSecret 检查是否是弱JWT密钥
func isWeakJWTSecret(secret string) bool {
	weakSecrets := []string{
		"nmp-secret-key-change-in-production",
		"change-this-secret-in-production",
		"your-secret-key",
		"secret",
		"jwt-secret",
		"my-secret-key",
	}
	
	lowerSecret := strings.ToLower(secret)
	for _, weak := range weakSecrets {
		if strings.Contains(lowerSecret, weak) {
			return true
		}
	}
	return false
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	validate := validator.New()
	
	if err := validate.Struct(config); err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, 
				fmt.Sprintf("Field '%s' failed validation: %s", err.Field(), err.Tag()))
		}
		return fmt.Errorf("validation errors: %s", strings.Join(validationErrors, "; "))
	}
	
	// 自定义验证逻辑
	if err := validateCustomRules(config); err != nil {
		return err
	}
	
	return nil
}

// validateCustomRules 自定义验证规则
func validateCustomRules(config *Config) error {
	// 验证JWT密钥长度
	if len(config.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}
	
	// 验证令牌过期时间
	if config.Auth.TokenExpiry <= 0 {
		return fmt.Errorf("token expiry must be positive")
	}
	
	if config.Auth.RefreshExpiry <= config.Auth.TokenExpiry {
		return fmt.Errorf("refresh expiry must be greater than token expiry")
	}
	
	// 验证插件目录是否存在
	if _, err := os.Stat(config.Plugins.Directory); os.IsNotExist(err) {
		// 尝试创建插件目录
		if err := os.MkdirAll(config.Plugins.Directory, 0755); err != nil {
			return fmt.Errorf("plugin directory does not exist and cannot be created: %s", config.Plugins.Directory)
		}
	}
	
	return nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// 服务器默认配置
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")

	// 数据库默认配置
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "nmp")
	viper.SetDefault("database.username", "nmp")
	viper.SetDefault("database.password", "nmp123")
	viper.SetDefault("database.ssl_mode", "disable")

	// Redis默认配置
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// InfluxDB默认配置
	viper.SetDefault("influxdb.url", "http://localhost:8086")
	viper.SetDefault("influxdb.token", "")
	viper.SetDefault("influxdb.org", "nmp")
	viper.SetDefault("influxdb.bucket", "monitoring")

	// 认证默认配置
	viper.SetDefault("auth.jwt_secret", "nmp-secret-key-change-in-production")
	viper.SetDefault("auth.token_expiry", "24h")
	viper.SetDefault("auth.refresh_expiry", "168h")

	// 插件默认配置
	viper.SetDefault("plugins.directory", "./plugins")
}

// GetConfig 获取当前配置实例（单例模式）
var (
	globalConfig *Config
	configWatcher *ConfigWatcher
)

func GetConfig() *Config {
	return globalConfig
}

// SetConfig 设置全局配置实例
func SetConfig(config *Config) {
	globalConfig = config
}

// GetConfigWatcher 获取配置监听器
func GetConfigWatcher() *ConfigWatcher {
	return configWatcher
}

// SetConfigWatcher 设置配置监听器
func SetConfigWatcher(watcher *ConfigWatcher) {
	configWatcher = watcher
}

// Reload 重新加载配置
func Reload() (*Config, error) {
	config, err := Load()
	if err != nil {
		return nil, err
	}
	
	SetConfig(config)
	return config, nil
}

// GetString 获取字符串配置值
func GetString(key string) string {
	return viper.GetString(key)
}

// GetInt 获取整数配置值
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool 获取布尔配置值
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetDuration 获取时间间隔配置值
func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

// Set 设置配置值
func Set(key string, value interface{}) {
	viper.Set(key, value)
}

// IsSet 检查配置键是否已设置
func IsSet(key string) bool {
	return viper.IsSet(key)
}