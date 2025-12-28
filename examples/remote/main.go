package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/vitaliitsarov/osciris"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	fp "github.com/vitaliitsarov/fingerprint-injector-go"
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
		Timeout:     60 * time.Second, // Увеличенный таймаут для навигации
		Fingerprint: fp.NewChrome119Android(),
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

			// Проверяем текущий URL перед навигацией
			var currentURL string
			err = page.URL(&currentURL)
			if err != nil {
				log.Printf("Warning: failed to get current URL: %v", err)
			} else {
				log.Printf("Current URL: %s", currentURL)
			}

			// Навигация только если не на Google
			needNavigation := currentURL == "" || (!strings.Contains(currentURL, "google.com") && !strings.Contains(currentURL, "google."))
			if needNavigation {
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
				}
			} else {
				log.Println("Already on Google, skipping navigation and continuing...")
				// Небольшая задержка для стабилизации
				time.Sleep(1 * time.Second)
			}

			// Продолжаем работу независимо от результата навигации
			{
				// Быстрая проверка наличия кнопки "Принять все" (Accept All) БЕЗ ожидания
				log.Println("Quick check for Accept All button...")
				
				// Используем оптимизированный метод с таймаутом 500ms
				if page.FastCheckElement("#L2AGLb") {
					log.Println("Accept All button found, clicking...")
					
					// Пробуем быстрый клик
					err = page.Click("#L2AGLb")
					if err != nil {
						err = page.ClickWithScroll("#L2AGLb")
					}
					if err == nil {
						log.Println("Accept All button clicked successfully")
						time.Sleep(500 * time.Millisecond)
					} else {
						log.Printf("Failed to click Accept All button: %v", err)
					}
				} else {
					log.Println("Accept All button not found, skipping modal handling")
				}


				// Проверяем, нужно ли вводить текст (если уже на странице результатов поиска, пропускаем)
				var currentURL string
				page.URL(&currentURL)
				isSearchResultsPage := strings.Contains(currentURL, "/search") || strings.Contains(currentURL, "?q=")
				
				if !isSearchResultsPage {
					// Быстрая проверка поля поиска и ввод текста
					log.Println("Checking for search input...")
					if page.FastCheckElement("[name='q']") {
						log.Println("Search input found")
						
						// Вводим текст "lego" быстро с таймаутом
						log.Println("Typing 'lego' in search field...")
						done := make(chan error, 1)
						go func() {
							done <- page.SendKeys("[name='q']", "lego")
						}()
						
						select {
						case err = <-done:
							if err != nil {
								log.Printf("Error typing text: %v", err)
							} else {
								log.Println("Text 'lego' entered successfully")
								// Небольшая задержка после ввода
								time.Sleep(200 * time.Millisecond)
								
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
						case <-time.After(2 * time.Second):
							log.Printf("Timeout typing text (2s), continuing anyway...")
							// Пробуем нажать Enter даже если ввод не завершился
							page.SendKeysEnter("[name='q']")
						}
					} else {
						log.Println("Search input not found, skipping text input")
					}
				} else {
					log.Println("Already on search results page, skipping text input")
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

				// Сохраняем ID текущей вкладки перед кликом по рекламе
				originalTabID := tabBrowser.GetTargetID()
				log.Printf("Original tab ID: %s", originalTabID)

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
							log.Println("Ad clicked successfully! Waiting 5 seconds...")
							
							// Ждем 5 секунд
							time.Sleep(5 * time.Second)
							
							// Получаем список вкладок для поиска новой вкладки
							tabs, err := browser.ListTabs()
							if err != nil {
								log.Printf("Error getting tabs list: %v", err)
							} else {
								// Находим новую вкладку (не исходную и не прикрепленную к нашему контексту)
								var adTabID osciris.Tab
								for _, tab := range tabs {
									if tab.ID != originalTabID && !tab.Attached {
										// Это новая вкладка с рекламой (не прикреплена к нашему контексту)
										adTabID = tab
										break
									}
								}
								
								// Если не нашли неприкрепленную, ищем любую другую
								if adTabID.ID == "" {
									for _, tab := range tabs {
										if tab.ID != originalTabID {
											adTabID = tab
											break
										}
									}
								}
								
								if adTabID.ID != "" {
									log.Printf("Found ad tab: %s (URL: %s)", adTabID.ID, adTabID.URL)
									
									// Пробуем закрыть вкладку через подключение к ней и выполнение window.close()
									log.Printf("Attempting to close ad tab: %s", adTabID.ID)
									adTabBrowser, err := browser.ConnectToTab(adTabID.ID)
									if err == nil {
										// Подключились к вкладке, пробуем закрыть через JavaScript
										adPage := adTabBrowser.NewPage()
										var result interface{}
										err = adPage.Evaluate("window.close()", &result)
										if err != nil {
											log.Printf("Error closing tab via JavaScript: %v", err)
										} else {
											log.Println("Ad tab closed via window.close()")
										}
										// Закрываем контекст подключения к вкладке
										adTabBrowser.Close()
										time.Sleep(500 * time.Millisecond)
									} else {
										log.Printf("Could not connect to ad tab to close it: %v", err)
										// Пробуем закрыть через CloseTabByID
										err = browser.CloseTabByID(adTabID.ID)
										if err != nil {
											log.Printf("Error closing tab via CloseTabByID: %v", err)
										} else {
											log.Println("Ad tab closed successfully via CloseTabByID")
										}
									}
								} else {
									log.Println("Ad tab not found in tabs list (might be already closed)")
								}
								
								// Подключаемся обратно к исходной вкладке Google
								log.Printf("Reconnecting to original Google tab: %s", originalTabID)
								originalTabBrowser, err := browser.ConnectToTab(originalTabID)
								if err != nil {
									log.Printf("Error reconnecting to original tab: %v", err)
								} else {
									log.Println("Reconnected to Google tab successfully")
									// Обновляем page для работы с исходной вкладкой
									page = originalTabBrowser.NewPage()
									
									// Небольшая задержка для стабилизации
									time.Sleep(1 * time.Second)
								}
							}
						}
					}
				}
			} // Конец блока работы с вкладкой
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
	uniqueLinks := make(map[string]bool) // Используем уникальность по ссылке, а не по домену

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

		log.Printf("Processing element %d/%d (NodeID: %d)", i+1, len(adNodes), node.NodeID)

		// Прокручиваем к элементу для гарантии его видимости
		err = page.RunActions(chromedp.ActionFunc(func(ctx context.Context) error {
			// Получаем box model для элемента
			box, err := dom.GetBoxModel().WithNodeID(node.NodeID).Do(ctx)
			if err != nil {
				log.Printf("Warning: failed to get box model for element %d: %v", i+1, err)
				return nil // Продолжаем даже если не удалось получить box model
			}

			// Прокручиваем к элементу
			y := box.Content[1]
			if err := input.DispatchMouseEvent(input.MouseWheel, 0, 0).
				WithDeltaY(float64(y - 400)). // Прокручиваем так, чтобы элемент был виден
				Do(ctx); err != nil {
				log.Printf("Warning: failed to scroll to element %d: %v", i+1, err)
				return nil // Продолжаем даже если прокрутка не удалась
			}
			time.Sleep(100 * time.Millisecond)
			return nil
		}))

		// Получаем атрибуты элемента через DOM API
		var link, domain string
		attrs, err := page.GetElementAttributes(node.NodeID)
		if err != nil {
			log.Printf("Warning: failed to get attributes for element %d: %v", i+1, err)
			continue // Пропускаем элемент если не удалось получить атрибуты
		}

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

		log.Printf("Element %d: link='%s', domain='%s'", i+1, link, domain)

		// Фильтруем и добавляем рекламу
		if link != "" && strings.Contains(link, "/aclk") {
			// Фильтруем нерелевантные ссылки
			if strings.Contains(link, "google.com/search") ||
				strings.Contains(link, "google.com/url") ||
				strings.Contains(link, "google.com/webhp") {
				log.Printf("Element %d: skipped (irrelevant link)", i+1)
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

			// Проверяем уникальность по ссылке (а не по домену, так как может быть несколько объявлений с одного домена)
			if !uniqueLinks[link] {
				uniqueLinks[link] = true
				results = append(results, AdInfo{
					Link:   link,
					Domain: domain,
				})
				log.Printf("✓ Added ad %d: Domain=%s, Link=%s", len(results), domain, link)
			} else {
				log.Printf("Element %d: skipped (duplicate link)", i+1)
			}
		} else {
			log.Printf("Element %d: skipped (no valid link or not an ad)", i+1)
		}
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
	log.Printf("Looking for ad element with link: %s", adLink)
	
	// Ищем элемент по ссылке - пробуем разные варианты селекторов
	selectors := []string{
		fmt.Sprintf("a[href*='%s']", adLink),
		fmt.Sprintf("a[data-rw*='%s']", adLink),
		fmt.Sprintf("a[href='%s']", adLink),
		fmt.Sprintf("a[data-rw='%s']", adLink),
	}
	
	var foundSelector string
	for _, sel := range selectors {
		if page.FastCheckElement(sel) {
			foundSelector = sel
			log.Printf("Found ad element with selector: %s", sel)
			break
		}
	}
	
	if foundSelector == "" {
		return fmt.Errorf("ad element not found for link: %s", adLink)
	}

	// Прокручиваем к элементу, чтобы он был виден на экране перед кликом
	log.Println("Scrolling to ad element to make it visible...")
	err := page.ScrollIntoView(foundSelector)
	if err != nil {
		log.Printf("Warning: failed to scroll to element: %v", err)
	}
	time.Sleep(500 * time.Millisecond) // Даем время на прокрутку

	// Используем новый метод ClickOnNewTab для открытия в новой вкладке
	log.Println("Clicking on ad to open in new tab...")
	err = page.ClickOnNewTab(foundSelector)
	if err != nil {
		return fmt.Errorf("failed to click ad: %w", err)
	}
	
	log.Println("Ad clicked successfully, waiting for new tab to open...")
	time.Sleep(1 * time.Second) // Даем время на открытие новой вкладки
	
	return nil
}


