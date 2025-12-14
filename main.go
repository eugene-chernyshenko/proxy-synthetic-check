package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

// hasScheme проверяет, содержит ли строка схему URL, используя url.Parse
func hasScheme(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != ""
}

// maskAuth скрывает пароль в URL для безопасного вывода
func maskAuth(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	return u.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <target_url> [socks5_proxy]")
		fmt.Println("Example: go run main.go https://httpbin.org/ip socks5://127.0.0.1:1080")
		os.Exit(1)
	}

	targetURL := os.Args[1]
	var proxyURL string

	if len(os.Args) >= 3 {
		proxyURL = os.Args[2]
	} else {
		// Использовать переменную окружения, если указана
		proxyURL = os.Getenv("SOCKS5_PROXY")
	}

	// Создаем HTTP клиент
	var client *http.Client

	if proxyURL != "" {
		// Если схема не указана, добавляем socks5://
		if !hasScheme(proxyURL) {
			proxyURL = "socks5://" + proxyURL
		}

		fmt.Printf("Используется SOCKS5 прокси: %s\n", maskAuth(proxyURL))

		// Парсим URL прокси
		proxyURI, err := url.Parse(proxyURL)
		if err != nil {
			fmt.Printf("Ошибка парсинга URL прокси: %v\n", err)
			os.Exit(1)
		}

		// Извлекаем адрес прокси (host:port)
		proxyAddr := proxyURI.Host
		if proxyAddr == "" {
			fmt.Printf("Ошибка: не указан адрес прокси (host:port)\n")
			os.Exit(1)
		}

		// Извлекаем учетные данные для аутентификации
		var auth *proxy.Auth
		if proxyURI.User != nil {
			password, _ := proxyURI.User.Password()
			auth = &proxy.Auth{
				User:     proxyURI.User.Username(),
				Password: password,
			}
		}

		// Создаем SOCKS5 dialer
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
		if err != nil {
			fmt.Printf("Ошибка создания SOCKS5 dialer: %v\n", err)
			os.Exit(1)
		}

		// Создаем HTTP транспорт с SOCKS5 dialer
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}

		client = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	} else {
		fmt.Println("Прокси не указан, используется прямое соединение")
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Выполняем запрос
	fmt.Printf("Выполняется запрос к: %s\n", targetURL)
	resp, err := client.Get(targetURL)
	if err != nil {
		fmt.Printf("Ошибка выполнения запроса: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Ошибка чтения ответа: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nСтатус ответа: %s\n", resp.Status)
	fmt.Printf("\nЗаголовки ответа:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	fmt.Printf("\nТело ответа:\n%s\n", string(body))
}
