// Package tr069 implements a minimal TR-069 ACS server for controlled
// configuration pushes against a device the operator owns.
package tr069

import (
	"bytes"
	"encoding/xml"
)

const soapNamespaces = `xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" ` +
	`xmlns:cwmp="urn:dslforum-org:cwmp-1-0" ` +
	`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
	`xmlns:xsd="http://www.w3.org/2001/XMLSchema" ` +
	`xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/"`

// Envelope wraps a SOAP body with the standard CWMP namespaces.
func Envelope(header, body string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<soap:Envelope ` + soapNamespaces + ` soap:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">` +
		`<soap:Header>` + header + `</soap:Header>` +
		`<soap:Body>` + body + `</soap:Body>` +
		`</soap:Envelope>`
}

// Header builds the CWMP ID header.
func Header(id string) string {
	return `<cwmp:ID soap:mustUnderstand="1">` + Escape(id) + `</cwmp:ID>`
}

// Escape XML-escapes a value for safe inclusion in a SOAP message.
func Escape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
