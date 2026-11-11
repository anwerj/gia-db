package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	giampc "github.com/anwerj/gia-db/mcp/gia"
	"github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/suite"
	"github.com/therewardstore/httpmatter"
)

type E2ESuite struct {
	suite.Suite
	binary string
}

func TestE2E(t *testing.T) {
	suite.Run(t, new(E2ESuite))
}

func (s *E2ESuite) SetupSuite() {
	_, file, _, ok := runtime.Caller(0)
	s.Require().True(ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	s.Require().NoError(httpmatter.Init(&httpmatter.Config{
		BaseDir:       filepath.Join(filepath.Dir(file), "testdata"),
		FileExtension: ".http",
	}))
	dir := s.T().TempDir()
	binary := filepath.Join(dir, "gia-mcp")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	s.Require().NoError(err, string(out))
	s.binary = binary
}

func (s *E2ESuite) TestRegistration() {
	_, err := giampc.Build(context.Background(), nil)
	s.Error(err)

	mcpServer := mcpserver.NewMCPServer("GiaDB", giampc.Version, mcpserver.WithToolCapabilities(true))
	s.Error(giampc.Register(context.Background(), mcpServer, []giampc.Tool{
		fakeTool{name: "create_resource"},
		fakeTool{name: "create_resource"},
	}))
	s.Error(giampc.Register(context.Background(), mcpServer, []giampc.Tool{
		fakeTool{name: "get_resource", defined: "fetch_resource"},
	}))

	session := s.session()
	listed, err := session.ListTools(context.Background(), mcp.ListToolsRequest{})
	s.Require().NoError(err)
	names := make([]string, 0, len(listed.Tools))
	byName := map[string]mcp.Tool{}
	for _, item := range listed.Tools {
		names = append(names, item.Name)
		byName[item.Name] = item
	}
	slices.Sort(names)
	s.Equal([]string{
		"create_resource",
		"forget_resource",
		"get_resource",
		"list_resources",
		"list_sections",
		"new_section",
		"reopen_resource",
		"update_resource",
	}, names)
	s.Contains(byName["create_resource"].InputSchema.Required, "title")
	s.Contains(byName["get_resource"].InputSchema.Required, "id")
	_, hasMatch := byName["get_resource"].InputSchema.Properties["if_none_match"]
	s.True(hasMatch)
	s.Contains(byName["new_section"].InputSchema.Required, "resource_id")
	s.Contains(byName["new_section"].InputSchema.Required, "body")
	_, hasID := byName["new_section"].InputSchema.Properties["id"]
	s.False(hasID)
}

func (s *E2ESuite) TestResources() {
	session := s.session(
		"create_resource_request", "create_resource_response",
		"create_resource_error_request", "create_resource_error_response",
		"get_resource_request", "get_resource_response",
		"get_resource_missing_request", "get_resource_missing_response",
		"get_resource_not_modified_request", "get_resource_not_modified_response",
		"update_resource_get_request", "update_resource_get_response",
		"update_resource_patch_request", "update_resource_patch_response",
		"update_resource_empty_get_request", "update_resource_empty_get_response",
		"update_resource_empty_patch_request", "update_resource_empty_patch_response",
		"update_resource_missing_request", "update_resource_missing_response",
		"forget_resource_get_request", "forget_resource_get_response",
		"forget_resource_patch_request", "forget_resource_patch_response",
		"forget_resource_missing_request", "forget_resource_missing_response",
		"reopen_resource_get_request", "reopen_resource_get_response",
		"reopen_resource_patch_request", "reopen_resource_patch_response",
		"reopen_resource_missing_request", "reopen_resource_missing_response",
	)

	missingTitle := decodeError(s.T(), s.call(session, "create_resource", map[string]any{"body": preservedBody}))
	s.Equal("invalid_argument", missingTitle.Code)
	s.Equal("not_applied", missingTitle.Outcome)

	var created resourceDocument
	decodeSuccess(s.T(), s.call(session, "create_resource", map[string]any{
		"title":  "Example",
		"body":   preservedBody,
		"labels": []string{"docs"},
	}), &created)
	s.Equal(int64(17), created.Resource.ID)
	s.Equal(preservedBody, created.Resource.Body)
	s.Equal("active", created.Resource.State)
	s.Equal([]string{"docs"}, created.Resource.Labels)
	s.Equal(`"etag-17"`, created.Resource.ETag)
	s.Equal(`"etag-17"`, created.Meta.ETag)

	validation := decodeError(s.T(), s.call(session, "create_resource", map[string]any{
		"title": "Broken",
		"body":  "nope",
	}))
	s.Equal("validation", validation.Code)
	s.Equal(422, validation.Status)
	s.Equal("req-422", validation.RequestID)
	s.Equal("not_applied", validation.Outcome)

	var got resourceDocument
	decodeSuccess(s.T(), s.call(session, "get_resource", map[string]any{"id": 4}), &got)
	s.Equal(int64(4), got.Resource.ID)
	s.Equal("Body", got.Resource.Body)
	s.Equal("active", got.Resource.State)
	s.Equal(`"etag-4"`, got.Resource.ETag)
	s.Equal("req-4", got.Meta.RequestID)

	missing := decodeError(s.T(), s.call(session, "get_resource", map[string]any{"id": 5}))
	s.Equal("not_found", missing.Code)
	s.Equal(404, missing.Status)
	s.Equal("req-404", missing.RequestID)
	s.Equal("not_applied", missing.Outcome)

	unchanged := decodeError(s.T(), s.call(session, "get_resource", map[string]any{
		"id": 3, "if_none_match": `"known"`,
	}))
	s.Equal("not_modified", unchanged.Code)
	s.Equal("not_applied", unchanged.Outcome)

	s.Equal("invalid_argument", decodeError(s.T(), s.call(session, "get_resource", map[string]any{"id": 0})).Code)
	s.Equal("invalid_argument", decodeError(s.T(), s.call(session, "get_resource", map[string]any{"id": 1.5})).Code)

	var updated resourceDocument
	decodeSuccess(s.T(), s.call(session, "update_resource", map[string]any{
		"id": 22, "body": preservedBody,
	}), &updated)
	s.Equal(int64(22), updated.Resource.ID)
	s.Equal(preservedBody, updated.Resource.Body)

	var cleared resourceDocument
	decodeSuccess(s.T(), s.call(session, "update_resource", map[string]any{
		"id": 24, "labels": []string{},
	}), &cleared)
	s.Equal(int64(24), cleared.Resource.ID)
	s.Equal("Body", cleared.Resource.Body)
	s.Empty(cleared.Resource.Labels)

	updateMissing := decodeError(s.T(), s.call(session, "update_resource", map[string]any{
		"id": 23, "title": "Next",
	}))
	s.Equal("not_found", updateMissing.Code)
	s.Equal("req-404", updateMissing.RequestID)
	s.Equal("invalid_argument", decodeError(s.T(), s.call(session, "update_resource", map[string]any{"id": 22})).Code)

	var forgotten resourceDocument
	decodeSuccess(s.T(), s.call(session, "forget_resource", map[string]any{"id": 31}), &forgotten)
	s.Equal(int64(31), forgotten.Resource.ID)
	s.Equal("forgotten", forgotten.Resource.State)
	s.Equal("Body", forgotten.Resource.Body)
	forgetMissing := decodeError(s.T(), s.call(session, "forget_resource", map[string]any{"id": 33}))
	s.Equal("not_found", forgetMissing.Code)
	s.Equal("req-404", forgetMissing.RequestID)

	var reopened resourceDocument
	decodeSuccess(s.T(), s.call(session, "reopen_resource", map[string]any{"id": 32}), &reopened)
	s.Equal(int64(32), reopened.Resource.ID)
	s.Equal("active", reopened.Resource.State)
	s.Equal("Body", reopened.Resource.Body)
	reopenMissing := decodeError(s.T(), s.call(session, "reopen_resource", map[string]any{"id": 34}))
	s.Equal("not_found", reopenMissing.Code)
	s.Equal("req-404", reopenMissing.RequestID)
}

func (s *E2ESuite) TestSectionsAndLists() {
	session := s.session(
		"list_resources_request", "list_resources_response",
		"list_resources_search_request", "list_resources_search_response",
		"list_resources_error_request", "list_resources_error_response",
		"new_section_get_request", "new_section_get_response",
		"new_section_post_request", "new_section_post_response",
		"new_section_missing_request", "new_section_missing_response",
		"list_sections_get_request", "list_sections_get_response",
		"list_sections_request", "list_sections_response",
		"list_sections_missing_request", "list_sections_missing_response",
	)

	var page resourcePageDocument
	decodeSuccess(s.T(), s.call(session, "list_resources", map[string]any{}), &page)
	s.Equal(base64.RawURLEncoding.EncodeToString([]byte("graphql:cursor-2")), page.NextToken)
	s.False(page.Exhausted)
	s.Nil(page.ReportedTotal)
	s.False(page.Incomplete)
	s.False(page.Truncated)
	s.Len(page.Resources, 2)
	s.Equal(int64(1), page.Resources[0].ID)
	s.Empty(page.Resources[0].Body)
	s.Equal(int64(2), page.Resources[1].ID)
	s.Equal("value", page.Resources[1].Body)
	s.Equal(5000, page.Meta.Rate.Limit)
	s.Equal(4999, page.Meta.Rate.Remaining)

	var search resourcePageDocument
	decodeSuccess(s.T(), s.call(session, "list_resources", map[string]any{"title_match": "example"}), &search)
	s.Equal(base64.RawURLEncoding.EncodeToString([]byte("search:2")), search.NextToken)
	s.False(search.Exhausted)
	s.Require().NotNil(search.ReportedTotal)
	s.Equal(1200, *search.ReportedTotal)
	s.True(search.Incomplete)
	s.True(search.Truncated)
	s.Len(search.Resources, 1)
	s.Equal(int64(9), search.Resources[0].ID)
	s.Equal("body", search.Resources[0].Body)

	auth := decodeError(s.T(), s.call(session, "list_resources", map[string]any{}))
	s.Equal("authentication", auth.Code)
	s.Equal(401, auth.Status)
	s.Equal("req-401", auth.RequestID)
	s.Equal("not_applied", auth.Outcome)
	s.Equal("invalid_argument", decodeError(s.T(), s.call(session, "list_resources", map[string]any{"page_size": 101})).Code)
	s.Equal("invalid_argument", decodeError(s.T(), s.call(session, "list_resources", map[string]any{"created_since": "yesterday"})).Code)

	var section sectionDocument
	decodeSuccess(s.T(), s.call(session, "new_section", map[string]any{
		"resource_id": 12,
		"body":        preservedBody,
	}), &section)
	s.Equal(int64(501), section.Section.ID)
	s.Equal(preservedBody, section.Section.Body)
	s.Equal("octocat", section.Section.Author)
	s.Equal("2026-01-02T03:04:05Z", section.Section.CreatedAt)
	sectionMissing := decodeError(s.T(), s.call(session, "new_section", map[string]any{
		"resource_id": 13,
		"body":        "section",
	}))
	s.Equal("not_found", sectionMissing.Code)
	s.Equal("req-404", sectionMissing.RequestID)
	s.Equal("invalid_argument", decodeError(s.T(), s.call(session, "new_section", map[string]any{"resource_id": 12})).Code)

	var sections sectionPageDocument
	decodeSuccess(s.T(), s.call(session, "list_sections", map[string]any{
		"resource_id": 14,
		"page_size":   10,
	}), &sections)
	s.Equal(base64.RawURLEncoding.EncodeToString([]byte("comments:2")), sections.NextToken)
	s.False(sections.Exhausted)
	s.Len(sections.Sections, 1)
	s.Equal(int64(501), sections.Sections[0].ID)
	s.Equal(preservedBody, sections.Sections[0].Body)
	s.Equal("octocat", sections.Sections[0].Author)
	s.Equal("not_found", decodeError(s.T(), s.call(session, "list_sections", map[string]any{"resource_id": 15})).Code)
}

func (s *E2ESuite) TestStreamableHTTP() {
	useFixtures(s.T(),
		"get_resource_request", "get_resource_response",
		"get_resource_missing_request", "get_resource_missing_response",
	)
	mcpServer, err := giampc.Build(context.Background(), testClient(s.T()))
	s.Require().NoError(err)
	httpServer := mcpserver.NewStreamableHTTPServer(mcpServer, mcpserver.WithEndpointPath("/mcp"))
	listener := httptest.NewServer(httpServer)
	s.T().Cleanup(listener.Close)

	session, err := client.NewStreamableHttpClient(
		listener.URL+"/mcp",
		mcptransport.WithHTTPBasicClient(&http.Client{Transport: &http.Transport{}}),
	)
	s.Require().NoError(err)
	ctx := context.Background()
	s.Require().NoError(session.Start(ctx))
	s.T().Cleanup(func() { session.Close() })
	_, err = session.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "gia-mcp-test", Version: "0.0.1"},
		},
	})
	s.Require().NoError(err)
	listed, err := session.ListTools(ctx, mcp.ListToolsRequest{})
	s.Require().NoError(err)
	s.Len(listed.Tools, 8)

	got, err := session.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_resource",
			Arguments: map[string]any{"id": 4},
		},
	})
	s.Require().NoError(err)
	s.False(got.IsError)

	missing, err := session.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_resource",
			Arguments: map[string]any{"id": 5},
		},
	})
	s.Require().NoError(err)
	s.True(missing.IsError)
	s.Contains(resultText(s.T(), missing), `"not_found"`)

	s.Require().NoError(httpServer.Shutdown(context.Background()))
	listener.Close()
	_, err = session.ListTools(ctx, mcp.ListToolsRequest{})
	s.Error(err)
}

