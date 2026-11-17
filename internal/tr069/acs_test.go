package tr069

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestACSConversation(t *testing.T) {
	s := &Server{
		RPC:        "set-params",
		ParamName:  "X_UNLOCK_TEST",
		ParamValue: "1",
		DryRun:     false,
	}
	ts := httptest.NewServer(s)
	defer ts.Close()

	inform := `<?xml version="1.0"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><cwmp:Inform xmlns:cwmp="urn:dslforum-org:cwmp-1-0"/></soap:Body></soap:Envelope>`
	resp, err := http.Post(ts.URL+"/acs", "text/xml", strings.NewReader(inform))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "InformResponse") {
		t.Fatalf("first response = %q", body)
	}

	resp2, err := http.Post(ts.URL+"/acs", "text/xml", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !strings.Contains(string(body2), "SetParameterValues") {
		t.Fatalf("second response = %q", body2)
	}
	if !strings.Contains(string(body2), "X_UNLOCK_TEST") {
		t.Fatalf("parameter missing from response: %q", body2)
	}

	resp3, err := http.Post(ts.URL+"/acs", "text/xml", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	body3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if strings.Contains(string(body3), "SetParameterValues") {
		t.Fatalf("RPC was pushed more than once: %q", body3)
	}
}

func TestACSDryRunSuppressesRPC(t *testing.T) {
	s := &Server{RPC: "reboot", DryRun: true}
	ts := httptest.NewServer(s)
	defer ts.Close()

	http.Post(ts.URL+"/acs", "text/xml", strings.NewReader("Inform"))
	resp, err := http.Post(ts.URL+"/acs", "text/xml", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "Reboot") {
		t.Fatalf("dry-run pushed a Reboot RPC: %q", body)
	}
}
