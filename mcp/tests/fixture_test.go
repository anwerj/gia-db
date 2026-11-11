package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/anwerj/gia-db/go"
	giampc "github.com/anwerj/gia-db/mcp/gia"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/therewardstore/httpmatter"
)

const preservedBody = "  keep\n\n→ markdown\n\n`code`\n{\"marker\":true}\n"

func useFixtures(t *testing.T, names ...string) {
	t.Helper()
	if len(names)%2 != 0 {
		t.Fatal("fixture names must be request/response pairs")
	}
	mock := httpmatter.NewHTTP(t, "github")
	for i := 0; i < len(names); i += 2 {
		mock = mock.Add(names[i], names[i+1]).Respond(matchFixture(t))
	}
	mock.Init()
	t.Cleanup(mock.Destroy)
}

func matchFixture(t *testing.T) func(*http.Request, *httpmatter.RequestMatter, []*httpmatter.ResponseMatter) *httpmatter.ResponseMatter {
	t.Helper()
	return func(req *http.Request, reqm *httpmatter.RequestMatter, respms []*httpmatter.ResponseMatter) *httpmatter.ResponseMatter {
		t.Helper()
		for key, want := range reqm.Header {
			if key == "Content-Length" || key == "User-Agent" {
				continue
			}
			if got := req.Header.Values(key); !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s header %s = %q, fixture %q", req.Method, req.URL, key, got, want)
			}
		}
		gotBody := readBody(t, req.Body)
		wantBody, err := reqm.BodyBytes()
		if err != nil {
			t.Fatal(err)
		}
		if !jsonEqual(gotBody, wantBody) {
			t.Errorf("%s %s body = %s, fixture %s", req.Method, req.URL, gotBody, wantBody)
		}
		return respms[0]
	}
}

func readBody(t *testing.T, body io.ReadCloser) []byte {
	t.Helper()
	if body == nil {
		return nil
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func jsonEqual(got, want []byte) bool {
	if len(bytes.TrimSpace(got)) == 0 && len(bytes.TrimSpace(want)) == 0 {
		return true
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		return false
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		return false
	}
	return reflect.DeepEqual(gotValue, wantValue)
}

func testClient(t *testing.T) *gia.Client {
	t.Helper()
	client, err := gia.NewClient(gia.Options{
		Owner:      "acme",
		Repository: "db",
		Token:      "test-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) != 1 {
		t.Fatalf("content = %#v", result)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content type %T", result.Content[0])
	}
	return text.Text
}

func decodeSuccess(t *testing.T, result *mcp.CallToolResult, dest any) {
	t.Helper()
	if result.IsError {
		t.Fatalf("tool error: %s", resultText(t, result))
	}
	if err := json.Unmarshal([]byte(resultText(t, result)), dest); err != nil {
		t.Fatalf("decode %s: %v", resultText(t, result), err)
	}
}

type parsedError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Outcome   string `json:"outcome"`
	Status    int    `json:"status"`
	RequestID string `json:"request_id"`
}

func decodeError(t *testing.T, result *mcp.CallToolResult) parsedError {
	t.Helper()
	if !result.IsError {
		t.Fatalf("expected tool error, got %s", resultText(t, result))
	}
	var document struct {
		Error parsedError `json:"error"`
	}
	if err := json.Unmarshal([]byte(resultText(t, result)), &document); err != nil {
		t.Fatal(err)
	}
	return document.Error
}

type resourceDocument struct {
	Resource struct {
		ID     int64    `json:"id"`
		Title  string   `json:"title"`
		Body   string   `json:"body"`
		Labels []string `json:"labels"`
		State  string   `json:"state"`
		ETag   string   `json:"etag"`
	} `json:"resource"`
	Meta struct {
		RequestID string `json:"request_id"`
		ETag      string `json:"etag"`
	} `json:"meta"`
}

type resourcePageDocument struct {
	Resources []struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
	} `json:"resources"`
	NextToken     string `json:"next_token"`
	Exhausted     bool   `json:"exhausted"`
	ReportedTotal *int   `json:"reported_total"`
	Incomplete    bool   `json:"incomplete"`
	Truncated     bool   `json:"truncated"`
	Meta          struct {
		Rate struct {
			Limit     int `json:"limit"`
			Remaining int `json:"remaining"`
		} `json:"rate"`
	} `json:"meta"`
}

type sectionDocument struct {
	Section struct {
		ID        int64  `json:"id"`
		Body      string `json:"body"`
		Author    string `json:"author"`
		CreatedAt string `json:"created_at"`
	} `json:"section"`
}

type sectionPageDocument struct {
	Sections []struct {
		ID     int64  `json:"id"`
		Body   string `json:"body"`
		Author string `json:"author"`
	} `json:"sections"`
	NextToken string `json:"next_token"`
	Exhausted bool   `json:"exhausted"`
}

func startSession(t *testing.T, mcpServer *server.MCPServer) *client.Client {
	t.Helper()
	session, err := client.NewInProcessClient(mcpServer)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	if _, err := session.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "gia-mcp-test",
				Version: "0.0.1",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	return session
}

func buildServer(t *testing.T) *server.MCPServer {
	t.Helper()
	mcpServer, err := giampc.Build(context.Background(), testClient(t))
	if err != nil {
		t.Fatal(err)
	}
	return mcpServer
}
