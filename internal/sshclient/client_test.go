package sshclient

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startTestServer runs a minimal in-process SSH server that accepts a single
// password and answers exec requests with the command it received.
func startTestServer(t *testing.T) int {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if string(password) == "secret" {
				return nil, nil
			}
			return nil, fmt.Errorf("bad password")
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serve(conn, cfg)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func serve(conn net.Conn, cfg *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for ch := range chans {
		if ch.ChannelType() != "session" {
			_ = ch.Reject(ssh.UnknownChannelType, "session only")
			continue
		}
		channel, requests, err := ch.Accept()
		if err != nil {
			continue
		}
		go func(channel ssh.Channel, requests <-chan *ssh.Request) {
			defer channel.Close()
			for req := range requests {
				switch req.Type {
				case "exec":
					var payload struct{ Command string }
					if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
						req.Reply(false, nil)
						continue
					}
					req.Reply(true, nil)
					fmt.Fprintf(channel, "echo:%s", payload.Command)
					_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
					return
				default:
					req.Reply(false, nil)
				}
			}
		}(channel, requests)
	}
}

func TestDialAndRun(t *testing.T) {
	port := startTestServer(t)

	client, err := Dial("127.0.0.1", port, "admin", "secret", 3*time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	out, err := client.Run("nv get sim_plmn", 3*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "echo:nv get sim_plmn") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestDialRejectsBadPassword(t *testing.T) {
	port := startTestServer(t)
	if _, err := Dial("127.0.0.1", port, "admin", "wrong", 3*time.Second); err == nil {
		t.Fatal("expected authentication failure")
	}
}
