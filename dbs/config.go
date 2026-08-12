package dbs

import (
	"fmt"
	"time"
)

// DBConf 多数据库配置，key 为数据库名称（如 "tnebula"）
type DBConf map[string]DBInstance

// DBInstance 单个数据库实例配置，包含主库和从库
type DBInstance struct {
	Master   DBConnConfig   `json:",optional"`
	Replicas []DBConnConfig `json:",optional"`
}

// DBConnConfig 连接配置和连接池参数
type DBConnConfig struct {
	Dsn             string        `json:",optional"`
	MaxOpenConns    int           `json:",default=100"`
	MaxIdleConns    int           `json:",default=50"`
	ConnMaxLifetime time.Duration `json:",default=30m"`
	ConnMaxIdleTime time.Duration `json:",default=10m"`
}

// Default 填充默认值（gorm / go-zero 的 tag default 在 map value 中不生效）
func (c *DBConnConfig) Default() {
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = 100
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 50
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = 30 * time.Minute
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = 10 * time.Minute
	}
}

// Validate 校验配置合法性
func (c DBConf) Validate() error {
	for name, inst := range c {
		if inst.Master.Dsn == "" {
			return fmt.Errorf("Db.%s.Master.Dsn 不能为空", name)
		}
	}
	return nil
}
