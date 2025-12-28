package osciris

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	cdruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	fp "github.com/vitaliitsarov/fingerprint-injector-go"
)

// Browser представляет браузер с поддержкой fingerprint injection
type Browser struct {
	ctx         context.Context
	cancel      context.CancelFunc
	injector    *fp.Injector
	options     *BrowserOptions
	allocCtx    context.Context
	allocCancel context.CancelFunc
	isRemote    bool
}

// Tab представляет вкладку браузера
type Tab struct {
	ID       target.ID `json:"id"`
	Type     string    `json:"type"`
	Title    string    `json:"title"`
	URL      string    `json:"url"`
	Attached bool      `json:"attached"`
}

// BrowserOptions содержит опции для создания браузера
type BrowserOptions struct {
	// Headless режим
	Headless bool

	// UserDataDir для сохранения данных браузера
	UserDataDir string

	// Fingerprint для инжектирования
	Fingerprint *fp.Fingerprint

	// Stealth режим (дополнительные флаги для скрытия автоматизации)
	Stealth bool

	// Timeout для операций
	Timeout time.Duration

	// Дополнительные флаги Chrome
	Flags []string

	// Window размеры
	WindowWidth  int
	WindowHeight int

	// RemoteURL адрес удаленного браузера (например, "http://127.0.0.1:17986")
	// Если указан, будет использован NewRemoteAllocator вместо NewExecAllocator
	RemoteURL string

	// TargetID ID существующей вкладки для подключения
	// Если указан, будет подключение к существующей вкладке вместо создания новой
	TargetID target.ID
}

// DefaultBrowserOptions возвращает опции по умолчанию
func DefaultBrowserOptions() *BrowserOptions {
	return &BrowserOptions{
		Headless:     false,
		Stealth:      true,
		Timeout:      60 * time.Second, // Увеличенный timeout для удаленного браузера
		WindowWidth:  1920,
		WindowHeight: 1080,
		Fingerprint:  fp.NewChrome119Windows11(),
	}
}

// NewBrowser создает новый экземпляр браузера
func NewBrowser(ctx context.Context, options *BrowserOptions) (*Browser, error) {
	if options == nil {
		options = DefaultBrowserOptions()
	}

	var allocCtx context.Context
	var allocCancel context.CancelFunc
	var isRemote bool

	// Проверяем, используется ли удаленный браузер
	if options.RemoteURL != "" {
		// Используем удаленный allocator
		allocCtx, allocCancel = chromedp.NewRemoteAllocator(ctx, options.RemoteURL)
		isRemote = true
	} else {
		// Настройка опций allocator для локального браузера
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", options.Headless),
			chromedp.Flag("window-size", fmt.Sprintf("%d,%d", options.WindowWidth, options.WindowHeight)),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("no-sandbox", true),
		)

		if options.Stealth {
			opts = append(opts,
				chromedp.Flag("disable-blink-features", "AutomationControlled"),
				chromedp.Flag("exclude-switches", "enable-automation"),
				chromedp.Flag("disable-extensions", false),
			)
		}

		if options.UserDataDir != "" {
			opts = append(opts, chromedp.UserDataDir(options.UserDataDir))
		}

		// Добавляем пользовательские флаги
		for _, flag := range options.Flags {
			opts = append(opts, chromedp.Flag(flag, ""))
		}

		// Создаем allocator context
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, opts...)
		isRemote = false
	}

	// Создаем browser context
	var browserCtx context.Context
	var browserCancel context.CancelFunc

	if options.TargetID != "" {
		// Подключаемся к существующей вкладке
		browserCtx, browserCancel = chromedp.NewContext(allocCtx, chromedp.WithTargetID(options.TargetID))
	} else {
		// Создаем новую вкладку
		browserCtx, browserCancel = chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, v ...interface{}) {
			// Можно добавить логирование
		}))
	}

	// Создаем инжектор fingerprint
	var injector *fp.Injector
	if options.Fingerprint != nil {
		injector = fp.NewInjector(options.Fingerprint)
	}

	browser := &Browser{
		ctx:         browserCtx,
		cancel:      func() { browserCancel() },
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		injector:    injector,
		options:     options,
		isRemote:    isRemote,
	}

	// Применяем fingerprint при создании
	if injector != nil {
		timeoutCtx, timeoutCancel := context.WithTimeout(browserCtx, options.Timeout)
		defer timeoutCancel()

		err := chromedp.Run(timeoutCtx, injector.ApplyAll(timeoutCtx))
		if err != nil {
			browser.Close()
			return nil, fmt.Errorf("failed to apply fingerprint: %w", err)
		}
	}

	return browser, nil
}

