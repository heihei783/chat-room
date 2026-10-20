package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Email     EmailConfig     `mapstructure:"email"`
	Upload    UploadConfig    `mapstructure:"upload"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
	Host string `mapstructure:"host"`
}

type WebSocketConfig struct {
	ReadBufferSize   int           `mapstructure:"read_buffer_size"`
	WriteBufferSize  int           `mapstructure:"write_buffer_size"`
	HandshakeTimeout time.Duration `mapstructure:"handshake_timeout"`
	PingPeriod       time.Duration `mapstructure:"ping_period"`
	PongWait         time.Duration `mapstructure:"pong_wait"`
	MaxMessageSize   int64         `mapstructure:"max_message_size"`
}

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Charset  string `mapstructure:"charset"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire string `mapstructure:"expire"`
}

// EmailConfig SMTP邮件服务器配置（用于发送验证码等邮件）
type EmailConfig struct {
	Host     string `mapstructure:"host"`     // SMTP服务器地址
	Port     int    `mapstructure:"port"`     // SMTP端口：465=SSL，587=STARTTLS
	Username string `mapstructure:"username"` // 登录账号
	Password string `mapstructure:"password"` // 密码或授权码
	From     string `mapstructure:"from"`     // 发件人邮箱
	SSL      bool   `mapstructure:"ssl"`      // 是否使用SSL
}

// UploadConfig 文件上传配置
type UploadConfig struct {
	Dir       string `mapstructure:"dir"`        // 文件存储目录（相对项目根目录或绝对路径）
	URLPrefix string `mapstructure:"url_prefix"` // 静态访问URL前缀（与Dir对应，如 /uploads）
	MaxSize   int64  `mapstructure:"max_size"`   // 单文件大小上限（字节），默认10MB
}

var AppConfig *Config

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败：%w", err)
	}

	viper.AutomaticEnv()

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败：%w", err)
	}

	AppConfig = config
	return config, nil
}

func setDefaults() {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.max_idle_conns", "10")
	viper.SetDefault("database.max_open_conns", "100")
}
