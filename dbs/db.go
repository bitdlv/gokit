package dbs

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// InitDB 初始化数据库连接，支持连接池和读写分离。
//
// 使用 gorm.io/plugin/dbresolver 实现：
//   - 写操作（Create/Update/Delete/Transaction）自动路由到 Master
//   - 读操作（Find/First/Take/Count）自动路由到 Replicas（随机策略）
//
// 返回的 *gorm.DB 可以直接用于 query.Use(db)，gen 生成的代码自动继承读写分离。
//
// name 为 DBConf 中的数据库实例键名（例如 "tnebula"）。
func InitDB(cfg DBConf, name string, gormCfg *gorm.Config) (*gorm.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	inst, ok := cfg[name]
	if !ok {
		return nil, fmt.Errorf("未找到数据库配置: %s", name)
	}

	inst.Master.Default()
	db, err := gorm.Open(mysql.Open(inst.Master.Dsn), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("连接主库失败: %w", err)
	}

	if err := configurePool(db, inst.Master); err != nil {
		return nil, fmt.Errorf("配置主库连接池失败: %w", err)
	}

	if len(inst.Replicas) > 0 {
		if err := setupReplicas(db, inst.Replicas); err != nil {
			return nil, fmt.Errorf("配置从库失败: %w", err)
		}
	}

	return db, nil
}

// configurePool 配置连接池参数
func configurePool(db *gorm.DB, cfg DBConnConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	return nil
}

// setupReplicas 配置从库并注册 dbresolver 插件
func setupReplicas(db *gorm.DB, replicas []DBConnConfig) error {
	if len(replicas) == 0 {
		return nil
	}

	dialectors := make([]gorm.Dialector, 0, len(replicas))
	for i := range replicas {
		replicas[i].Default()
		dialectors = append(dialectors, mysql.Open(replicas[i].Dsn))
	}

	resolverCfg := dbresolver.Config{
		Replicas: dialectors,
		Policy:   dbresolver.RandomPolicy{},
	}

	return db.Use(dbresolver.Register(resolverCfg).
		SetConnMaxIdleTime(replicas[0].ConnMaxIdleTime).
		SetConnMaxLifetime(replicas[0].ConnMaxLifetime).
		SetMaxIdleConns(replicas[0].MaxIdleConns).
		SetMaxOpenConns(replicas[0].MaxOpenConns),
	)
}

// DBPoolStats 返回连接池统计信息，用于健康检查
type DBPoolStats struct {
	Master struct {
		MaxOpenConnections int           `json:"max_open_connections"`
		OpenConnections    int           `json:"open_connections"`
		InUse              int           `json:"in_use"`
		Idle               int           `json:"idle"`
		WaitCount          int64         `json:"wait_count"`
		WaitDuration       time.Duration `json:"wait_duration"`
	} `json:"master"`
}

// GetPoolStats 获取主库连接池状态
func GetPoolStats(db *gorm.DB) (*DBPoolStats, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	stats := sqlDB.Stats()
	return &DBPoolStats{
		Master: struct {
			MaxOpenConnections int           `json:"max_open_connections"`
			OpenConnections    int           `json:"open_connections"`
			InUse              int           `json:"in_use"`
			Idle               int           `json:"idle"`
			WaitCount          int64         `json:"wait_count"`
			WaitDuration       time.Duration `json:"wait_duration"`
		}{
			MaxOpenConnections: stats.MaxOpenConnections,
			OpenConnections:    stats.OpenConnections,
			InUse:              stats.InUse,
			Idle:               stats.Idle,
			WaitCount:          stats.WaitCount,
			WaitDuration:       stats.WaitDuration,
		},
	}, nil
}