func (s *E2ESuite) TestStdio() {
	stdout, stderr, code := s.speakStdio([]string{
		"GIADB_OWNER=acme",
		"GIADB_REPOSITORY=db",
		"GIADB_TOKEN=test-token",
	}, true)
	s.Equal(0, code)
	s.NotContains(stdout, "stdio transport")
	s.Contains(stderr, "stdio transport")
	s.NotContains(stderr, `"jsonrpc"`)
}

func (s *E2ESuite) TestCLI() {
	output, code := s.runCLI(nil, nil)
	s.Equal(1, code)
	s.Contains(output, "GIADB_OWNER is not set; GIADB_REPOSITORY is not set; GIADB_TOKEN or GITHUB_TOKEN is not set")

	output, code = s.runCLI(nil, []string{
		"GIADB_OWNER=acme",
		"GIADB_REPOSITORY=db",
	})
	s.Equal(1, code)
	s.Contains(output, "GIADB_TOKEN or GITHUB_TOKEN is not set")
	s.NotContains(output, "GIADB_OWNER is not set")

	output, code = s.runCLI([]string{"sse"}, nil)
	s.Equal(1, code)
	s.Contains(output, `unknown command "sse"`)

	output, code = s.runCLI([]string{"http", "--listen", "0.0.0.0:8080"}, []string{
		"GIADB_OWNER=acme",
		"GIADB_REPOSITORY=db",
		"GIADB_TOKEN=test-token",
	})
	s.Equal(1, code)
	s.Contains(output, "--allow-non-loopback")

	_, stderr, code := s.speakStdio([]string{
		"GIADB_OWNER=acme",
		"GIADB_REPOSITORY=db",
		"GITHUB_TOKEN=from-github",
	}, false)
	s.Equal(0, code)
	s.Contains(stderr, "stdio transport")
	s.NotContains(stderr, "are required")
}