// NewRemoteBrowser создает подключение к удаленному браузеру
// Если не указан TargetID, создается новая вкладка
func NewRemoteBrowser(ctx context.Context, remoteURL string, options *BrowserOptions) (*Browser, error) {
	if options == nil {
		options = DefaultBrowserOptions()
	}
	options.RemoteURL = remoteURL
	return NewBrowser(ctx, options)
}

// NewRemoteBrowserManager создает подключение к удаленному браузеру БЕЗ создания новой вкладки
// Используется для управления браузером (получение списка вкладок, создание новых и т.д.)
func NewRemoteBrowserManager(ctx context.Context, remoteURL string, options *BrowserOptions) (*Browser, error) {
	if options == nil {
		options = DefaultBrowserOptions()
	}
	options.RemoteURL = remoteURL
	
	var allocCtx context.Context
	var allocCancel context.CancelFunc
	
	// Используем удаленный allocator
	allocCtx, allocCancel = chromedp.NewRemoteAllocator(ctx, remoteURL)
	
	// НЕ создаем chromedp контекст, чтобы не создавать новую вкладку
	// Вместо этого используем allocCtx напрямую для выполнения CDP команд
	// Создаем фиктивный контекст, который не будет использоваться для операций со страницей
	browserCtx, browserCancel := context.WithCancel(allocCtx)
	
	// Создаем инжектор fingerprint (но не применяем его, так как нет активной вкладки)
	var injector *fp.Injector
	if options.Fingerprint != nil {
		injector = fp.NewInjector(options.Fingerprint)
	}
	
	browser := &Browser{
		ctx:         browserCtx,
		cancel:      func() { browserCancel() },
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		injector:    injector,
		options:     options,
		isRemote:    true,
	}
	
	return browser, nil
}

// Context возвращает context браузера
func (b *Browser) Context() context.Context {
	return b.ctx
}

// Close закрывает браузер и освобождает ресурсы
func (b *Browser) Close() error {
	if b.cancel != nil {
		b.cancel()
	}
	// Для удаленного браузера не закрываем allocator, так как он может использоваться другими вкладками
	if !b.isRemote && b.allocCancel != nil {
		b.allocCancel()
	}
	return nil
}

// CloseTab закрывает текущую вкладку (только для удаленного браузера)
func (b *Browser) CloseTab() error {
	if !b.isRemote {
		return fmt.Errorf("CloseTab can only be used with remote browser")
	}

	// Для удаленного браузера создаем временный контекст из allocCtx
	tempCtx, tempCancel := chromedp.NewContext(b.allocCtx)
	defer tempCancel()

	timeoutCtx, cancel := context.WithTimeout(tempCtx, b.options.Timeout)
	defer cancel()

	// Получаем target ID текущей вкладки через список всех вкладок
	var targetID target.ID
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		targets, err := target.GetTargets().Do(ctx)
		if err != nil {
			return err
		}
		// Находим прикрепленную вкладку
		for _, t := range targets {
			if t.Attached && t.Type == "page" {
				targetID = t.TargetID
				return nil
			}
		}
		return fmt.Errorf("target ID not found")
	}))
	if err != nil {
		return fmt.Errorf("failed to get target ID: %w", err)
	}

	if targetID == "" {
		return fmt.Errorf("target ID not found")
	}

	// Закрываем вкладку через CDP
	err = chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return target.CloseTarget(targetID).Do(ctx)
	}))
	if err != nil {
		return fmt.Errorf("failed to close tab: %w", err)
	}

	// Закрываем context вкладки
	if b.cancel != nil {
		b.cancel()
	}

	return nil
}

