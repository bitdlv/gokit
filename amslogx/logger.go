package amslogx

import (
	"context"
	"github.com/bitdlv/gokit/middleware"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm/logger"
	"time"
)

type AmsLogger struct {
	logx.Logger
	ctx context.Context
}

func New(ctx context.Context) *AmsLogger {
	//logx.SetLevel(logx.ErrorLevel)
	amsLogger := &AmsLogger{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
	}
	return amsLogger.withRequestID()
}

func (l *AmsLogger) withRequestID() *AmsLogger {
	requestID := l.getRequestID()
	if requestID != "" {
		l.Logger = l.Logger.WithFields(logx.LogField{
			Key:   middleware.LogRequestIdKey,
			Value: requestID,
		})
	}
	return l
}

func (l *AmsLogger) getRequestID() string {
	if md, ok := metadata.FromIncomingContext(l.ctx); ok {
		if values := md.Get(middleware.RequestIdKey); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func (l *AmsLogger) Info(v ...interface{}) {
	l.Logger.Info(v...)
}

func (l *AmsLogger) Infof(format string, v ...interface{}) {
	l.Logger.Infof(format, v...)
}

func (l *AmsLogger) Error(v ...interface{}) {
	l.Logger.Error(v...)
}

func (l *AmsLogger) Errorf(format string, v ...interface{}) {
	l.Logger.Errorf(format, v...)
}

func (l *AmsLogger) Warn(v ...interface{}) {
	l.Logger.Info(v...)
}

func (l *AmsLogger) Warnf(format string, v ...interface{}) {
	l.Logger.Infof(format, v...)
}

// GormLogxLogger Gorm 使用日志
type GormLogxLogger struct {
	logLevel logger.LogLevel
}

// NewGormLogxLogger 创建一个新的 GormLogxLogger
func NewGormLogxLogger(logLevel logger.LogLevel) logger.Interface {
	return &GormLogxLogger{logLevel: logLevel}
}

// LogMode 设置日志模式
func (l *GormLogxLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info 打印信息日志
func (l *GormLogxLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Info {
		//logx.Infof(msg, data...)
		New(ctx).Infof(msg, data...)
	}
}

// Warn 打印警告日志
func (l *GormLogxLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Warn {
		logx.Debugf(msg, data...)
	}
}

// Error 打印错误日志
func (l *GormLogxLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= logger.Error {
		logx.Errorf(msg, data...)
	}
}

// Trace 打印慢查询日志
func (l *GormLogxLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	if err != nil && l.logLevel >= logger.Error {
		logx.Errorf("GORM Error: %v | Elapsed: %v | Rows: %v | SQL: %v", err, elapsed, rows, sql)
	} else if elapsed > time.Millisecond*200 && l.logLevel >= logger.Warn { // 慢查询阈值: 200ms
		logx.Debugf("GORM Slow Query (>200ms): Elapsed: %v | Rows: %v | SQL: %v", elapsed, rows, sql)
	} else if l.logLevel >= logger.Info {
		logx.Infof("GORM Query: Elapsed: %v | Rows: %v | SQL: %v", elapsed, rows, sql)
	}
}