func (s *E2ESuite) session(fixtures ...string) *client.Client {
	s.T().Helper()
	if len(fixtures) > 0 {
		useFixtures(s.T(), fixtures...)
	}
	return startSession(s.T(), buildServer(s.T()))
}

func (s *E2ESuite) call(session *client.Client, name string, args map[string]any) *mcp.CallToolResult {
	s.T().Helper()
	result, err := session.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	s.Require().NoError(err)
	return result
}

type fakeTool struct {
	name    string
	defined string
}

func (f fakeTool) Name() string { return f.name }

func (f fakeTool) Define(context.Context) mcp.Tool {
	name := f.defined
	if name == "" {
		name = f.name
	}
	return mcp.NewTool(name)
}

func (f fakeTool) Handle(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("ok"), nil
}

func (s *E2ESuite) runCLI(args, extra []string) (string, int) {
	s.T().Helper()
	cmd := exec.Command(s.binary, args...)
	cmd.Env = processEnv(extra)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err == nil {
		return buf.String(), 0
	}
	var exit *exec.ExitError
	s.Require().ErrorAs(err, &exit)
	return buf.String(), exit.ExitCode()
}

func (s *E2ESuite) speakStdio(extra []string, listTools bool) (string, string, int) {
	s.T().Helper()
	cmd := exec.Command(s.binary, "stdio")
	cmd.Env = processEnv(extra)
	stdin, err := cmd.StdinPipe()
	s.Require().NoError(err)
	stdout, err := cmd.StdoutPipe()
	s.Require().NoError(err)
	stderr, err := cmd.StderrPipe()
	s.Require().NoError(err)
	s.Require().NoError(cmd.Start())

	errDone := make(chan string, 1)
	go func() {
		data, readErr := io.ReadAll(stderr)
		if readErr != nil {
			errDone <- readErr.Error()
			return
		}
		errDone <- string(data)
	}()
	s.T().Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})

	reader := bufio.NewReader(stdout)
	var out bytes.Buffer
	writeMessage(s.T(), stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "gia-mcp-test", "version": "0.0.1"},
		},
	})
	initResponse := readResponse(s.T(), reader, &out, 1)
	s.Nil(initResponse["error"])
	writeMessage(s.T(), stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	if listTools {
		writeMessage(s.T(), stdin, map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/list",
		})
		listed := readResponse(s.T(), reader, &out, 2)
		raw, err := json.Marshal(listed["result"])
		s.Require().NoError(err)
		s.Contains(string(raw), `"create_resource"`)
		s.Contains(string(raw), `"reopen_resource"`)
	}
	s.Require().NoError(stdin.Close())
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	select {
	case err := <-wait:
		code := 0
		if err != nil {
			var exit *exec.ExitError
			s.Require().ErrorAs(err, &exit)
			code = exit.ExitCode()
		}
		select {
		case diagnostic := <-errDone:
			return out.String(), diagnostic, code
		case <-time.After(5 * time.Second):
			s.FailNow("timed out reading stderr")
		}
	case <-time.After(5 * time.Second):
		s.FailNow("stdio server did not exit after stdin closed")
	}
	return "", "", 1
}

