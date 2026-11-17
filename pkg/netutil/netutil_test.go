package netutil

import (
	"net"
	"testing"
	"time"
)

func TestScanPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	if !ScanPort("127.0.0.1", port, time.Second) {
		t.Fatal("open port reported closed")
	}
	if ScanPort("127.0.0.1", port+1000, 200*time.Millisecond) {
		t.Fatal("closed port reported open")
	}
}

func TestScanPorts(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	results := ScanPorts("127.0.0.1", []int{port, port + 1000}, 2, time.Second)
	if !results[port] {
		t.Fatalf("open port missing: %v", results)
	}
	if results[port+1000] {
		t.Fatalf("closed port reported open: %v", results)
	}
}

func TestLocalIP(t *testing.T) {
	ip, err := LocalIP()
	if err != nil {
		t.Skipf("no route available: %v", err)
	}
	if ip == "" || ip == "0.0.0.0" {
		t.Fatalf("LocalIP = %q", ip)
	}
}
