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
	"github.com/shirou/gopsutil/mem"
	"golang.org/x/sys/windows/registry"
)

var (
	backgroundImg *image.RGBA
	lastUsedMB    uint64
	mAutoStart    *systray.MenuItem
)

// main is the entry point of the application.
// It initializes the system tray with the memory monitoring functionality.
func main() {
	systray.Run(onReadyMem, nil)
}

// onReadyMem sets up the system tray interface and starts monitoring memory usage.
func onReadyMem() {
	mWeb := systray.AddMenuItem("Mem Monitor v1.2.0", "Open the website in browser")
	mWeb.Click(func() {
		openBrowser("https://github.com/lutischan-ferenc/resource-monitor")
	})
	systray.AddSeparator()

	// Create menu items to display memory statistics
	mUsed := systray.AddMenuItem("Used: 0 MB", "")
	mFree := systray.AddMenuItem("Free: 0 MB", "")
	mCached := systray.AddMenuItem("Cached: 0 MB", "")
	mSwap := systray.AddMenuItem("Swap: 0 MB", "")

	addAutoStartMenuOnWin()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit the application")
	mQuit.Click(func() {
		systray.Quit()
	})

	// Start monitoring memory usage in a separate goroutine
	go func() {
		ticker := time.NewTicker(time.Second) // 1 másodperces frissítési időköz
		defer ticker.Stop()

		for range ticker.C {
			// Fetch memory usage information
			memInfo, err := mem.VirtualMemory()
			if err != nil {
				fmt.Println("Error fetching memory info:", err)
				continue
			}

			// Calculate used memory in MB
			usedMB := memInfo.Used / 1024 / 1024

			// Update the system tray tooltip with memory usage percentage
			systray.SetTooltip(fmt.Sprintf("Memory Usage: %.2f%%", memInfo.UsedPercent))

			// Update menu items with detailed memory statistics
			mUsed.SetTitle(fmt.Sprintf("Used: %d MB", usedMB))
			mFree.SetTitle(fmt.Sprintf("Free: %d MB", memInfo.Free/1024/1024))
			mCached.SetTitle(fmt.Sprintf("Cached: %d MB", memInfo.Cached/1024/1024))
			mSwap.SetTitle(fmt.Sprintf("Swap: %d MB", memInfo.SwapTotal/1024/1024))

			// Generate a pie chart icon only if memory usage has changed significantly
			if usedMB != lastUsedMB {
				lastUsedMB = usedMB
				img := generateCircleDiagram(memInfo.UsedPercent)

				// Encode the image as an ICO file
				var buf bytes.Buffer
				if err := png.Encode(&buf, img); err != nil {
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

// generateCircleDiagram creates a pie chart image representing memory usage.
// The used memory is displayed as a filled slice starting from the top (12 o'clock position).
func generateCircleDiagram(usedPercent float64) image.Image {
	size := 64 // Size of the icon

	// Hozzuk létre a háttérképet, ha még nem létezik
	if backgroundImg == nil {
		backgroundImg = image.NewRGBA(image.Rect(0, 0, size, size))
		// Töltsük ki a képet transzparens háttérrel
		draw.Draw(backgroundImg, backgroundImg.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)
	}

	// Másoljuk a háttérképet egy új képre, hogy ne módosítsuk közvetlenül
	img := image.NewRGBA(backgroundImg.Bounds())
	draw.Draw(img, img.Bounds(), backgroundImg, image.Point{}, draw.Src)

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

const AUTO_START_NAME = "MemMonitor"

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