module github.com/playy/osciris

go 1.25

require (
	github.com/chromedp/cdproto v0.0.0-20250803210736-d308e07a266d
	github.com/chromedp/chromedp v0.14.2
	github.com/vitaliitsarov/fingerprint-injector-go v0.0.0-20240101000000-000000000000
)

// ВАЖНО: Библиотека fingerprint-injector-go может требовать ручной установки.
// Если go get не работает, используйте replace директиву:
//
// Для локальной разработки:
// replace github.com/vitaliitsarov/fingerprint-injector-go => ../fingerprint-injector-go
//
// Или для форка:
// replace github.com/vitaliitsarov/fingerprint-injector-go => github.com/your-fork/fingerprint-injector-go@main

require (
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20250910080747-cc2cfa0554c3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
)

// Для использования библиотеки fingerprint-injector-go, убедитесь что она доступна:
// go get github.com/vitaliitsarov/fingerprint-injector-go@latest
// или используйте replace для локальной разработки:
// replace github.com/vitaliitsarov/fingerprint-injector-go => ../fingerprint-injector-go

replace github.com/vitaliitsarov/fingerprint-injector-go => github.com/vitaliitsarov/fingerprint-injector-go v1.0.0