// CloseTabByID закрывает вкладку по её ID
// Использует временный контекст для закрытия вкладки через CDP
func (b *Browser) CloseTabByID(targetID target.ID) error {
	if !b.isRemote {
		return fmt.Errorf("CloseTabByID can only be used with remote browser")
	}

	// Создаем временный контекст для выполнения команды закрытия
	// Этот контекст создаст новую временную вкладку, но мы её не используем
	tempCtx, tempCancel := chromedp.NewContext(b.allocCtx)
	defer tempCancel()

	timeoutCtx, cancel := context.WithTimeout(tempCtx, b.options.Timeout)
	defer cancel()

	// Закрываем вкладку через CDP команду CloseTarget
	// Используем временный контекст, который не привязан к закрываемой вкладке
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Проверяем, что вкладка существует
		targets, err := target.GetTargets().Do(ctx)
		if err != nil {
			return err
		}
		
		found := false
		for _, t := range targets {
			if t.TargetID == targetID {
				found = true
				break
			}
		}
		
		if !found {
			return fmt.Errorf("target %s not found", targetID)
		}
		
		// Закрываем вкладку через CloseTarget
		// Важно: это работает только если к вкладке не прикреплен активный контекст chromedp
		return target.CloseTarget(targetID).Do(ctx)
	}))
	
	if err != nil {
		return fmt.Errorf("failed to close tab: %w", err)
	}

	return nil
}

// Run выполняет действия в браузере
func (b *Browser) Run(actions ...chromedp.Action) error {
	// Используем timeout из опций (без умножения для удаленного браузера)
	timeout := b.options.Timeout
	timeoutCtx, cancel := context.WithTimeout(b.ctx, timeout)
	defer cancel()
	return chromedp.Run(timeoutCtx, chromedp.Tasks(actions))
}

// Page представляет страницу браузера
type Page struct {
	browser *Browser
	ctx     context.Context
}

// NewPage создает новую страницу
func (b *Browser) NewPage() *Page {
	return &Page{
		browser: b,
		ctx:     b.ctx,
	}
}

// Navigate переходит по URL
func (p *Page) Navigate(url string) error {
	return p.browser.Run(chromedp.Navigate(url))
}

// NavigateAndWait переходит по URL и ждет загрузки
func (p *Page) NavigateAndWait(url string, waitVisible string) error {
	return p.browser.Run(
		chromedp.Navigate(url),
		chromedp.WaitVisible(waitVisible),
	)
}

// WaitVisible ждет появления элемента
func (p *Page) WaitVisible(selector string) error {
	return p.browser.Run(chromedp.WaitVisible(selector))
}

// Click кликает по элементу
func (p *Page) Click(selector string) error {
	return p.browser.Run(chromedp.Click(selector))
}

// SendKeys отправляет текст в элемент
func (p *Page) SendKeys(selector, text string) error {
	return p.browser.Run(chromedp.SendKeys(selector, text))
}

// Value получает значение элемента
func (p *Page) Value(selector string, result *string) error {
	return p.browser.Run(chromedp.Value(selector, result))
}

// Text получает текст элемента
func (p *Page) Text(selector string, result *string) error {
	return p.browser.Run(chromedp.Text(selector, result))
}

// Screenshot делает скриншот страницы
func (p *Page) Screenshot(buf *[]byte) error {
	return p.browser.Run(chromedp.CaptureScreenshot(buf))
}

// Evaluate выполняет JavaScript и возвращает результат
func (p *Page) Evaluate(expression string, result interface{}) error {
	return p.browser.Run(chromedp.Evaluate(expression, result))
}

// Reload перезагружает страницу
func (p *Page) Reload() error {
	return p.browser.Run(chromedp.Reload())
}

// Back возвращается назад
func (p *Page) Back() error {
	return p.browser.Run(chromedp.NavigateBack())
}

// Forward переходит вперед
func (p *Page) Forward() error {
	return p.browser.Run(chromedp.NavigateForward())
}

// Title получает заголовок страницы
func (p *Page) Title(result *string) error {
	return p.browser.Run(chromedp.Title(result))
}

