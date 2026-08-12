package nacos

import (
	"fmt"
	"testing"
	"time"
)

type Config struct {
	Name     string
	ListenOn string
}

func TestNewNsClient(t *testing.T) {
	var c Config
	err := LoadNsConfig("calculate", &c)
	time.Sleep(100 * time.Second)
	fmt.Println(c, err)
}

func TestNewNsListenClient(t *testing.T) {

}
