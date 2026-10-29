package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/domain"
)

// RenderProgressBar renders an inline terminal progress bar.
func RenderProgressBar(bytesProcessed, totalBytes int64, percent float64, speedBps float64, eta time.Duration, stepName string) {
	const barWidth = 28
	completed := int((percent / 100.0) * float64(barWidth))
	if completed > barWidth {
		completed = barWidth
	}
	if completed < 0 {
		completed = 0
	}

	bar := strings.Repeat("█", completed) + strings.Repeat("░", barWidth-completed)

	mbSpeed := speedBps / (1024 * 1024)
	etaStr := "--:--"
	if eta > 0 {
		m := int(eta.Minutes())
		s := int(eta.Seconds()) % 60
		etaStr = fmt.Sprintf("%02d:%02d", m, s)
	}

	fmt.Printf("\r  [%s] %5.1f%% (%6.1f MB/s, ETA %s) %-25s",
		Cyan(bar), percent, mbSpeed, etaStr, Gray(stepName))
}

// ClearProgressBar moves to the next line after progress finishes.
func ClearProgressBar() {
	fmt.Print("\r\033[K")
}

// ConfirmPlan displays the write plan and requests user confirmation.
func ConfirmPlan(plan *domain.WritePlan) bool {
	fmt.Println()
	fmt.Println(Cyan(Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println(Cyan(Bold("                    PROPOSED OPERATION PLAN                    ")))
	fmt.Println(Cyan(Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Printf(" • Target USB:       %s (%s)\n", Bold(plan.Target.Path), Cyan(plan.Target.FormattedSize()))
	if plan.Target.Model != "" || plan.Target.Vendor != "" {
		fmt.Printf("   Device Model:     %s %s\n", plan.Target.Vendor, plan.Target.Model)
	}
	if plan.Source != nil {
		fmt.Printf(" • Source Image:     %s (%s)\n", Bold(plan.Source.Name), Cyan(plan.Source.FormattedSize()))
		fmt.Printf("   Classification:   %s (EFI: %v, Large WIM: %v)\n", plan.Source.Classification, plan.Source.IsEFI, plan.Source.HasLargeWIM)
	}
	fmt.Printf(" • Creation Mode:    %s\n", Bold(string(plan.Mode)))
	fmt.Printf(" • Partition Scheme: %s\n", plan.Scheme)
	fmt.Printf(" • Target FS:        %s (Label: %s)\n", plan.Filesystem, plan.VolumeLabel)
	if plan.BypassWindows11 {
		fmt.Printf(" • Win11 Bypasses:   %s\n", Green("Enabled (TPM 2.0, Secure Boot, RAM, CPU, Storage, NRO)"))
	}
	if plan.VerifyAfterWrite {
		fmt.Printf(" • Data Verify:      %s\n", Green("Enabled (SHA-256 Checksum)"))
	}
	if plan.DryRun {
		fmt.Printf(" • Dry Run:          %s\n", Yellow(Bold("ENABLED (Simulated execution without disk writes)")))
	}

	fmt.Println(Cyan(Bold("───────────────────────────────────────────────────────────────")))
	fmt.Println(Bold(" Planned Execution Steps:"))
	for i, step := range plan.Steps {
		fmt.Printf("   [%d] %s\n", i+1, step)
	}
	fmt.Println(Cyan(Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println()

	if plan.DryRun {
		resp, _ := Prompt(fmt.Sprintf("Proceed with %s simulation? [Y/n]", Bold("DRY-RUN")), "Y")
		return strings.ToLower(resp) == "y" || strings.ToLower(resp) == "yes"
	}

	fmt.Println(Red(Bold("===============================================================")))
	fmt.Println(Red(Bold(" ⚠️  CRITICAL WARNING: ALL DATA ON THE USB WILL BE ERASED!   ⚠️ ")))
	fmt.Println(Red(Bold("===============================================================")))
	resp, _ := Prompt(fmt.Sprintf("Type %s to proceed and write bootable image", Bold("YES")), "NO")
	return strings.ToUpper(strings.TrimSpace(resp)) == "YES"
}