// URL получает текущий URL
func (p *Page) URL(result *string) error {
	return p.browser.Run(chromedp.Location(result))
}

// RunActions выполняет произвольные действия chromedp
func (p *Page) RunActions(actions ...chromedp.Action) error {
	return p.browser.Run(actions...)
}

// WaitReady ждет готовности элемента
func (p *Page) WaitReady(selector string) error {
	return p.browser.Run(chromedp.WaitReady(selector))
}

// Focus устанавливает фокус на элемент
func (p *Page) Focus(selector string) error {
	return p.browser.Run(chromedp.Focus(selector))
}

// ScrollIntoView прокручивает страницу к элементу
func (p *Page) ScrollIntoView(selector string) error {
	return p.browser.Run(chromedp.ScrollIntoView(selector))
}

// ClickWithScroll прокручивает к элементу и кликает по нему
func (p *Page) ClickWithScroll(selector string) error {
	return p.browser.Run(
		chromedp.ScrollIntoView(selector),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Click(selector),
	)
}

// ClickXY кликает по координатам
func (p *Page) ClickXY(x, y float64) error {
	return p.browser.Run(chromedp.MouseClickXY(x, y))
}

// KeyEvent отправляет событие нажатия клавиши
func (p *Page) KeyEvent(key string) error {
	return p.browser.Run(chromedp.KeyEvent(key))
}

// SendKeysChar отправляет текст посимвольно (имитация человеческого ввода)
func (p *Page) SendKeysChar(selector, text string) error {
	for _, char := range text {
		err := p.browser.Run(chromedp.SendKeys(selector, string(char)))
		if err != nil {
			return err
		}
		// Случайная задержка между символами (50-150ms)
		delay := time.Duration(50+rand.Intn(100)) * time.Millisecond
		time.Sleep(delay)
	}
	return nil
}

// SendKeysEnter отправляет Enter в элемент
func (p *Page) SendKeysEnter(selector string) error {
	return p.browser.Run(chromedp.SendKeys(selector, kb.Enter))
}

// Nodes получает список узлов DOM по селектору
func (p *Page) Nodes(selector string) ([]*cdp.Node, error) {
	var nodes []*cdp.Node
	err := p.browser.Run(chromedp.Nodes(selector, &nodes))
	return nodes, err
}

// NodesAll получает все узлы DOM по селектору
func (p *Page) NodesAll(selector string) ([]*cdp.Node, error) {
	var nodes []*cdp.Node
	err := p.browser.Run(chromedp.Nodes(selector, &nodes, chromedp.ByQueryAll))
	return nodes, err
}

// ClearInput очищает поле ввода
func (p *Page) ClearInput(selector string) error {
	return p.browser.Run(
		chromedp.Focus(selector),
		chromedp.Evaluate(fmt.Sprintf(`document.querySelector('%s').value = '';`, selector), nil),
	)
}

// MouseMove перемещает мышь к координатам
func (p *Page) MouseMove(x, y float64) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		return input.DispatchMouseEvent(input.MouseMoved, x, y).Do(ctx)
	}))
}

// MouseClick выполняет клик мыши по координатам
func (p *Page) MouseClick(x, y float64, button input.MouseButton) error {
	return p.browser.Run(
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Нажатие
			if err := input.DispatchMouseEvent(input.MousePressed, x, y).
				WithButton(button).
				WithClickCount(1).
				Do(ctx); err != nil {
				return err
			}
			time.Sleep(50 * time.Millisecond)
			// Отпускание
			return input.DispatchMouseEvent(input.MouseReleased, x, y).
				WithButton(button).
				WithClickCount(1).
				Do(ctx)
		}),
	)
}

