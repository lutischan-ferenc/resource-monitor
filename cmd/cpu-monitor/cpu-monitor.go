package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os/exec"
	"runtime"
	"time"

	"github.com/Kodeworks/golang-image-ico"
	"github.com/getlantern/systray"
	"github.com/shirou/gopsutil/cpu"
)

// main is the entry point of the application.
// It initializes the system tray with the CPU monitoring functionality.
func main() {
	systray.Run(onReady, nil)
}

// onReady sets up the system tray interface and starts monitoring CPU usage.
func onReady() {
	mWeb := systray.AddMenuItem("CPU Usage per Core V1.0", "Open the website in browser")
	systray.AddSeparator()

	// Create menu items to display CPU core usage
	var cpuMenuItems []*systray.MenuItem
	for i := 0; i < 12; i++ { // Assume there are 12 CPU cores
		menuItem := systray.AddMenuItem(fmt.Sprintf("CPU %d: 0.00%%", i), "")
		cpuMenuItems = append(cpuMenuItems, menuItem)
	}

	// Add a separator and an exit button
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")

	// Handle the exit button click
	go func() {
		for {
			select {
			case <-mWeb.ClickedCh:
				openBrowser("https://github.com/lustischan-ferenc/resource-monitor/")
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Start monitoring CPU usage in a separate goroutine
	go func() {
		for {
			// Measure CPU usage for each core
			cpuPercents, err := cpu.Percent(time.Second, true)
			if err != nil {
				fmt.Println("Error fetching CPU usage:", err)
				continue
			}

			// Calculate the average CPU usage
			var sum float64
			for _, p := range cpuPercents {
				sum += p
			}
			avg := sum / float64(len(cpuPercents))

			// Update the system tray tooltip with the average CPU usage
			systray.SetTooltip(fmt.Sprintf("CPU Avg: %.2f%%", avg))

			// Update menu items with CPU core usage percentages
			for i, p := range cpuPercents {
				if i < len(cpuMenuItems) {
					cpuMenuItems[i].SetTitle(fmt.Sprintf("CPU %d: %.2f%%", i, p))
				}
			}

			// Generate a bar chart icon based on CPU core usage
			img := generateBarChart(cpuPercents)

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

// generateBarChart creates a bar chart image representing CPU core usage.
// Each bar corresponds to a CPU core, and its height represents the usage percentage.
func generateBarChart(percents []float64) image.Image {
	width := 64 / len(percents) // Width of each bar
	height := 64                // Height of the icon
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))

	// Set the background to white
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw the bars
	for i, p := range percents {
		barHeight := int(p * float64(height) / 100) // Height of the bar
		for y := height - barHeight; y < height; y++ {
			for x := i * width; x < (i+1)*width; x++ {
				img.Set(x, y, color.Black) // Set the bar color to black
			}
		}
	}
	return img
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
