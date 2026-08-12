package dbs

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

type MysqlConf struct {
	Dsn          string
	MaxOpenConns int `json:"MaxOpenConns,omitempty"`
	MaxIdleConns int `json:"MaxIdleConns,omitempty"`
	//ConnMaxLifetime time.Duration `json:"ConnMaxLifetime,omitempty"`
	//ConnMaxIdleTime time.Duration `json:"ConnMaxIdleTime,omitempty"`
}

type GormInitConf map[string]MysqlConf

var (
	instance = make(map[string]*gorm.DB)
	conf     GormInitConf
)

func Init(c GormInitConf, gormConfigs ...*gorm.Config) {
	conf = c
	for name, mc := range conf {
		var (
			db  *gorm.DB
			err error
		)

		if len(gormConfigs) > 0 {
			db, err = gorm.Open(mysql.Open(mc.Dsn), gormConfigs[0])
		} else {
			db, err = gorm.Open(mysql.Open(mc.Dsn))
		}
		if err != nil {
			panic(err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			panic(err)
		}

		// 给默认值，防止没配
		if mc.MaxOpenConns > 0 {
			sqlDB.SetMaxOpenConns(mc.MaxOpenConns)
		} else {
			sqlDB.SetMaxOpenConns(50)
		}

		if mc.MaxIdleConns > 0 {
			sqlDB.SetMaxIdleConns(mc.MaxIdleConns)
		} else {
			sqlDB.SetMaxIdleConns(20)
		}

		//if mc.ConnMaxLifetime > 0 {
		//	sqlDB.SetConnMaxLifetime(mc.ConnMaxLifetime)
		//} else {
		//	sqlDB.SetConnMaxLifetime(30 * time.Minute)
		//}
		sqlDB.SetConnMaxLifetime(30 * time.Minute)

		//if mc.ConnMaxIdleTime > 0 {
		//	sqlDB.SetConnMaxIdleTime(mc.ConnMaxIdleTime)
		//} else {
		//	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
		//}
		sqlDB.SetConnMaxIdleTime(10 * time.Minute)

		instance[name] = db
	}
}

func Get(name string) *gorm.DB {
	return instance[name]
}
