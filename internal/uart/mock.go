package uart

import (
	"strings"
	"time"
)

// MockTransport is an in-memory Transport for testing and --simulate mode.
// It answers AT commands with realistic Tozed responses.
type MockTransport struct {
	written []string
}

// NewMock creates a mock serial transport.
func NewMock() *MockTransport { return &MockTransport{} }

// Written returns the commands sent so far.
func (m *MockTransport) Written() []string { return append([]string(nil), m.written...) }

// WriteString records a command.
func (m *MockTransport) WriteString(s string) (int, error) {
	m.written = append(m.written, s)
	return len(s), nil
}

// ReadUntil returns a canned response based on the last command.
func (m *MockTransport) ReadUntil(expect string, timeout time.Duration) (string, error) {
	last := ""
	if len(m.written) > 0 {
		last = strings.ToUpper(strings.TrimSpace(m.written[len(m.written)-1]))
	}
	switch {
	case strings.Contains(last, "ATI"):
		return "Manufacturer: TOZED\r\nModel: ZLT S20\r\nOK\r\n", nil
	case strings.Contains(last, "AT+CSQ"):
		return "+CSQ: 20,99\r\nOK\r\n", nil
	case strings.Contains(last, "AT+CLCK"), strings.Contains(last, "AT^NVW"), strings.Contains(last, "AT^RESET"):
		return "OK\r\n", nil
	default:
		return "OK\r\n", nil
	}
}
