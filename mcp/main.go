package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gia "github.com/anwerj/gia-db/go"
	giampc "github.com/anwerj/gia-db/mcp/gia"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const defaultListen = "127.0.0.1:8080"

type config struct {
	Owner      string
	Repository string
	Token      string
}

type httpOptions struct {
	Listen           string
	AllowNonLoopback bool
	Origins          []string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gia-mcp: %s\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "stdio" || strings.HasPrefix(args[0], "-") {
		if len(args) > 0 && args[0] == "stdio" {
			args = args[1:]
		}
		cfg, err := parseStdio(args)
		if err != nil {
			return err
		}
		return serveStdio(cfg)
	}
	if args[0] == "http" {
		opts, cfg, err := parseHTTP(args[1:])
		if err != nil {
			return err
		}
		if err := validateListen(opts.Listen, opts.AllowNonLoopback); err != nil {
			return err
		}
		return serveHTTP(cfg, opts)
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func parseStdio(args []string) (config, error) {
	fs := flag.NewFlagSet("stdio", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return parseConfig(fs, args)
}

func parseHTTP(args []string) (httpOptions, config, error) {
	fs := flag.NewFlagSet("http", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	listen := fs.String("listen", defaultListen, "Streamable HTTP listen address")
	allowNonLoopback := fs.Bool("allow-non-loopback", false, "Allow a non-loopback listen address")
	var origins []string
	fs.Func("allow-origin", "Allowed browser Origin. Repeat for more than one.", func(value string) error {
		origins = append(origins, value)
		return nil
	})
	cfg, err := parseConfig(fs, args)
	if err != nil {
		return httpOptions{}, config{}, err
	}
	return httpOptions{
		Listen:           *listen,
		AllowNonLoopback: *allowNonLoopback,
		Origins:          origins,
	}, cfg, nil
}

func parseConfig(fs *flag.FlagSet, args []string) (config, error) {
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() != 0 {
		return config{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	owner := strings.TrimSpace(os.Getenv("GIADB_OWNER"))
	repository := strings.TrimSpace(os.Getenv("GIADB_REPOSITORY"))
	token := strings.TrimSpace(os.Getenv("GIADB_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}
	cfg := config{Owner: owner, Repository: repository, Token: token}
	if err := missingCredentials(cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func missingCredentials(cfg config) error {
	var missing []string
	if cfg.Owner == "" {
		missing = append(missing, "GIADB_OWNER is not set")
	}
	if cfg.Repository == "" {
		missing = append(missing, "GIADB_REPOSITORY is not set")
	}
	if cfg.Token == "" {
		missing = append(missing, "GIADB_TOKEN or GITHUB_TOKEN is not set")
	}
	if len(missing) == 0 {
		return nil
	}
	return errors.New(strings.Join(missing, "; "))
}

func validateListen(addr string, allowNonLoopback bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("listen address: %w", err)
	}
	if isLoopback(host) {
		return nil
	}
	if !allowNonLoopback {
		return fmt.Errorf("refusing non-loopback listen address %s without --allow-non-loopback", addr)
	}
	return nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func newClient(cfg config) (*gia.Client, error) {
	return gia.NewClient(gia.Options{
		Owner:      cfg.Owner,
		Repository: cfg.Repository,
		Token:      cfg.Token,
	})
}

func serveStdio(cfg config) error {
	db, err := newClient(cfg)
	if err != nil {
		return err
	}
	mcpServer, err := giampc.Build(context.Background(), db)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "gia-mcp: stdio transport\n")
	return mcpserver.ServeStdio(mcpServer, mcpserver.WithErrorLogger(log.New(os.Stderr, "gia-mcp: ", 0)))
}

func serveHTTP(cfg config, opts httpOptions) error {
	db, err := newClient(cfg)
	if err != nil {
		return err
	}
	mcpServer, err := giampc.Build(context.Background(), db)
	if err != nil {
		return err
	}
	serverOptions := []mcpserver.StreamableHTTPOption{
		mcpserver.WithEndpointPath("/mcp"),
	}
	if len(opts.Origins) > 0 {
		serverOptions = append(serverOptions, mcpserver.WithStreamableHTTPCORS(
			mcpserver.WithCORSAllowedOrigins(opts.Origins...),
		))
	}
	httpServer := mcpserver.NewStreamableHTTPServer(mcpServer, serverOptions...)
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Start(opts.Listen)
	}()
	fmt.Fprintf(os.Stderr, "gia-mcp: streamable HTTP listening on http://%s/mcp\n", opts.Listen)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownErr := httpServer.Shutdown(shutdownCtx)
		startErr := <-errCh
		if shutdownErr != nil {
			return shutdownErr
		}
		if startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			return startErr
		}
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
