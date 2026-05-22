package config

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"github.com/zeromicro/go-zero/core/logc"
)

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	MaxConcurrentNodes int           `json:"maxConcurrentNodes"`
	MaxRetries         int           `json:"maxRetries"`
	RetryDelay         time.Duration `json:"retryDelay"`
	Timeout            time.Duration `json:"timeout"`
}

type Config struct {
	Server   Server          `json:"Server"`
	Database Database        `json:"Database"`
	Redis    Redis           `json:"Redis"`
	Jwt      Jwt             `json:"Jwt"`
	MCP      *MCPConfig      `json:"mcp"`
	Skill    *SkillConfig    `json:"skill"`
	Workflow *WorkflowConfig `json:"workflow"`
	License  LicenseConfig   `json:"license"`
}

type Server struct {
	Mode string `json:"mode"`
	Port string `json:"port"`
}

// SkillConfig Skill配置
type SkillConfig struct {
	Dir     string `json:"dir"`     // skill文件目录
	Enabled bool   `json:"enabled"` // 是否启用skill
	Default string `json:"default"` // 默认使用的skill名称
}

type Database struct {
	Type    string `json:"type"`    // mysql 或 sqlite
	Host    string `json:"host"`    // MySQL 主机地址
	Port    string `json:"port"`    // MySQL 端口
	User    string `json:"user"`    // MySQL 用户名
	Pass    string `json:"pass"`    // MySQL 密码
	DBName  string `json:"dbName"`  // MySQL 数据库名
	Timeout string `json:"timeout"` // MySQL 连接超时
	Path    string `json:"path"`    // SQLite 数据库文件路径
}

type Redis struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	Database int    `json:"database"`
}

type Jwt struct {
	Expire int64 `json:"expire"`
}

type MCPConfig struct {
	Servers map[string]MCPServer `json:"servers"`
}

// MCPServer MCP服务器配置
type MCPServer struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	APIKey      string            `json:"api_key,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description"`
}

// LicenseConfig License配置
type LicenseConfig struct {
	Path string `json:"path"` // license文件路径
}

var (
	Application = new(Config)
)

func Init() {
	v := viper.New()
	v.SetConfigFile("config/config.yaml")
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		logc.Error(context.Background(), fmt.Sprintf("配置读取失败: %s", err))
		return
	}

	if err := v.Unmarshal(&Application); err != nil {
		logc.Error(context.Background(), fmt.Sprintf("配置解析失败: %s", err))
		return
	}

	logc.Info(context.Background(), "配置文件初始化完成!")
}
