package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/lutischan-ferenc/systray"
	"github.com/shirou/gopsutil/cpu"
	"golang.org/x/sys/windows/registry"
)

var (
	backgroundImg *image.RGBA
	mAutoStart    *systray.MenuItem
)

// main is the entry point of the application.
// It initializes the system tray with the CPU monitoring functionality.
func main() {
	systray.Run(onReady, nil)
}

// onReady sets up the system tray interface and starts monitoring CPU usage.
func onReady() {
	cpuPercents, err := cpu.Percent(time.Millisecond*1250, true)
	if err != nil {
		fmt.Println("Error fetching CPU usage:", err)
	}
	mWeb := systray.AddMenuItem("CPU Usage per Core v1.2.0", "Open the website in browser")
	mWeb.Click(func() {
		openBrowser("https://github.com/lutischan-ferenc/resource-monitor")
	})
	systray.AddSeparator()

	var cpuMenuItems []*systray.MenuItem
	for i := 0; i < len(cpuPercents); i++ {
		menuItem := systray.AddMenuItem(fmt.Sprintf("CPU %d: 0.00%%", i+1), "")
		cpuMenuItems = append(cpuMenuItems, menuItem)
	}

	addAutoStartMenuOnWin()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")
	mQuit.Click(func() {
		systray.Quit()
	})

	// Start monitoring CPU usage in a separate goroutine
	go func() {
		ticker := time.NewTicker(time.Millisecond * 1250)
		defer ticker.Stop()

		var lastPercents []float64

		for range ticker.C {
			// Measure CPU usage for each core
			cpuPercents, err := cpu.Percent(time.Millisecond*1250, true)
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
					newTitle := fmt.Sprintf("CPU %d: %.2f%%", i+1, p)
					cpuMenuItems[i].SetTitle(newTitle)
				}
			}

			// Generate a bar chart icon only if CPU usage has changed significantly
			if cpuUsageChanged(lastPercents, cpuPercents) {
				lastPercents = cpuPercents
				img := generateBarChart(cpuPercents)

				// Encode the image as an ICO file
				var buf bytes.Buffer
				err = png.Encode(&buf, img)
				if err != nil {
					fmt.Println("Failed to encode icon, set empty icon")
					// Set a blank icon if encoding fails
					systray.SetIconFromMemory([]byte{0x00})
				} else {
					// Set the generated icon in the system tray
					systray.SetIconFromMemory(buf.Bytes())
				}
			}
		}
	}()
}

// generateBarChart creates a bar chart image representing CPU core usage.
// Each bar corresponds to a CPU core, and its height represents the usage percentage.
func generateBarChart(percents []float64) image.Image {
	numCores := len(percents)
	maxWidth := 64  // Maximális ikon szélesség
	barSpacing := 1 // Szóköz a sávok között

	// Csoportosítás, ha több mint 64 mag van
	groupSize := 1
	if numCores > maxWidth {
		groupSize = numCores / maxWidth
		if numCores%maxWidth != 0 {
			groupSize++
		}
	}

	// Számítsuk ki a csoportok átlagos használatát
	numGroups := (numCores + groupSize - 1) / groupSize
	groupPercents := make([]float64, numGroups)
	for i := 0; i < numCores; i++ {
		groupIndex := i / groupSize
		groupPercents[groupIndex] += percents[i]
	}
	for i := range groupPercents {
		groupPercents[i] /= float64(groupSize)
	}

	// Számítsuk ki a sávok szélességét
	barWidth := (maxWidth - (numGroups-1)*barSpacing) / numGroups
	if barWidth < 1 {
		barWidth = 1 // Legalább 1 pixel széles legyen minden sáv
	}

	// Kép mérete
	imgWidth := barWidth*numGroups + (numGroups-1)*barSpacing
	imgHeight := 64

	// Hozzuk létre a háttérképet, ha még nem létezik
	if backgroundImg == nil {
		backgroundImg = image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
		// Töltsük ki a képet transzparens háttérrel
		draw.Draw(backgroundImg, backgroundImg.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)
	}

	// Másoljuk a háttérképet egy új képre, hogy ne módosítsuk közvetlenül
	img := image.NewRGBA(backgroundImg.Bounds())
	draw.Draw(img, img.Bounds(), backgroundImg, image.Point{}, draw.Src)

	// Rajzoljuk meg a sávokat
	for i, p := range groupPercents {
		barHeight := int(p * float64(imgHeight) / 100)
		xStart := i * (barWidth + barSpacing)
		xEnd := xStart + barWidth

		for y := imgHeight - barHeight; y < imgHeight; y++ {
			for x := xStart; x < xEnd; x++ {
				if barHeight > 32 {
					img.Set(x, y, color.RGBA{139, 0, 0, 255})
				} else {
					img.Set(x, y, color.RGBA{100, 100, 100, 255})
				}
			}
		}
	}

	return img
}

// cpuUsageChanged checks if the CPU usage has changed significantly.
func cpuUsageChanged(old, new []float64) bool {
	if len(old) != len(new) {
		return true
	}
	for i := range new {
		if math.Abs(old[i]-new[i]) > 0.1 { // 0.1% tolerancia
			return true
		}
	}
	return false
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

const AUTO_START_NAME = "CpuMonitor"

func addAutoStartMenuOnWin() {
	// Add auto-start menu item for Windows only
	if runtime.GOOS == "windows" {
		systray.AddSeparator()
		mAutoStart = systray.AddMenuItemCheckbox("Start on System Startup", "Auto-start on System Startup", false)
		// Check the current state of auto-start in the registry
		if isAutoStartEnabled() {
			mAutoStart.Check()
		}

		mAutoStart.Click(func() {
			if mAutoStart.Checked() {
				// Disable auto-start
				if err := setAutoStart(false); err != nil {
					fmt.Println("Failed to disable auto-start:", err)
				} else {
					fmt.Println("Auto-start disabled")
					mAutoStart.Uncheck()
				}
			} else {
				// Enable auto-start
				if err := setAutoStart(true); err != nil {
					fmt.Println("Failed to enable auto-start:", err)
				} else {
					fmt.Println("Auto-start enabled")
					mAutoStart.Check()
				}
			}
		})
	}
}

// setAutoStart sets or removes the application from the Windows startup registry.
func setAutoStart(enable bool) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("auto-start is only supported on Windows")
	}

	// Get the path to the current executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Open the registry key for auto-start programs
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %v", err)
	}
	defer key.Close()

	// Set or remove the auto-start entry
	if enable {
		if err := key.SetStringValue(AUTO_START_NAME, exePath); err != nil {
			return fmt.Errorf("failed to set registry value: %v", err)
		}
	} else {
		if err := key.DeleteValue(AUTO_START_NAME); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("failed to delete registry value: %v", err)
		}
	}

	return nil
}

// isAutoStartEnabled checks if the application is set to auto-start in the Windows registry.
func isAutoStartEnabled() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// Get the path to the current executable
	exePath, err := os.Executable()
	if err != nil {
		fmt.Println("Failed to get executable path:", err)
		return false
	}

	// Open the registry key for auto-start programs
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		fmt.Println("Failed to open registry key:", err)
		return false
	}
	defer key.Close()

	// Check if the registry value exists and matches the current executable path
	value, _, err := key.GetStringValue(AUTO_START_NAME)
	if err != nil {
		if err == registry.ErrNotExist {
			return false
		}
		fmt.Println("Failed to read registry value:", err)
		return false
	}

	return value == exePath
}
