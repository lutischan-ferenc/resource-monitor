package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os/exec"
	"runtime"
	"time"

	"github.com/Kodeworks/golang-image-ico"
	"github.com/getlantern/systray"
	"github.com/shirou/gopsutil/mem"
)

// main is the entry point of the application.
// It initializes the system tray with the memory monitoring functionality.
func main() {
	systray.Run(onReadyMem, nil)
}

// onReadyMem sets up the system tray interface and starts monitoring memory usage.
func onReadyMem() {
	mWeb := systray.AddMenuItem("Mem Monitor V1.0", "Open the website in browser")
	systray.AddSeparator()

	// Create menu items to display memory statistics
	mUsed := systray.AddMenuItem("Used: 0 MB", "")
	mFree := systray.AddMenuItem("Free: 0 MB", "")
	mCached := systray.AddMenuItem("Cached: 0 MB", "")
	mSwap := systray.AddMenuItem("Swap: 0 MB", "")

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")

	// Handle the exit button click
	go func() {
		for {
			select {
			case <-mWeb.ClickedCh:
				openBrowser("https://github.com/lutischan-ferenc/resource-monitor")
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Start monitoring memory usage in a separate goroutine
	go func() {
		for {
			// Fetch memory usage information
			memInfo, err := mem.VirtualMemory()
			if err != nil {
				fmt.Println("Error fetching memory info:", err)
				continue
			}

			// Calculate used memory percentage
			usedPercent := memInfo.UsedPercent

			// Update the system tray tooltip with memory usage percentage
			systray.SetTooltip(fmt.Sprintf("Memory Usage: %.2f%%", usedPercent))

			// Update menu items with detailed memory statistics
			mUsed.SetTitle(fmt.Sprintf("Used: %d MB", int(memInfo.Used/1024/1024)))
			mFree.SetTitle(fmt.Sprintf("Free: %d MB", int(memInfo.Free/1024/1024)))
			mCached.SetTitle(fmt.Sprintf("Cached: %d MB", int(memInfo.Cached/1024/1024)))
			mSwap.SetTitle(fmt.Sprintf("Swap: %d MB", int(memInfo.SwapTotal/1024/1024)))

			// Generate a pie chart icon based on memory usage
			img := generateCircleDiagram(usedPercent)

			// Encode the image as an ICO file
			var buf bytes.Buffer
			err = ico.Encode(&buf, img)
			if err != nil {
				// Set a blank icon if encoding fails
				systray.SetIcon([]byte{0x00})
			} else {
				// Set the generated icon in the system tray
				systray.SetIcon(buf.Bytes())
			}

			// Wait for a second before the next update
			time.Sleep(time.Second)
		}
	}()
}

// generateCircleDiagram creates a pie chart image representing memory usage.
// The used memory is displayed as a filled slice starting from the top (12 o'clock position).
func generateCircleDiagram(usedPercent float64) image.Image {
	size := 64 // Size of the icon
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Set the background to transparent
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	// Center and radius of the pie chart
	center := image.Point{size / 2, size / 2}
	radius := size / 2

	// Colors for used and free memory
	usedColor := color.RGBA{200, 200, 200, 255} // Light gray for used memory
	freeColor := color.RGBA{100, 100, 100, 255} // Dark gray for free memory

	// Calculate the end angle for the used memory slice
	endAngle := int(360 * usedPercent / 100)

	// Draw the pie chart
	for angle := 0; angle < 360; angle++ {
		for r := 0; r < radius; r++ {
			// Rotate the angle by -90 degrees to start from the top (12 o'clock position)
			rotatedAngle := float64(angle) - 90
			x := center.X + int(float64(r)*cos(rotatedAngle))
			y := center.Y + int(float64(r)*sin(rotatedAngle))

			// Fill the slice with the appropriate color based on the angle
			if angle < endAngle {
				img.Set(x, y, usedColor)
			} else {
				img.Set(x, y, freeColor)
			}
		}
	}

	return img
}

// cos calculates the cosine of an angle in degrees.
func cos(angle float64) float64 {
	return math.Cos(angle * math.Pi / 180)
}

// sin calculates the sine of an angle in degrees.
func sin(angle float64) float64 {
	return math.Sin(angle * math.Pi / 180)
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		fmt.Println("Failed to open browser:", err)
	}
}
