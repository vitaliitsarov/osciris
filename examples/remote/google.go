package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/playy/osciris"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	// fp "github.com/vitaliitsarov/fingerprint-injector-go"
)

// AdInfo содержит информацию о рекламном объявлении
type AdInfo struct {
	Link   string
	Domain string
}

func main() {
	ctx := context.Background()

	// Подключаемся к удаленному браузеру на порту 17986
	// Убедитесь, что Chrome запущен с флагом: chrome --remote-debugging-port=17986
	remoteURL := "http://127.0.0.1:53817"

	options := &osciris.BrowserOptions{
		Timeout:     30 * time.Second,
		// Fingerprint: fp.NewChrome119Windows11(),
	}

	// Подключаемся к удаленному браузеру БЕЗ создания новой вкладки
	// Используем NewRemoteBrowserManager для управления браузером
	browser, err := osciris.NewRemoteBrowserManager(ctx, remoteURL, options)
	if err != nil {
		log.Fatal(err)
	}
	// НЕ закрываем browser.Close() - оставляем браузер открытым
	// defer browser.Close()

	log.Println("Connected to remote browser")

	// Получаем список всех вкладок
	tabs, err := browser.ListTabs()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Found %d tabs:", len(tabs))
	for i, tab := range tabs {
		log.Printf("  Tab %d: ID=%s, Title=%s, URL=%s, Attached=%v",
			i+1, tab.ID, tab.Title, tab.URL, tab.Attached)
	}

	// Если есть существующие вкладки, подключаемся к первой неприкрепленной
	// (чтобы избежать подключения к вкладке, которая уже используется)
	var tabToConnect *osciris.Tab
	for i := range tabs {
		if !tabs[i].Attached {
			tabToConnect = &tabs[i]
			break
		}
	}
	
	if tabToConnect != nil {
		log.Printf("\nConnecting to existing tab: %s", tabToConnect.ID)
		tabBrowser, err := browser.ConnectToTab(tabToConnect.ID)
		if err != nil {
			log.Printf("Error connecting to tab: %v", err)
			log.Println("Continuing anyway...")
		} else {
			// НЕ закрываем tabBrowser.Close() - оставляем вкладку открытой
			// defer tabBrowser.Close()

			page := tabBrowser.NewPage()

			// Открываем Google
			startTime := time.Now()
			log.Println("Navigating to Google...")
			err = page.Navigate("https://www.google.com")
			if err != nil {
				log.Printf("Error navigating to Google: %v", err)
				// Не завершаем программу, продолжаем работу
			} else {
				navTime := time.Since(startTime)
				log.Printf("Navigation completed in %v", navTime)
				
				// Небольшая задержка для загрузки страницы
				log.Println("Waiting for page to load...")
				time.Sleep(2 * time.Second)
				
				
				// Проверяем наличие кнопки "Принять все" (Accept All) в DOM
				// Используем только методы chromedp без JavaScript
				log.Println("Checking for Accept All button in DOM...")
				
				var buttonFound bool
				for i := 0; i < 2; i++ {
					// Ищем элемент в DOM через chromedp.Nodes (работает даже если элемент не виден)
					nodes, err := page.Nodes("#L2AGLb")
					if err == nil && len(nodes) > 0 {
						log.Printf("Found button in DOM (attempt %d), trying to click...", i+1)
						
						// Метод 1: Прокручиваем к элементу и кликаем
						err = page.ScrollIntoView("#L2AGLb")
						if err == nil {
							time.Sleep(300 * time.Millisecond) // Даем время на прокрутку
							err = page.Click("#L2AGLb")
							if err == nil {
								log.Println("Accept All button clicked successfully (method: ScrollIntoView + Click)")
								buttonFound = true
								break
							}
						}
						
						// Метод 2: Клик с прокруткой через ClickWithScroll
						err = page.ClickWithScroll("#L2AGLb")
						if err == nil {
							log.Println("Accept All button clicked successfully (method: ClickWithScroll)")
							buttonFound = true
							break
						}
						
						// Метод 3: Получаем координаты элемента и кликаем по ним
						box, err := page.GetElementBox("#L2AGLb")
						if err == nil && box != nil {
							// Вычисляем центр элемента
							x := (box.Content[0] + box.Content[2]) / 2
							y := (box.Content[1] + box.Content[5]) / 2
							log.Printf("Clicking at coordinates: x=%.2f, y=%.2f", x, y)
							
							// Прокручиваем к элементу перед кликом
							err = page.ScrollIntoView("#L2AGLb")
							if err == nil {
								time.Sleep(300 * time.Millisecond)
							}
							
							// Кликаем по координатам
							err = page.ClickXY(x, y)
							if err == nil {
								log.Println("Accept All button clicked successfully (method: ClickXY)")
								buttonFound = true
								break
							}
							
							// Альтернатива: MouseClick по координатам (используем 0 для левой кнопки)
							// Примечание: MouseClick требует input.MouseButton, но мы можем использовать ClickXY
							// err = page.MouseClick(x, y, input.Left) // Требует импорт input
							if err == nil {
								log.Println("Accept All button clicked successfully (method: MouseClick)")
								buttonFound = true
								break
							}
						}
						
						// Метод 4: Просто клик (может работать если элемент уже в видимой области)
						err = page.Click("#L2AGLb")
						if err == nil {
							log.Println("Accept All button clicked successfully (method: Click)")
							buttonFound = true
							break
						}
						
						log.Printf("All click methods failed for attempt %d, retrying...", i+1)
					} else {
						log.Printf("Button not found in DOM (attempt %d), retrying...", i+1)
					}
					
					// Ждем перед следующей попыткой
					time.Sleep(500 * time.Millisecond)
				}
				
				if !buttonFound {
					log.Println("Accept All button not found or could not be clicked after multiple attempts")
				} else {
					// Небольшая задержка после клика
					time.Sleep(1 * time.Second)
				}


				// Пытаемся найти поле поиска и ввести текст
				log.Println("Checking for search input...")
				waitStart := time.Now()
				nodes, err := page.Nodes("[name='q']")
				if err == nil && len(nodes) > 0 {
					log.Printf("Search input found in %v", time.Since(waitStart))
					
					// Вводим текст "lego" в поле поиска посимвольно (имитация человеческого ввода)
					log.Println("Typing 'lego' in search field (character by character)...")
					err = page.SendKeysChar("[name='q']", "lego")
					if err != nil {
						log.Printf("Error typing text: %v", err)
					} else {
						log.Println("Text 'lego' entered successfully")
						// Небольшая задержка после ввода
						time.Sleep(500 * time.Millisecond)
						
						// Нажимаем Enter для поиска
						log.Println("Pressing Enter...")
						err = page.SendKeysEnter("[name='q']")
						if err != nil {
							log.Printf("Error pressing Enter: %v", err)
							// Пробуем альтернативный способ
							err = page.KeyEvent("Enter")
							if err != nil {
								log.Printf("Error with KeyEvent Enter: %v", err)
							} else {
								log.Println("Enter pressed successfully")
							}
						} else {
							log.Println("Enter pressed successfully")
						}
					}
				} else {
					log.Printf("Search input not found yet (took %v), continuing anyway...", time.Since(waitStart))
				}


				// Получаем текущий URL
				var url string
				err = page.URL(&url)
				if err != nil {
					log.Printf("Warning: failed to get URL: %v", err)
				} else {
					log.Printf("Current URL: %s", url)
				}

				// Получаем заголовок
				var title string
				err = page.Title(&title)
				if err != nil {
					log.Printf("Warning: failed to get title: %v", err)
				} else {
					log.Printf("Page title: %s", title)
				}

				// Ждем загрузки результатов поиска
				log.Println("\nWaiting for search results to load...")
				time.Sleep(3 * time.Second)

				// Ищем рекламу на странице
				log.Println("Searching for ads...")
				ads, err := searchAdsNative(page, 5) // Максимум 5 рекламных объявлений
				if err != nil {
					log.Printf("Error searching ads: %v", err)
				} else {
					log.Printf("Found %d ads", len(ads))
					
					// Кликаем по первой найденной рекламе
					if len(ads) > 0 {
						log.Printf("Clicking on ad: %s (domain: %s)", ads[0].Link, ads[0].Domain)
						err = clickAdByLink(page, ads[0].Link)
						if err != nil {
							log.Printf("Error clicking ad: %v", err)
						} else {
							log.Println("Ad clicked successfully!")
							time.Sleep(2 * time.Second)
						}
					}
				}
			}
		}
	}

	// Создаем новую вкладку
	// log.Println("\nOpening new tab...")
	// newTab, err := browser.OpenTab("https://example.com")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// НЕ закрываем newTab.Close() - оставляем вкладку открытой
	// defer newTab.Close()

	// newPage := newTab.NewPage()

	// // Получаем URL новой вкладки
	// var newURL string
	// err = newPage.URL(&newURL)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("New tab URL: %s", newURL)

	// Получаем список вкладок снова
	tabs, err = browser.ListTabs()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("\nNow we have %d tabs", len(tabs))

	// Получаем ID текущей вкладки
	// currentTabID := newTab.GetTargetID()
	// log.Printf("Current tab ID: %s", currentTabID)

	log.Println("\nBrowser will stay open. Press Ctrl+C to exit.")
	// Ждем, чтобы браузер оставался открытым
	// Программа будет работать до тех пор, пока вы не нажмете Ctrl+C
	select {}
}

