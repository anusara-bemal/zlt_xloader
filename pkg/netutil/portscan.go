// Package netutil provides small network helpers shared by commands.
package netutil

import (
	"net"
	"strconv"
	"sync"
	"time"
)

// ScanPort reports whether a TCP port accepts connections.
func ScanPort(host string, port int, timeout time.Duration) bool {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ScanPorts scans ports concurrently and returns a map of port -> open.
func ScanPorts(host string, ports []int, workers int, timeout time.Duration) map[int]bool {
	if workers < 1 {
		workers = len(ports)
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	results := make(map[int]bool)
	var mu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				open := ScanPort(host, p, timeout)
				mu.Lock()
				results[p] = open
				mu.Unlock()
			}
		}()
	}
	for _, p := range ports {
		jobs <- p
	}
	close(jobs)
	wg.Wait()

	return results
}
