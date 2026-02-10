package dash0

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// safeBuffer is a thread-safe buffer for capturing collector output concurrently.
type safeBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (sb *safeBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf = append(sb.buf, p...)
	return len(p), nil
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return string(sb.buf)
}

// readOtelcolVersion reads the collector version from .otelcol-version.
func readOtelcolVersion(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(".otelcol-version")
	if err != nil {
		t.Fatalf("failed to read .otelcol-version: %v", err)
	}
	return strings.TrimSpace(string(data))
}

// resolveLatestOtelcolVersion resolves "latest" to an actual version number
// by following the GitHub releases/latest redirect.
func resolveLatestOtelcolVersion(t *testing.T) string {
	t.Helper()
	// Use a client that does not follow redirects so we can read the Location header.
	httpClient := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := httpClient.Get("https://github.com/open-telemetry/opentelemetry-collector-releases/releases/latest")
	if err != nil {
		t.Fatalf("failed to resolve latest OTel Collector version: %v", err)
	}
	_ = resp.Body.Close()

	loc := resp.Header.Get("Location")
	// Location looks like: .../releases/tag/v0.145.0
	if i := strings.LastIndex(loc, "/v"); i != -1 {
		version := loc[i+2:]
		t.Logf("resolved 'latest' to OTel Collector version %s", version)
		return version
	}
	t.Fatalf("could not parse version from redirect Location: %s", loc)
	return ""
}

// ensureOtelcol returns the path to a cached collector binary, downloading it if necessary.
// If version is "latest", it resolves the actual version from GitHub releases.
func ensureOtelcol(t *testing.T, version string) string {
	t.Helper()

	version = strings.TrimPrefix(version, "v")

	if version == "latest" {
		version = resolveLatestOtelcolVersion(t)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	switch goos {
	case "linux", "darwin":
		// supported
	default:
		t.Skipf("unsupported OS for OTel Collector test: %s", goos)
	}
	switch goarch {
	case "amd64", "arm64":
		// supported
	default:
		t.Skipf("unsupported architecture for OTel Collector test: %s", goarch)
	}

	cacheDir := filepath.Join("testdata", ".cache", fmt.Sprintf("otelcol-%s", version))
	binaryPath := filepath.Join(cacheDir, "otelcol")

	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// Download the collector binary.
	archiveName := fmt.Sprintf("otelcol_%s_%s_%s.tar.gz", version, goos, goarch)
	url := fmt.Sprintf(
		"https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v%s/%s",
		version, archiveName,
	)

	t.Logf("downloading OTel Collector %s from %s", version, url)

	resp, err := http.Get(url)
	if err != nil {
		t.Skipf("failed to download OTel Collector (network error, skipping): %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		t.Fatalf("OTel Collector version %s not found (HTTP 404).\n"+
			"Check that .otelcol-version contains a valid release version from:\n"+
			"https://github.com/open-telemetry/opentelemetry-collector-releases/releases",
			version)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to download OTel Collector (HTTP %d) from %s", resp.StatusCode, url)
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}

	// Extract tar.gz to find the otelcol binary.
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read tar entry: %v", err)
		}
		if hdr.Name == "otelcol" && hdr.Typeflag == tar.TypeReg {
			f, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				t.Fatalf("failed to create otelcol binary: %v", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				t.Fatalf("failed to write otelcol binary: %v", err)
			}
			_ = f.Close()
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("otelcol binary not found in archive %s", archiveName)
	}

	return binaryPath
}

// freePort returns an available TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// waitForPort polls until the given port is accepting connections or the timeout is reached.
func waitForPort(t *testing.T, port int, timeout time.Duration, output *safeBuffer) {
	t.Helper()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for port %d to be ready.\nCollector output:\n%s", port, output.String())
}

// collectorConfig returns a YAML config for the OTel Collector with the given OTLP HTTP port.
func collectorConfig(port int) string {
	return fmt.Sprintf(`receivers:
  otlp:
    protocols:
      http:
        endpoint: "127.0.0.1:%d"

exporters:
  debug:
    verbosity: detailed

service:
  telemetry:
    logs:
      level: debug
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      exporters: [debug]
    logs:
      receivers: [otlp]
      exporters: [debug]
`, port)
}

// startCollector starts the collector subprocess with the given binary and config paths.
func startCollector(t *testing.T, binaryPath, configPath string) (*exec.Cmd, *safeBuffer) {
	t.Helper()

	cmd := exec.Command(binaryPath, "--config=file:"+configPath)
	output := &safeBuffer{}
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start collector: %v", err)
	}

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	return cmd, output
}

func TestOTLPCollector(t *testing.T) {
	version := readOtelcolVersion(t)
	binaryPath := ensureOtelcol(t, version)
	port := freePort(t)

	// Write collector config to a temp directory.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(collectorConfig(port)), 0644); err != nil {
		t.Fatalf("failed to write collector config: %v", err)
	}

	// Start collector and wait for it to be ready.
	_, output := startCollector(t, binaryPath, configPath)
	waitForPort(t, port, 10*time.Second, output)

	// Create a dash0 client pointed at the collector's OTLP endpoint.
	c, err := NewClient(
		WithOtlpEndpoint(OtlpEncodingJson, fmt.Sprintf("http://127.0.0.1:%d", port)),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })

	// Send all three signal types.
	t.Run("send traces", func(t *testing.T) {
		if err := c.SendTraces(context.Background(), newTestTraces(), nil); err != nil {
			t.Fatalf("SendTraces failed: %v", err)
		}
	})

	t.Run("send metrics", func(t *testing.T) {
		if err := c.SendMetrics(context.Background(), newTestMetrics(), nil); err != nil {
			t.Fatalf("SendMetrics failed: %v", err)
		}
	})

	t.Run("send logs", func(t *testing.T) {
		if err := c.SendLogs(context.Background(), newTestLogs(), nil); err != nil {
			t.Fatalf("SendLogs failed: %v", err)
		}
	})

	// Wait for the collector pipeline to flush.
	time.Sleep(2 * time.Second)

	collectorOutput := output.String()

	// Verify the collector received and logged the expected data.
	t.Run("verify span name in output", func(t *testing.T) {
		if !strings.Contains(collectorOutput, "test-span") {
			t.Errorf("collector output does not contain 'test-span'.\nOutput:\n%s", collectorOutput)
		}
	})

	t.Run("verify metric name in output", func(t *testing.T) {
		if !strings.Contains(collectorOutput, "test.metric") {
			t.Errorf("collector output does not contain 'test.metric'.\nOutput:\n%s", collectorOutput)
		}
	})

	t.Run("verify log body in output", func(t *testing.T) {
		if !strings.Contains(collectorOutput, "test log message") {
			t.Errorf("collector output does not contain 'test log message'.\nOutput:\n%s", collectorOutput)
		}
	})

	t.Run("verify resource attribute in output", func(t *testing.T) {
		if !strings.Contains(collectorOutput, "test-service") {
			t.Errorf("collector output does not contain 'test-service'.\nOutput:\n%s", collectorOutput)
		}
	})
}
