package main

import (
	"context"
	"log"

	"github.com/playy/osciris"
)

func main() {
	ctx := context.Background()

	// Создаем браузер с настройками по умолчанию
	browser, err := osciris.NewBrowser(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer browser.Close()

	// Создаем страницу
	page := browser.NewPage()

	// Переходим на сайт
	log.Println("Navigating to example.com...")
	err = page.Navigate("https://example.com")
	if err != nil {
		log.Fatal(err)
	}

	// Получаем заголовок страницы
	var title string
	err = page.Title(&title)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Page title: %s", title)

	// Получаем URL
	var url string
	err = page.URL(&url)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Current URL: %s", url)

	// Получаем текст элемента
	var text string
	err = page.Text("h1", &text)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("H1 text: %s", text)
}

