package server_test

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/aof"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

func TestServerReplaysAOFOnStartup(t *testing.T) {
	dir := t.TempDir()
	aofDir := filepath.Join(dir, aof.DefaultAppendDirName)
	if err := os.MkdirAll(aofDir, 0o755); err != nil {
		t.Fatalf("mkdir aof dir: %v", err)
	}

	incrName := aof.DefaultAppendFilename + ".1.incr.aof"
	if err := os.WriteFile(filepath.Join(aofDir, aof.DefaultAppendFilename+".manifest"), []byte("file "+incrName+" seq 1 type i\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aofDir, incrName), []byte(resp.Array([]string{"SET", "foo", "bar"})), 0o644); err != nil {
		t.Fatalf("write incr file: %v", err)
	}

	srv, err := server.New("127.0.0.1:0", "master", "",
		server.WithDir(dir),
		server.WithConfigOverrides(map[string]string{
			"appendonly":    "yes",
			"appenddirname": aof.DefaultAppendDirName,
			"appendfilename": aof.DefaultAppendFilename,
		}),
	)
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	t.Cleanup(func() { srv.Close() })
	go srv.Run()

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	reader := bufio.NewReader(conn)
	conn.SetDeadline(time.Now().Add(2 * time.Second))

	if _, err := conn.Write([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")); err != nil {
		t.Fatalf("write GET: %v", err)
	}
	header, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read header: %v", err)
	}
	if header != "$3\r\n" {
		t.Fatalf("expected $3\\r\\n, got %q", header)
	}
	body, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if body != "bar\r\n" {
		t.Fatalf("expected restored value, got %q", body)
	}
}