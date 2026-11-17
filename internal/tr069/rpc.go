package tr069

import "fmt"

// Parameter is a single CWMP parameter value.
type Parameter struct {
	Name  string
	Value string
	Type  string // XSD type, defaults to xsd:string
}

// InformResponse acknowledges a CPE Inform.
func InformResponse() string {
	return `<cwmp:InformResponse><MaxEnvelopes>1</MaxEnvelopes></cwmp:InformResponse>`
}

// SetParameterValues pushes configuration parameters to the CPE.
func SetParameterValues(params []Parameter, key string) string {
	if len(params) == 0 {
		return ""
	}
	body := fmt.Sprintf(`<cwmp:SetParameterValues><ParameterList SOAP-ENC:arrayType="cwmp:ParameterValueStruct[%d]">`, len(params))
	for _, p := range params {
		typ := p.Type
		if typ == "" {
			typ = "xsd:string"
		}
		body += fmt.Sprintf(
			`<ParameterValueStruct><Name>%s</Name><Value xsi:type="%s">%s</Value></ParameterValueStruct>`,
			Escape(p.Name), Escape(typ), Escape(p.Value),
		)
	}
	body += `</ParameterList><ParameterKey>` + Escape(key) + `</ParameterKey></cwmp:SetParameterValues>`
	return body
}

// Download asks the CPE to fetch a firmware/config image from url.
func Download(url, username, password string, fileSize int64) string {
	return fmt.Sprintf(
		`<cwmp:Download><CommandKey>zlt-unlocker</CommandKey>`+
			`<FileType>1 Firmware Upgrade Image</FileType>`+
			`<URL>%s</URL><Username>%s</Username><Password>%s</Password>`+
			`<FileSize>%d</FileSize></cwmp:Download>`,
		Escape(url), Escape(username), Escape(password), fileSize,
	)
}

// Reboot asks the CPE to reboot after the configuration push.
func Reboot() string {
	return `<cwmp:Reboot><CommandKey>zlt-unlocker</CommandKey></cwmp:Reboot>`
}
