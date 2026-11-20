package uart

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/anusara-bemal/zlt_xloader/internal/profile"
)

type fakeTransport struct {
	written []string
	answer  string
	err     error
}

func (f *fakeTransport) WriteString(s string) (int, error) {
	f.written = append(f.written, s)
	return len(s), f.err
}

func (f *fakeTransport) ReadUntil(expect string, timeout time.Duration) (string, error) {
	return f.answer, f.err
}

func TestEngineSend(t *testing.T) {
	fake := &fakeTransport{answer: "AT+CSQ\r\n+CSQ: 20,99\r\nOK\r\n"}
	e := NewEngine(fake, false)
	got, err := e.Send("AT+CSQ", "OK", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "OK") {
		t.Fatalf("response = %q", got)
	}
	if len(fake.written) != 1 || fake.written[0] != "AT+CSQ\r\n" {
		t.Fatalf("written = %q", fake.written)
	}
}

func TestEngineSendError(t *testing.T) {
	fake := &fakeTransport{err: errors.New("port closed")}
	e := NewEngine(fake, false)
	if _, err := e.Send("AT", "OK", time.Second); err == nil {
		t.Fatal("expected write error")
	}
}

func TestEngineDryRun(t *testing.T) {
	e := NewEngine(nil, true)
	if _, err := e.Send("AT", "OK", time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestMockTransport(t *testing.T) {
	mock := NewMock()
	e := NewEngine(mock, false)
	resp, err := e.Send("AT+CSQ", "OK", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, "+CSQ") {
		t.Fatalf("mock response = %q", resp)
	}
	if len(mock.Written()) != 1 {
		t.Fatalf("written = %v", mock.Written())
	}
}

func TestEngineRun(t *testing.T) {
	fake := &fakeTransport{answer: "OK\r\n"}
	e := NewEngine(fake, false)
	cmds := []profile.UARTCommand{{Name: "probe", Command: "AT", Expect: "OK"}}
	results := e.Run(cmds, time.Second)
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("results = %+v", results)
	}
}