// MouseClickCtrl выполняет Ctrl+Click по координатам (открытие в новой вкладке)
func (p *Page) MouseClickCtrl(x, y float64) error {
	return p.browser.Run(
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Сначала перемещаем мышь к элементу
			if err := input.DispatchMouseEvent(input.MouseMoved, x, y).Do(ctx); err != nil {
				return err
			}
			time.Sleep(50 * time.Millisecond)
			
			// Нажатие с Ctrl
			if err := input.DispatchMouseEvent(input.MousePressed, x, y).
				WithButton(input.Left).
				WithModifiers(input.ModifierCtrl).
				WithClickCount(1).
				Do(ctx); err != nil {
				return err
			}
			time.Sleep(100 * time.Millisecond) // Увеличенная задержка для надежности
			
			// Отпускание с Ctrl
			if err := input.DispatchMouseEvent(input.MouseReleased, x, y).
				WithButton(input.Left).
				WithModifiers(input.ModifierCtrl).
				WithClickCount(1).
				Do(ctx); err != nil {
				return err
			}
			
			// Дополнительная задержка для открытия новой вкладки
			time.Sleep(300 * time.Millisecond)
			return nil
		}),
	)
}

// MouseWheel прокручивает колесом мыши
func (p *Page) MouseWheel(x, y, deltaY float64) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		return input.DispatchMouseEvent(input.MouseWheel, x, y).
			WithDeltaY(deltaY).
			Do(ctx)
	}))
}

// GetElementBox получает координаты элемента
func (p *Page) GetElementBox(selector string) (*dom.BoxModel, error) {
	var nodes []*cdp.Node
	err := p.browser.Run(chromedp.Nodes(selector, &nodes))
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	var box *dom.BoxModel
	err = p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		box, err = dom.GetBoxModel().WithNodeID(nodes[0].NodeID).Do(ctx)
		return err
	}))
	return box, err
}

// ClickElementWithCtrl выполняет Ctrl+Click по элементу (открытие в новой вкладке)
func (p *Page) ClickElementWithCtrl(selector string) error {
	box, err := p.GetElementBox(selector)
	if err != nil {
		return err
	}

	// Вычисляем центр элемента
	x := (box.Content[0] + box.Content[2]) / 2
	y := (box.Content[1] + box.Content[5]) / 2

	return p.MouseClickCtrl(x, y)
}

// ClickOnNewTab выполняет Ctrl+Click по элементу для открытия в новой вкладке
// Прокручивает к элементу и кликает с модификатором Ctrl
func (p *Page) ClickOnNewTab(selector string) error {
	// Прокручиваем к элементу для гарантии его видимости
	err := p.ScrollIntoView(selector)
	if err != nil {
		// Продолжаем даже если прокрутка не удалась
	}
	
	// Небольшая задержка для прокрутки
	time.Sleep(300 * time.Millisecond)
	
	// Дополнительно прокручиваем так, чтобы элемент был в центре видимой области
	err = p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		// Получаем box model для более точной прокрутки
		var nodes []*cdp.Node
		if err := chromedp.Nodes(selector, &nodes).Do(ctx); err != nil || len(nodes) == 0 {
			return nil // Продолжаем даже если не удалось получить узлы
		}
		
		box, err := dom.GetBoxModel().WithNodeID(nodes[0].NodeID).Do(ctx)
		if err != nil {
			return nil // Продолжаем даже если не удалось получить box model
		}
		
		// Вычисляем центр элемента
		centerY := (box.Content[1] + box.Content[5]) / 2
		
		// Прокручиваем так, чтобы элемент был в центре экрана (примерно на 40% от верха)
		scrollY := centerY - 400 // 400px от верха экрана для лучшей видимости
		if scrollY > 0 {
			if err := input.DispatchMouseEvent(input.MouseWheel, 0, 0).
				WithDeltaY(float64(scrollY)).
				Do(ctx); err != nil {
				return nil // Продолжаем даже если прокрутка не удалась
			}
			time.Sleep(200 * time.Millisecond)
		}
		return nil
	}))
	
	// Получаем координаты элемента
	box, err := p.GetElementBox(selector)
	if err != nil {
		return fmt.Errorf("failed to get element box: %w", err)
	}

	// Вычисляем центр элемента
	x := (box.Content[0] + box.Content[2]) / 2
	y := (box.Content[1] + box.Content[5]) / 2

	// Выполняем Ctrl+Click
	return p.MouseClickCtrl(x, y)
}

