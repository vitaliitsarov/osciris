# Osciris

Библиотека-обертка над chromedp для Go с интеграцией fingerprint injection. Упрощает работу с автоматизацией браузера и защитой от детекции.

## 🚀 Возможности

- ✅ Простой и удобный API для работы с chromedp
- ✅ Интеграция с [fingerprint-injector-go](https://github.com/vitaliitsarov/fingerprint-injector-go)
- ✅ Поддержка stealth режима
- ✅ Готовые пресеты fingerprint
- ✅ Упрощенная работа со страницами и элементами
- ✅ Автоматическое применение fingerprint при создании браузера
- ✅ **Работа с удаленным браузером** через `NewRemoteAllocator`
- ✅ **Управление вкладками**: открытие, подключение к существующим, закрытие
- ✅ **Избежание создания дублирующих вкладок** при переподключении

## 📦 Установка

```bash
go get github.com/vitaliitsarov/osciris
```

## 🎯 Быстрый старт

### Базовое использование

```go
package main

import (
    "context"
    "log"
    
    "github.com/vitaliitsarov/osciris"
    fp "github.com/vitaliitsarov/fingerprint-injector-go"
)

func main() {
    ctx := context.Background()
    
    // Создаем браузер с fingerprint
    browser, err := osciris.NewBrowser(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer browser.Close()
    
    // Создаем страницу
    page := browser.NewPage()
    
    // Переходим на сайт
    err = page.Navigate("https://example.com")
    if err != nil {
        log.Fatal(err)
    }
    
    // Получаем заголовок
    var title string
    err = page.Title(&title)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Title: %s", title)
}
```

### Кастомные опции

```go
options := &osciris.BrowserOptions{
    Headless:    false,
    Stealth:     true,
    Timeout:     30 * time.Second,
    WindowWidth: 1920,
    WindowHeight: 1080,
    Fingerprint: fp.NewChrome119Windows11(),
    UserDataDir: "./chrome-data",
}

browser, err := osciris.NewBrowser(ctx, options)
```

### Кастомный fingerprint

```go
fingerprint := &fp.Fingerprint{
    UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
    Platform:  "Win32",
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
}

options := osciris.DefaultBrowserOptions()
options.Fingerprint = fingerprint

browser, err := osciris.NewBrowser(ctx, options)
```

### Работа с удаленным браузером

**Важно:** Перед подключением к удаленному браузеру, Chrome должен быть запущен с флагом удаленной отладки:

```bash
# Windows
chrome.exe --remote-debugging-port=17986

# Linux/Mac
google-chrome --remote-debugging-port=17986
# или
chromium --remote-debugging-port=17986
```

```go
// Подключаемся к удаленному браузеру
remoteURL := "http://127.0.0.1:17986"

browser, err := osciris.NewRemoteBrowser(ctx, remoteURL, nil)
if err != nil {
    log.Fatal(err)
}
defer browser.Close()

// Получаем список всех вкладок
tabs, err := browser.ListTabs()
if err != nil {
    log.Fatal(err)
}

// Подключаемся к существующей вкладке (избегает создания новой)
if len(tabs) > 0 {
    tabBrowser, err := browser.ConnectToTab(tabs[0].ID)
    if err != nil {
        log.Fatal(err)
    }
    defer tabBrowser.Close()
    
    page := tabBrowser.NewPage()
    page.Navigate("https://example.com")
}

// Открываем новую вкладку
newTab, err := browser.OpenTab("https://example.com")
if err != nil {
    log.Fatal(err)
}
defer newTab.Close()

// Получаем ID текущей вкладки
tabID := newTab.GetTargetID()
log.Printf("Tab ID: %s", tabID)
```

## 📖 API Reference

### Browser

#### NewBrowser

Создает новый экземпляр браузера.

```go
browser, err := osciris.NewBrowser(ctx, options)
```

#### BrowserOptions

```go
type BrowserOptions struct {
    Headless     bool              // Headless режим
    UserDataDir  string            // Директория для данных браузера
    Fingerprint  *fp.Fingerprint   // Fingerprint для инжектирования
    Stealth      bool              // Stealth режим
    Timeout      time.Duration     // Timeout для операций
    Flags        []string          // Дополнительные флаги Chrome
    WindowWidth  int               // Ширина окна
    WindowHeight int               // Высота окна
    RemoteURL    string            // Адрес удаленного браузера (например, "http://127.0.0.1:17986")
    TargetID     target.ID         // ID существующей вкладки для подключения
}
```

#### Методы Browser

- `Context() context.Context` - Возвращает context браузера
- `Close() error` - Закрывает браузер
- `Run(...chromedp.Action) error` - Выполняет действия chromedp
- `NewPage() *Page` - Создает новую страницу
- `ListTabs() ([]Tab, error)` - Возвращает список всех вкладок (только для удаленного браузера)
- `OpenTab(url string) (*Browser, error)` - Открывает новую вкладку (только для удаленного браузера)
- `ConnectToTab(targetID target.ID) (*Browser, error)` - Подключается к существующей вкладке (только для удаленного браузера)
- `CloseTab() error` - Закрывает текущую вкладку (только для удаленного браузера)
- `GetTargetID() target.ID` - Возвращает ID текущей вкладки

#### Tab

Структура, представляющая вкладку браузера.

```go
type Tab struct {
    ID       target.ID  // Уникальный идентификатор вкладки
    Type     string     // Тип вкладки (обычно "page")
    Title    string     // Заголовок вкладки
    URL      string     // URL вкладки
    Attached bool       // Подключена ли вкладка
}
```

#### NewRemoteBrowser

Создает подключение к удаленному браузеру.

```go
browser, err := osciris.NewRemoteBrowser(ctx, "http://127.0.0.1:17986", options)
```

### Page

#### Методы Page

- `Navigate(url string) error` - Переходит по URL
- `NavigateAndWait(url, waitVisible string) error` - Переходит и ждет элемент
- `WaitVisible(selector string) error` - Ждет появления элемента
- `Click(selector string) error` - Кликает по элементу
- `SendKeys(selector, text string) error` - Отправляет текст
- `Value(selector string, result *string) error` - Получает значение
- `Text(selector string, result *string) error` - Получает текст
- `Screenshot(buf *[]byte) error` - Делает скриншот
- `Evaluate(expression string, result interface{}) error` - Выполняет JavaScript
- `Reload() error` - Перезагружает страницу
- `Back() error` - Возвращается назад
- `Forward() error` - Переходит вперед
- `Title(result *string) error` - Получает заголовок
- `URL(result *string) error` - Получает URL
- `RunActions(...chromedp.Action) error` - Выполняет произвольные действия

## 💡 Примеры

### Удаленный браузер и управление вкладками

```go
// Подключаемся к удаленному браузеру
browser, _ := osciris.NewRemoteBrowser(ctx, "http://127.0.0.1:17986", nil)

// Получаем список вкладок
tabs, _ := browser.ListTabs()
for _, tab := range tabs {
    log.Printf("Tab: %s - %s", tab.Title, tab.URL)
}

// Подключаемся к существующей вкладке
if len(tabs) > 0 {
    tabBrowser, _ := browser.ConnectToTab(tabs[0].ID)
    page := tabBrowser.NewPage()
    page.Navigate("https://example.com")
}

// Открываем новую вкладку
newTab, _ := browser.OpenTab("https://google.com")
defer newTab.Close()
```

### Работа с формами

```go
page := browser.NewPage()
page.Navigate("https://example.com/login")
page.SendKeys("#username", "user123")
page.SendKeys("#password", "pass123")
page.Click("#submit")
page.WaitVisible(".success-message")
```

### Скриншот

```go
var buf []byte
err := page.Screenshot(&buf)
if err != nil {
    log.Fatal(err)
}

err = os.WriteFile("screenshot.png", buf, 0644)
```

### JavaScript выполнение

```go
var result string
err := page.Evaluate(`document.querySelector("h1").textContent`, &result)
```

### Использование chromedp напрямую

```go
err := page.RunActions(
    chromedp.WaitVisible("#element"),
    chromedp.Click("#button"),
    chromedp.Sleep(2*time.Second),
)
```

## 🛡️ Stealth режим

По умолчанию включен stealth режим, который:
- Отключает признаки автоматизации
- Скрывает webdriver флаги
- Применяет fingerprint injection

Для отключения:

```go
options := osciris.DefaultBrowserOptions()
options.Stealth = false
```

## 📁 Примеры

В папке `examples/` вы найдете полные примеры использования:

- `examples/basic/` - Базовое использование с preset
- `examples/custom/` - Использование кастомного fingerprint
- `examples/stealth/` - Максимальная защита от детекции
- `examples/remote/` - Работа с удаленным браузером и управление вкладками

Запуск примеров:

```bash
cd examples/basic
go run main.go

# Для удаленного браузера сначала запустите Chrome:
# chrome --remote-debugging-port=17986
# Затем запустите пример:
cd examples/remote
go run main.go
```

## 🔗 Интеграция с fingerprint-injector-go

Osciris полностью интегрирован с [fingerprint-injector-go](https://github.com/vitaliitsarov/fingerprint-injector-go). Все пресеты и настройки fingerprint доступны:

```go
// Использование готовых пресетов
options.Fingerprint = fp.NewChrome119Windows11()
options.Fingerprint = fp.NewChrome119MacOS()
options.Fingerprint = fp.NewChrome119Linux()

// Кастомная настройка
fingerprint := fp.NewChrome119Windows11()
fingerprint.WebRTC.Disable = true
fingerprint.Canvas.Noise = 0.05
options.Fingerprint = fingerprint
```

## 📝 Лицензия

MIT

## 🤝 Вклад

Пул реквесты приветствуются!

## ⚠️ Дисклеймер

Этот инструмент предназначен только для легитимных целей:
- Тестирование защиты от ботов
- Автоматизация тестирования
- Исследование browser fingerprinting

Не используйте для обхода систем защиты или других незаконных действий.

