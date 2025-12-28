package main

import (
	"context"
	"log"
	"time"

	"github.com/vitaliitsarov/osciris"
	fp "github.com/vitaliitsarov/fingerprint-injector-go"
)

func main() {
	ctx := context.Background()

	// Создаем кастомный fingerprint
	fingerprint := &fp.Fingerprint{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		Platform:  "Win32",
		Vendor:    "Google Inc.",
		Language:  "ru-RU",
		Languages: []string{"ru-RU", "ru", "en"},
		Screen: &fp.Screen{
			Width:            1920,
			Height:           1080,
			ColorDepth:       24,
			DevicePixelRatio: 1.0,
		},
		Timezone: &fp.Timezone{
			ID:     "Europe/Moscow",
			Offset: -180,
		},
		WebGL: &fp.WebGL{
			Vendor:   "Google Inc. (NVIDIA)",
			Renderer: "ANGLE (NVIDIA GeForce RTX 3080)",
		},
		Canvas: &fp.Canvas{
			Noise: 0.02,
		},
		WebRTC: &fp.WebRTC{
			Disable: true,
		},
		HardwareConcurrency: 16,
		DeviceMemory:        32,
	}

	// Настраиваем опции браузера
	options := &osciris.BrowserOptions{
		Headless:    false,
		Stealth:     true,
		Timeout:     30 * time.Second,
		WindowWidth: 1920,
		WindowHeight: 1080,
		Fingerprint: fingerprint,
		UserDataDir: "./chrome-data",
	}

	// Создаем браузер
	browser, err := osciris.NewBrowser(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	defer browser.Close()

	// Создаем страницу
	page := browser.NewPage()

	// Переходим на сайт для проверки fingerprint
	log.Println("Navigating to browserleaks.com...")
	err = page.Navigate("https://browserleaks.com/canvas")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Page loaded. Check fingerprint in browser.")
	log.Println("Press Enter to close...")
	
	// Ждем немного для просмотра
	time.Sleep(10 * time.Second)
}