// HumanMouseMove имитирует человеческое движение мыши
func (p *Page) HumanMouseMove(x, y float64) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		// Случайное движение с небольшими отклонениями
		offsetX := float64(rand.Intn(10) - 5)
		offsetY := float64(rand.Intn(10) - 5)
		
		if err := input.DispatchMouseEvent(input.MouseMoved, x+offsetX, y+offsetY).Do(ctx); err != nil {
			return err
		}
		
		time.Sleep(time.Duration(30+rand.Intn(120)) * time.Millisecond)
		
		return input.DispatchMouseEvent(input.MouseMoved, x, y).Do(ctx)
	}))
}

// HumanScroll имитирует человеческую прокрутку
func (p *Page) HumanScroll(x, y float64) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		deltaY := float64(50 - rand.Intn(100))
		return input.DispatchMouseEvent(input.MouseWheel, x, y).
			WithDeltaY(deltaY).
			Do(ctx)
	}))
}

// SetUserAgent устанавливает User-Agent и платформу
func (p *Page) SetUserAgent(userAgent, platform string) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		return emulation.SetUserAgentOverride(userAgent).
			WithPlatform(platform).
			Do(ctx)
	}))
}

// SetViewport устанавливает размеры окна просмотра
func (p *Page) SetViewport(width, height int64, mobile bool) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		return emulation.SetDeviceMetricsOverride(width, height, 1.0, mobile).Do(ctx)
	}))
}

// SetGeolocation устанавливает геолокацию
func (p *Page) SetGeolocation(lat, lng float64) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		return emulation.SetGeolocationOverride().
			WithLatitude(lat).
			WithLongitude(lng).
			Do(ctx)
	}))
}

// AddScriptToEvaluateOnNewDocument добавляет скрипт, который выполняется на каждой новой странице
func (p *Page) AddScriptToEvaluateOnNewDocument(jsCode string) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(jsCode).Do(ctx)
		if err != nil {
			return err
		}
		// Также выполняем скрипт на текущей странице
		_, _, err = cdruntime.Evaluate(jsCode).Do(ctx)
		return err
	}))
}

// ReadyState получает состояние готовности страницы
func (p *Page) ReadyState() (string, error) {
	var readyState string
	err := p.browser.Run(chromedp.Evaluate(`document.readyState`, &readyState))
	return readyState, err
}

// WaitForReadyState ждет определенного состояния готовности страницы
func (p *Page) WaitForReadyState(state string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for readyState %s", state)
		default:
			currentState, err := p.ReadyState()
			if err != nil {
				return err
			}
			if currentState == state {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// FastWaitForElement быстро ждет появления элемента с коротким таймаутом
func (p *Page) FastWaitForElement(selector string, maxWaitMs int) error {
	timeout := time.Duration(maxWaitMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	// Пробуем найти элемент несколько раз с короткими интервалами
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			nodes, err := p.Nodes(selector)
			if err == nil && len(nodes) > 0 {
				return nil
			}
			time.Sleep(timeout / 10)
		}
	}
	return fmt.Errorf("element not found: %s", selector)
}

// FastCheckElement быстро проверяет наличие элемента без ожидания (с коротким таймаутом)
func (p *Page) FastCheckElement(selector string) bool {
	// Создаем контекст с коротким таймаутом для быстрой проверки
	ctx, cancel := context.WithTimeout(p.ctx, 1*time.Second)
	defer cancel()
	
	// Используем канал для получения результата с таймаутом
	resultChan := make(chan bool, 1)
	go func() {
		var nodes []*cdp.Node
		err := chromedp.Run(ctx, chromedp.Nodes(selector, &nodes))
		resultChan <- (err == nil && len(nodes) > 0)
	}()
	
	select {
	case result := <-resultChan:
		return result
	case <-time.After(500 * time.Millisecond):
		// Таймаут - считаем что элемента нет
		return false
	}
}

// ScrollPage прокручивает страницу для загрузки всех элементов
func (p *Page) ScrollPage(scrollDownTimes, scrollUpTimes int) error {
	return p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		// Прокручиваем страницу вниз несколько раз
		for i := 0; i < scrollDownTimes; i++ {
			if err := input.DispatchMouseEvent(input.MouseWheel, 0, 0).
				WithDeltaY(500).
				Do(ctx); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)
		}
		
		// Прокручиваем страницу вверх обратно
		for i := 0; i < scrollUpTimes; i++ {
			if err := input.DispatchMouseEvent(input.MouseWheel, 0, 0).
				WithDeltaY(-500).
				Do(ctx); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)
		}
		return nil
	}))
}

