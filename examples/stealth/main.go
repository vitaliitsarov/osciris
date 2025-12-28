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

	// Используем готовый preset и настраиваем для максимальной защиты
	fingerprint := fp.NewChrome119Windows11()
	fingerprint.WebRTC.Disable = true
	fingerprint.Canvas.Noise = 0.05

	// Настройки для stealth режима
	options := &osciris.BrowserOptions{
		Headless:    false,
		Stealth:     true,
		Timeout:     30 * time.Second,
		WindowWidth: 1920,
		WindowHeight: 1080,
		Fingerprint: fingerprint,
		UserDataDir: "./chrome-data",
		Flags: []string{
			"disable-blink-features=AutomationControlled",
			"exclude-switches=enable-automation",
		},
	}

	browser, err := osciris.NewBrowser(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	defer browser.Close()

	page := browser.NewPage()

	// Тестируем на сайте проверки ботов
	log.Println("Testing on bot detection site...")
	err = page.Navigate("https://bot.sannysoft.com/")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Page loaded. Check detection results in browser.")
	log.Println("Waiting 15 seconds for review...")
	time.Sleep(15 * time.Second)
}

