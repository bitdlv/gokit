package random

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	// 设置全局前缀为 "order"
	if err := Init("PATROL"); err != nil {
		log.Fatal(err)
	}
	// 自动使用 Init 中设置的前缀
	fmt.Println("自动带前缀 ID:", GenerateIDWithPrefix())
	// 获取纯数字字符串 ID（无前缀）
	fmt.Println("纯字符串 ID1:", GenerateID())
	time.Sleep(time.Second * 2)
	fmt.Println("纯字符串 ID2:", GenerateID())
	time.Sleep(time.Second * 3)
	fmt.Println("纯字符串 ID3:", GenerateID())
}
