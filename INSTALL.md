# Установка зависимостей

## Требования

- Go 1.21 или выше
- Chrome/Chromium браузер

## Установка библиотеки

```bash
go get github.com/playy/osciris
```

## Установка зависимостей

Библиотека требует следующие зависимости:

- `github.com/chromedp/chromedp` - автоматически установится
- `github.com/vitaliitsarov/fingerprint-injector-go` - требуется установка

### Установка fingerprint-injector-go

Если библиотека доступна через go get:

```bash
go get github.com/vitaliitsarov/fingerprint-injector-go@latest
```

Если библиотека не доступна через go get, используйте replace директиву в `go.mod`:

```go
replace github.com/vitaliitsarov/fingerprint-injector-go => /path/to/fingerprint-injector-go
```

Или если библиотека находится в другом репозитории:

```go
replace github.com/vitaliitsarov/fingerprint-injector-go => github.com/your-fork/fingerprint-injector-go@main
```

## Проверка установки

```bash
go mod tidy
go build ./...
```

## Запуск примеров

```bash
# Базовый пример
cd examples/basic
go run main.go

# Кастомный fingerprint
cd examples/custom
go run main.go

# Stealth режим
cd examples/stealth
go run main.go
```