func processEnv(extra []string) []string {
	env := []string{"PATH=" + os.Getenv("PATH")}
	if tmp := os.Getenv("TMPDIR"); tmp != "" {
		env = append(env, "TMPDIR="+tmp)
	}
	return append(env, extra...)
}

func writeMessage(t *testing.T, w io.Writer, payload any) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		t.Fatal(err)
	}
}

func readResponse(t *testing.T, reader *bufio.Reader, out *bytes.Buffer, id int) map[string]any {
	t.Helper()
	for range 5 {
		line := readLine(t, reader)
		out.Write(line)
		if bytes.Contains(line, []byte("stdio transport")) {
			t.Fatalf("stdout contains a diagnostic: %s", line)
		}
		var message map[string]any
		if err := json.Unmarshal(line, &message); err != nil {
			t.Fatalf("stdout is not JSON-RPC: %s", line)
		}
		if message["jsonrpc"] != "2.0" {
			t.Fatalf("stdout message = %s", line)
		}
		if message["id"] == float64(id) {
			return message
		}
	}
	t.Fatalf("missing response %d", id)
	return nil
}

func readLine(t *testing.T, reader *bufio.Reader) []byte {
	t.Helper()
	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := reader.ReadBytes('\n')
		ch <- result{line, err}
	}()
	select {
	case got := <-ch:
		if got.err != nil {
			t.Fatal(got.err)
		}
		return got.line
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for stdout")
	}
	return nil
}
