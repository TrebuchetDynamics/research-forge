package retrieval

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPEmbeddingPostsTextAndModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request["text"] != "solar catalysts" || request["model"] != "fixture-model" {
			t.Fatalf("request = %#v", request)
		}
		_, _ = w.Write([]byte(`{"embedding":[0.1,0.2,0.3]}`))
	}))
	defer server.Close()

	emb := HTTPEmbedding{Endpoint: server.URL, Model: "fixture-model"}
	vector, err := emb.Embed("solar catalysts")
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}
	if emb.EmbeddingBackendName() != "http-embedding:fixture-model" {
		t.Fatalf("backend name = %s", emb.EmbeddingBackendName())
	}
	if len(vector) != 3 || vector[0] != 0.1 || vector[2] != 0.3 {
		t.Fatalf("vector = %#v", vector)
	}
}

func TestHTTPEmbeddingAcceptsVectorAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"vector":[1,0]}`))
	}))
	defer server.Close()

	vector, err := (HTTPEmbedding{Endpoint: server.URL}).Embed("query")
	if err != nil {
		t.Fatalf("Embed returned error: %v", err)
	}
	if len(vector) != 2 || vector[0] != 1 {
		t.Fatalf("vector = %#v", vector)
	}
}
