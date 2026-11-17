package profile

import "testing"

func TestLoadShippedProfile(t *testing.T) {
	p, err := Load("../../profiles/zlt_s20_1.12.7.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.Target != "192.168.1.1" {
		t.Fatalf("target = %q", p.Target)
	}
	if len(p.Payloads) == 0 {
		t.Fatal("expected payloads")
	}
	if p.Endpoints.EffectivePostPath() != "/reqproc/proc_post" {
		t.Fatalf("post path = %q", p.Endpoints.EffectivePostPath())
	}
	if p.Endpoints.EffectiveGetPath() != "/reqproc/proc_get" {
		t.Fatalf("get path = %q", p.Endpoints.EffectiveGetPath())
	}
	if p.Login.Username == "" || p.Fuzz.MaxRequests == 0 {
		t.Fatal("missing login or fuzz defaults")
	}
}

func TestBuiltinDefaults(t *testing.T) {
	p := Builtin()
	if p.UART.Baud != 115200 || p.UART.DataBits != 8 {
		t.Fatalf("unexpected UART defaults: %+v", p.UART)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}