// GetElementAttributes получает атрибуты элемента по NodeID
func (p *Page) GetElementAttributes(nodeID cdp.NodeID) (map[string]string, error) {
	attrsMap := make(map[string]string)
	err := p.browser.Run(chromedp.ActionFunc(func(ctx context.Context) error {
		attrs, err := dom.GetAttributes(nodeID).Do(ctx)
		if err == nil && len(attrs) > 0 {
			// Атрибуты возвращаются как массив [name1, value1, name2, value2, ...]
			for j := 0; j < len(attrs)-1; j += 2 {
				attrName := attrs[j]
				attrValue := attrs[j+1]
				attrsMap[attrName] = attrValue
			}
		}
		return err
	}))
	return attrsMap, err
}

// ListTabs возвращает список всех вкладок браузера
func (b *Browser) ListTabs() ([]Tab, error) {
	if !b.isRemote {
		return nil, fmt.Errorf("ListTabs can only be used with remote browser")
	}

	// Для удаленного браузера создаем временный контекст из allocCtx для выполнения CDP команд
	// Этот контекст создаст временную вкладку, но мы её не используем
	tempCtx, tempCancel := chromedp.NewContext(b.allocCtx)
	// НЕ используем defer tempCancel() здесь, чтобы контекст не отменялся преждевременно
	// Контекст будет отменен после выполнения команды

	timeoutCtx, cancel := context.WithTimeout(tempCtx, b.options.Timeout*2)
	
	// Выполняем команду через chromedp.Run для установки соединения
	var targets []*target.Info
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		targets, err = target.GetTargets().Do(ctx)
		return err
	}))
	
	// Отменяем контексты после выполнения команды
	cancel()
	tempCancel()
	
	if err != nil {
		return nil, fmt.Errorf("failed to get targets: %w", err)
	}

	tabs := make([]Tab, 0, len(targets))
	for _, t := range targets {
		if t.Type == "page" {
			tabs = append(tabs, Tab{
				ID:       t.TargetID,
				Type:     t.Type,
				Title:    t.Title,
				URL:      t.URL,
				Attached: t.Attached,
			})
		}
	}

	return tabs, nil
}

// OpenTab открывает новую вкладку в браузере
func (b *Browser) OpenTab(url string) (*Browser, error) {
	if !b.isRemote {
		return nil, fmt.Errorf("OpenTab can only be used with remote browser")
	}

	// Для удаленного браузера создаем временный контекст из allocCtx для выполнения CDP команд
	// Этот контекст создаст временную вкладку, но мы её не используем
	tempCtx, tempCancel := chromedp.NewContext(b.allocCtx)
	// НЕ используем defer tempCancel() здесь, чтобы контекст не отменялся преждевременно

	// Используем очень большой timeout для создания вкладки
	timeoutCtx, cancel := context.WithTimeout(tempCtx, b.options.Timeout*5)

	createURL := "about:blank"
	if url != "" {
		createURL = url
	}

	// Создаем новую вкладку через CDP
	var targetID target.ID
	err := chromedp.Run(timeoutCtx, 
		chromedp.Sleep(1*time.Second), // Увеличенная задержка для установки соединения
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			targetID, err = target.CreateTarget(createURL).Do(ctx)
			return err
		}))
	
	// Отменяем временные контексты после успешного создания вкладки
	// Но делаем это с небольшой задержкой, чтобы операции успели завершиться
	if err != nil {
		cancel()
		tempCancel()
		return nil, fmt.Errorf("failed to create new tab: %w", err)
	}
	
	// Даем время для завершения операций перед отменой временных контекстов
	time.Sleep(200 * time.Millisecond)
	cancel()
	tempCancel()

	// Подключаемся к новой вкладке
	tabCtx, tabCancel := chromedp.NewContext(b.allocCtx, chromedp.WithTargetID(targetID))

	// Создаем новый Browser для вкладки
	newBrowser := &Browser{
		ctx:         tabCtx,
		cancel:      tabCancel,
		allocCtx:    b.allocCtx,
		allocCancel: nil, // Не закрываем allocator, он общий
		injector:    b.injector,
		options:     b.options,
		isRemote:    true,
	}

	// Применяем fingerprint
	if newBrowser.injector != nil {
		fpCtx, fpCancel := context.WithTimeout(tabCtx, b.options.Timeout)
		defer fpCancel()

		err := chromedp.Run(fpCtx, newBrowser.injector.ApplyAll(fpCtx))
		if err != nil {
			newBrowser.Close()
			return nil, fmt.Errorf("failed to apply fingerprint: %w", err)
		}
	}

	// Если URL был указан, переходим на него
	if url != "" {
		err = chromedp.Run(tabCtx, chromedp.Navigate(url))
		if err != nil {
			newBrowser.Close()
			return nil, fmt.Errorf("failed to navigate: %w", err)
		}
	}

	return newBrowser, nil
}

