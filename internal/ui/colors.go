package ui

import "fmt"

const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorGray    = "\033[90m"
	BoldText     = "\033[1m"
)

func Bold(s string) string    { return BoldText + s + ColorReset }
func Cyan(s string) string    { return ColorCyan + s + ColorReset }
func Green(s string) string   { return ColorGreen + s + ColorReset }
func Yellow(s string) string  { return ColorYellow + s + ColorReset }
func Red(s string) string     { return ColorRed + s + ColorReset }
func Gray(s string) string    { return ColorGray + s + ColorReset }
func Magenta(s string) string { return ColorMagenta + s + ColorReset }

func Banner() {
	fmt.Println()
	fmt.Println(Cyan(Bold("╔═══════════════════════════════════════════════════════════════╗")))
	fmt.Println(Cyan(Bold("║                      ⚡ V E N T O X ⚡                        ║")))
	fmt.Println(Cyan(Bold("║     Next-Gen UEFI/GPT Bootable USB Creator for Linux & Win    ║")))
	fmt.Println(Cyan(Bold("╚═══════════════════════════════════════════════════════════════╝")))
	fmt.Println()
}

func PrintStep(step int, total int, message string) {
	fmt.Printf("\n%s %s\n", Cyan(Bold(fmt.Sprintf("[%d/%d]", step, total))), Bold(message))
}

func PrintSuccess(message string) {
	fmt.Printf("%s %s\n", Green(Bold("✔")), Green(message))
}

func PrintWarning(message string) {
	fmt.Printf("%s %s\n", Yellow(Bold("⚠")), Yellow(message))
}

func PrintError(message string) {
	fmt.Printf("%s %s\n", Red(Bold("✖")), Red(message))
}

func PrintInfo(message string) {
	fmt.Printf("%s %s\n", Blue(Bold("ℹ")), message)
}

func Blue(s string) string { return ColorBlue + s + ColorReset }
