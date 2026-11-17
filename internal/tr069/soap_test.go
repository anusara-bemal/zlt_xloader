package tr069

import (
	"strings"
	"testing"
)

func TestEnvelope(t *testing.T) {
	msg := Envelope(Header("42"), InformResponse())
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`xmlns:cwmp="urn:dslforum-org:cwmp-1-0"`,
		`<cwmp:ID soap:mustUnderstand="1">42</cwmp:ID>`,
		`<cwmp:InformResponse>`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("envelope missing %q", want)
		}
	}
}

func TestEscape(t *testing.T) {
	if got := Escape(`a<b&c"d`); got != "a&lt;b&amp;c&#34;d" {
		t.Fatalf("Escape = %q", got)
	}
}

func TestSetParameterValues(t *testing.T) {
	msg := SetParameterValues([]Parameter{
		{Name: "InternetGatewayDevice.ManagementServer.URL", Value: "http://192.168.1.2:7547/acs"},
	}, "key")
	if !strings.Contains(msg, `SOAP-ENC:arrayType="cwmp:ParameterValueStruct[1]"`) {
		t.Fatal("missing arrayType")
	}
	if !strings.Contains(msg, `xsi:type="xsd:string"`) {
		t.Fatal("missing xsi type")
	}
	if !strings.Contains(msg, `<ParameterKey>key</ParameterKey>`) {
		t.Fatal("missing parameter key")
	}
}
