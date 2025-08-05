package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Env       string     `mapstructure:"env"`
	Server    Server     `mapstructure:"server"`
	Database  Database   `mapstructure:"database"`
	Logger    Logger     `mapstructure:"logger"`
	JWT       JWT        `mapstructure:"jwt"`
	RateLimit RateLimit  `mapstructure:"rate_limit"`
	Security  Security   `mapstructure:"security"`
}

type Server struct {
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

type Database struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	TimeZone        string `mapstructure:"timezone"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"`
}


type Logger struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type JWT struct {
	Secret     string `mapstructure:"secret"`
	Expiration int    `mapstructure:"expiration"`
}

type RateLimit struct {
	Enabled bool `mapstructure:"enabled"`
	RPS     int  `mapstructure:"rps"`
	Burst   int  `mapstructure:"burst"`
}

type Security struct {
	BcryptCost     int      `mapstructure:"bcrypt_cost"`
	PasswordMinLen int      `mapstructure:"password_min_len"`
	HTTPSOnly      bool     `mapstructure:"https_only"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	viper.SetDefault("env", "development")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 10)
	viper.SetDefault("server.write_timeout", 10)
	viper.SetDefault("server.idle_timeout", 60)
	
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "alertbot")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.dbname", "alertbot")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.timezone", "UTC")
	
	
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")
	
	viper.SetDefault("jwt.secret", "your-secret-key")
	viper.SetDefault("jwt.expiration", 24)
	
	viper.SetDefault("database.max_idle_conns", 25)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.conn_max_lifetime", 3600)
	viper.SetDefault("database.conn_max_idle_time", 1800)
	
	viper.SetDefault("rate_limit.enabled", true)
	viper.SetDefault("rate_limit.rps", 100)
	viper.SetDefault("rate_limit.burst", 200)
	
	viper.SetDefault("security.bcrypt_cost", 12)
	viper.SetDefault("security.password_min_len", 8)
	viper.SetDefault("security.https_only", false)
	viper.SetDefault("security.trusted_proxies", []string{})

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}