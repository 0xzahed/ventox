package detector

import (
	"testing"
)

func TestDetectUSBDisks(t *testing.T) {
	disks, err := DetectUSBDisks()
	if err != nil {
		t.Logf("DetectUSBDisks() returned info: %v", err)
		return
	}
	t.Logf("Disks found: %d", len(disks))
	for _, d := range disks {
		t.Logf("Found USB: %s", d.DisplayString())
	}
}

func TestFindISOFallbacks(t *testing.T) {
	isos, err := FindISOFallbacks()
	if err != nil {
		t.Fatalf("FindISOFallbacks() failed: %v", err)
	}
	t.Logf("ISOs found: %d", len(isos))
	for _, iso := range isos {
		t.Logf("Found ISO: %s (%s) at %s", iso.Name, iso.FormattedSize, iso.Path)
	}
}
