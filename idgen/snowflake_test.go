package idgen

import (
	"testing"
	"time"
)

func TestSnowflakeDefaultEpochReducesIDMagnitude(t *testing.T) {
	// 默认纪元 (2024-01-01) 下，2026 年生成的 ID 应远小于 Unix 纪元下的等价值
	s := NewSnowflakeWithWorker(1)
	id := s.NextID()
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
	// 相对 2024-01-01 至今约 2 年 = ~6.3e10 ms，左移 22 位后 ~ 2.6e17
	// Unix 纪元下当前时间戳 ~1.79e12，左移 22 位后 ~7.5e18
	// 差 ~30x，验证 epoch 生效
	if id >= 1<<62 {
		t.Errorf("id %d unexpectedly large; epoch not applied?", id)
	}
}

func TestSnowflakeWithUnixEpoch(t *testing.T) {
	// 显式退回 Unix 纪元
	s := NewSnowflakeWithWorker(1, WithEpoch(0))
	id := s.NextID()
	nowMs := time.Now().UnixNano() / 1e6
	expectedMin := nowMs << snowflakeTimeShift
	if id < expectedMin/2 {
		t.Errorf("id %d smaller than expected under unix epoch (>=~%d)", id, expectedMin)
	}
}

func TestSnowflakeMonotonic(t *testing.T) {
	s := NewSnowflakeWithWorker(7)
	var prev int64
	for i := 0; i < 10000; i++ {
		id := s.NextID()
		if id <= prev {
			t.Fatalf("non-monotonic: prev=%d cur=%d at i=%d", prev, id, i)
		}
		prev = id
	}
}