// ConnectToTab подключается к существующей вкладке по ID
func (b *Browser) ConnectToTab(targetID target.ID) (*Browser, error) {
	if !b.isRemote {
		return nil, fmt.Errorf("ConnectToTab can only be used with remote browser")
	}

	// Подключаемся к существующей вкладке
	tabCtx, tabCancel := chromedp.NewContext(b.allocCtx, chromedp.WithTargetID(targetID))

	// Создаем новый Browser для вкладки
	newBrowser := &Browser{
		ctx:         tabCtx,
		cancel:      tabCancel,
		allocCtx:    b.allocCtx,
		allocCancel: nil, // Не закрываем allocator, он общий
		injector:    b.injector,
		options:     b.options,
		isRemote:    true,
	}

	// Устанавливаем соединение с вкладкой
	// Используем tabCtx напрямую без дополнительного timeout
	// Выполняем простую команду для установки соединения
	// Ждем достаточно времени, чтобы соединение установилось
	err := chromedp.Run(tabCtx, 
		chromedp.Sleep(2*time.Second), // Увеличенная задержка для установки соединения
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Пытаемся получить текущий URL для проверки соединения
			var url string
			err := chromedp.Location(&url).Do(ctx)
			if err != nil {
				// Если не удалось получить URL, это нормально для новой вкладки
				// Просто проверяем, что соединение установлено
				return nil
			}
			return nil
		}))
	
	if err != nil {
		// Если ошибка при установке соединения, все равно продолжаем
		// Соединение может установиться позже при использовании
		// return nil, fmt.Errorf("failed to connect to tab: %w", err)
	}

	// Применяем fingerprint
	if newBrowser.injector != nil {
		fpCtx, fpCancel := context.WithTimeout(tabCtx, b.options.Timeout)
		defer fpCancel()

		err := chromedp.Run(fpCtx, newBrowser.injector.ApplyAll(fpCtx))
		if err != nil {
			// Не закрываем браузер при ошибке fingerprint, просто логируем
			// return nil, fmt.Errorf("failed to apply fingerprint: %w", err)
		}
	}

	return newBrowser, nil
}

// GetTargetID возвращает ID текущей вкладки
func (b *Browser) GetTargetID() target.ID {
	if !b.isRemote {
		return ""
	}

	// Для удаленного браузера создаем временный контекст из allocCtx
	tempCtx, tempCancel := chromedp.NewContext(b.allocCtx)
	defer tempCancel()

	timeoutCtx, cancel := context.WithTimeout(tempCtx, b.options.Timeout)
	defer cancel()

	var targetID target.ID
	chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		targets, err := target.GetTargets().Do(ctx)
		if err != nil {
			return err
		}
		// Находим текущую вкладку (прикрепленную)
		for _, t := range targets {
			if t.Attached && t.Type == "page" {
				targetID = t.TargetID
				return nil
			}
		}
		return nil
	}))

	return targetID
}

