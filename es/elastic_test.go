package es

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestElastic 集成测试示例。
//
// 默认跳过。设置 GOKIT_ES_INTEGRATION=1 并提供
// GOKIT_ES_ADDR / GOKIT_ES_USER / GOKIT_ES_PASS 后才会真正运行。
//
//	export GOKIT_ES_INTEGRATION=1
//	export GOKIT_ES_ADDR=http://localhost:9200
//	export GOKIT_ES_USER=
//	export GOKIT_ES_PASS=
//	go test -run TestElastic ./es/...
func TestElastic(t *testing.T) {
	if os.Getenv("GOKIT_ES_INTEGRATION") != "1" {
		t.Skip("integration test skipped; set GOKIT_ES_INTEGRATION=1 to enable")
	}

	addr := os.Getenv("GOKIT_ES_ADDR")
	if addr == "" {
		addr = "http://localhost:9200"
	}

	esClient := MustNewEs(&Config{
		Addresses: []string{addr},
		Username:  os.Getenv("GOKIT_ES_USER"),
		Password:  os.Getenv("GOKIT_ES_PASS"),
	})

	searchResult, err := esClient.Search(
		esClient.Search.WithContext(context.Background()),
		esClient.Search.WithIndex("mybook"),
		esClient.Search.WithBody(strings.NewReader(`{"query":{"match":{"title":"三体"}}}`)),
		esClient.Search.WithPretty(),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("searchResult: ", searchResult)
}
