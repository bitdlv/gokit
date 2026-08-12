package es

import (
	"context"
	"strings"
	"testing"
)

func TestElastic(t *testing.T) {

	esClient := MustNewEs(&Config{
		Addresses: []string{"http://10.0.0.101:9200"},
		Username:  "es",
		Password:  "123456",
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
