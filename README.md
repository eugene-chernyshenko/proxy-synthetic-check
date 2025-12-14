# HTTP запросы через SOCKS5 прокси на Go

Простая программа на Go для выполнения HTTP запросов через SOCKS5 прокси.

## Установка зависимостей

```bash
go mod download
```

## Использование

### С указанием прокси в аргументах:

```bash
go run main.go <target_url> <socks5_proxy>
```

Пример:

```bash
go run main.go https://httpbin.org/ip socks5://127.0.0.1:1080
```

### С использованием переменной окружения:

```bash
export SOCKS5_PROXY=socks5://127.0.0.1:1080
go run main.go https://httpbin.org/ip
```

### Без прокси (прямое соединение):

```bash
go run main.go https://httpbin.org/ip
```

## Примеры

Получить ваш IP адрес через прокси:

```bash
go run main.go https://httpbin.org/ip socks5://127.0.0.1:1080
```

Запрос с аутентификацией (если прокси поддерживает):

```bash
# Формат: socks5://username:password@host:port
go run main.go https://example.com socks5://user:pass@127.0.0.1:1080
```

## Сборка

Для создания исполняемого файла:

```bash
go build -o http-proxy main.go
```

Затем запуск:

```bash
./http-proxy https://httpbin.org/ip socks5://127.0.0.1:1080
```