// searchAdsNative ищет рекламные объявления на странице Google БЕЗ использования Evaluate
// Использует скролл по странице для обнаружения всех элементов
func searchAdsNative(page *osciris.Page, maxAds int) ([]AdInfo, error) {
	var results []AdInfo
	uniqueDomains := make(map[string]bool)

	// Делаем скролл по странице для загрузки всех элементов
	log.Println("Scrolling page to load all elements...")
	err := page.ScrollPage(3, 3)
	if err != nil {
		return results, fmt.Errorf("failed to scroll page: %w", err)
	}

	// Ждем появления контейнера рекламы
	log.Println("Waiting for ad container...")
	_ = page.FastWaitForElement(".uEierd", 2000)

	// Получаем все ссылки рекламы через NodesAll
	log.Println("Searching for ad elements...")
	adNodes, err := page.NodesAll(".uEierd a")
	if err != nil {
		return results, fmt.Errorf("failed to get ad nodes: %w", err)
	}

	log.Printf("Found %d ad link elements", len(adNodes))

	// Обрабатываем каждый найденный элемент
	for i, node := range adNodes {
		if len(results) >= maxAds {
			break
		}

		// Прокручиваем к элементу для гарантии его видимости
		err = page.RunActions(chromedp.ActionFunc(func(ctx context.Context) error {
			// Получаем box model для элемента
			box, err := dom.GetBoxModel().WithNodeID(node.NodeID).Do(ctx)
			if err != nil {
				return err
			}

			// Прокручиваем к элементу
			y := box.Content[1]
			if err := input.DispatchMouseEvent(input.MouseWheel, 0, 0).
				WithDeltaY(float64(y - 400)). // Прокручиваем так, чтобы элемент был виден
				Do(ctx); err != nil {
				return err
			}
			time.Sleep(100 * time.Millisecond)
			return nil
		}))

		// Получаем атрибуты элемента через DOM API
		var link, domain string
		attrs, err := page.GetElementAttributes(node.NodeID)
		if err == nil {
			// Ищем href, data-rw, data-pcu в атрибутах
			if href, ok := attrs["href"]; ok && href != "" {
				link = href
			}
			if dataRw, ok := attrs["data-rw"]; ok && dataRw != "" && link == "" {
				link = dataRw
			}
			if dataPcu, ok := attrs["data-pcu"]; ok && dataPcu != "" {
				domain = dataPcu
			}
			if dataDtld, ok := attrs["data-dtld"]; ok && dataDtld != "" && domain == "" {
				domain = dataDtld
			}
		}

		// Фильтруем и добавляем рекламу
		if link != "" && strings.Contains(link, "/aclk") {
			// Фильтруем нерелевантные ссылки
			if strings.Contains(link, "google.com/search") ||
				strings.Contains(link, "google.com/url") ||
				strings.Contains(link, "google.com/webhp") {
				continue
			}

			// Нормализуем домен
			if domain == "" {
				domain = extractDomainFromLink(link)
			} else {
				domain = strings.TrimPrefix(domain, "https://")
				domain = strings.TrimPrefix(domain, "http://")
				domain = strings.TrimPrefix(domain, "www.")
				parts := strings.Split(domain, ",")
				if len(parts) > 0 {
					domain = strings.TrimSpace(parts[0])
				}
				domain = strings.TrimSuffix(domain, "/")
			}

			// Проверяем уникальность домена
			if domain != "" && !uniqueDomains[domain] {
				uniqueDomains[domain] = true
				results = append(results, AdInfo{
					Link:   link,
					Domain: domain,
				})
				log.Printf("Found ad %d: Domain=%s, Link=%s", len(results), domain, link)
			}
		}
		_ = i
	}

	log.Printf("Total ads found: %d", len(results))
	return results, nil
}

