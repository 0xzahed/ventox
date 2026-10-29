package native

import (
	"strings"
	"testing"
)

func TestGenerateAutoUnattendXML(t *testing.T) {
	xml := GenerateAutoUnattendXML()
	if !strings.Contains(xml, "BypassTPMCheck") {
		t.Errorf("Expected autounattend.xml to contain BypassTPMCheck")
	}
	if !strings.Contains(xml, "BypassSecureBootCheck") {
		t.Errorf("Expected autounattend.xml to contain BypassSecureBootCheck")
	}
	if !strings.Contains(xml, "BypassNRO") {
		t.Errorf("Expected autounattend.xml to contain BypassNRO")
	}
}
