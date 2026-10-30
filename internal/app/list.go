package app

import (
	"context"
	"fmt"
	"time"

	"github.com/0xzahed/ventox/internal/iso"
	"github.com/0xzahed/ventox/internal/ui"
)

const listTimeout = 10 * time.Second

type listCommand struct{ deps Deps }

func (c listCommand) Run(ctx context.Context, cfg Config) error {
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	// Scan failures are non-fatal: partial results are still useful to the user.
	devices, devErr := c.deps.Devices.ListDevices(ctx)
	images, imgErr := iso.DiscoverImages(ctx)

	if cfg.JSONOutput {
		return writeJSON(c.deps.Stdout, map[string]any{"devices": devices, "images": images})
	}
	if devErr != nil {
		ui.PrintWarning(fmt.Sprintf("Device scan returned warning: %v", devErr))
	}
	if imgErr != nil {
		ui.PrintWarning(fmt.Sprintf("Image discovery warning: %v", imgErr))
	}

	ui.Banner()
	fmt.Println(ui.Cyan(ui.Bold("--- CONNECTED USB STORAGE DEVICES ---")))
	usb := usbDevices(devices)
	for i, d := range usb {
		fmt.Printf("  %s %s\n", ui.Cyan(fmt.Sprintf("[%d]", i+1)), d.DisplayString())
	}
	if len(usb) == 0 {
		ui.PrintWarning("No USB storage drives currently connected.")
	}

	fmt.Println()
	fmt.Println(ui.Cyan(ui.Bold("--- DETECTED ISO DISK IMAGES ---")))
	if len(images) == 0 {
		ui.PrintWarning("No .iso or .img files found in user Downloads or Desktop.")
	}
	for i, img := range images {
		fmt.Printf("  %s %s %s\n     %s\n",
			ui.Cyan(fmt.Sprintf("[%d]", i+1)),
			ui.Bold(img.Name),
			ui.Cyan(fmt.Sprintf("(%s)", img.FormattedSize())),
			ui.Gray(img.Path),
		)
	}
	fmt.Println()
	return nil
}