// extractDomainFromLink извлекает домен из ссылки
func extractDomainFromLink(link string) string {
	// Упрощенное извлечение домена
	if strings.Contains(link, "googleadservices.com") {
		// Парсим параметр url из ссылки
		if idx := strings.Index(link, "url="); idx != -1 {
			urlPart := link[idx+4:]
			if endIdx := strings.Index(urlPart, "&"); endIdx != -1 {
				urlPart = urlPart[:endIdx]
			}
			decoded, err := url.QueryUnescape(urlPart)
			if err == nil {
				u, err := url.Parse(decoded)
				if err == nil && u.Host != "" {
					host := strings.TrimPrefix(u.Host, "www.")
					return strings.Split(host, ":")[0]
				}
			}
		}
	}
	return ""
}

// clickAdByLink кликает по рекламе по ссылке (Ctrl+Click для открытия в новой вкладке)
func clickAdByLink(page *osciris.Page, adLink string) error {
	// Ищем элемент по ссылке
	selector := fmt.Sprintf("a[href*='%s'], a[data-rw*='%s']", adLink, adLink)
	
	// Проверяем наличие элемента
	if !page.FastCheckElement(selector) {
		return fmt.Errorf("ad element not found for link: %s", adLink)
	}

	// Получаем координаты элемента
	box, err := page.GetElementBox(selector)
	if err != nil {
		return fmt.Errorf("failed to get element box: %w", err)
	}

	// Вычисляем центр элемента
	x := (box.Content[0] + box.Content[2]) / 2
	y := (box.Content[1] + box.Content[5]) / 2

	// Выполняем Ctrl+Click
	return page.MouseClickCtrl(x, y)
}

