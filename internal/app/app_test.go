package app

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xzahed/ventox/internal/domain"
)

type stubProvider struct{ devs []domain.BlockDevice }

func (s stubProvider) ListDevices(context.Context) ([]domain.BlockDevice, error) { return s.devs, nil }
func (s stubProvider) GetDevice(_ context.Context, p string) (domain.BlockDevice, error) {
	for _, d := range s.devs {
		if d.Path == p {
			return d, nil
		}
	}
	return domain.BlockDevice{}, domain.ErrDeviceRemoved
}
func (s stubProvider) GetRawPath(p string) string { return p }

func TestLookup(t *testing.T) {
	a := New(Deps{Devices: stubProvider{}})
	for _, name := range []string{"", "list", "LS", "format", "wipe"} {
		if _, err := a.Lookup(name); err != nil {
			t.Errorf("Lookup(%q) = %v", name, err)
		}
	}
	if _, err := a.Lookup("bogus"); !errors.Is(err, ErrUnknownCommand) {
		t.Errorf("want ErrUnknownCommand, got %v", err)
	}
}

func TestRequiresRoot(t *testing.T) {
	if RequiresRoot("list") || RequiresRoot("LS") {
		t.Error("list must not require root")
	}
	if !RequiresRoot("format") || !RequiresRoot("") {
		t.Error("format and boot require root")
	}
}

func TestParseFilesystem(t *testing.T) {
	cases := map[string]domain.TargetFilesystem{
		"FAT32": domain.FSFAT32, " ntfs ": domain.FSNTFS, "ext4": domain.FSEXT4, "": domain.FSExFAT, "zzz": domain.FSExFAT,
	}
	for in, want := range cases {
		if got := parseFilesystem(in); got != want {
			t.Errorf("parseFilesystem(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestListJSON(t *testing.T) {
	var out bytes.Buffer
	a := New(Deps{Devices: stubProvider{devs: []domain.BlockDevice{{Path: "/dev/sdz", IsRemovable: true}}}, Stdout: &out})
	if err := a.Run(context.Background(), Config{Command: "list", JSONOutput: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "/dev/sdz") {
		t.Errorf("device missing from JSON output: %s", out.String())
	}
}

func TestFormatDryRunJSON(t *testing.T) {
	var out bytes.Buffer
	dev := domain.BlockDevice{Path: "/dev/sdz", SizeBytes: 8 << 30, IsRemovable: true}
	a := New(Deps{Devices: stubProvider{devs: []domain.BlockDevice{dev}}, Stdout: &out})
	err := a.Run(context.Background(), Config{
		Command: "format", DevicePath: "/dev/sdz", FS: "fat32", Label: "T",
		DryRun: true, AutoConfirm: true, JSONOutput: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "DRY-RUN") {
		t.Errorf("expected dry-run journal, got %s", out.String())
	}
}
