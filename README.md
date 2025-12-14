# HTTP запросы через SOCKS5 прокси на Go

Простая программа на Go для выполнения HTTP запросов через SOCKS5 прокси. Прокси **обязателен** и берется из переменных окружения или `.env` файла.

## Установка зависимостей

```bash
go mod download
```

## Использование

### Настройка прокси

Прокси настраивается через переменную окружения `SOCKS5_PROXY` или через `.env` файл (для dev окружения).

#### Вариант 1: Переменная окружения

```bash
export SOCKS5_PROXY=socks5://127.0.0.1:1080
go run main.go https://httpbin.org/ip
```

#### Вариант 2: .env файл (для dev)

Создайте файл `.env` в корне проекта:

```env
SOCKS5_PROXY=socks5://127.0.0.1:1080
```

Затем запустите:

```bash
go run main.go https://httpbin.org/ip
```

### Запуск программы

```bash
go run main.go <target_url>
```

Пример:

```bash
go run main.go https://httpbin.org/ip
```

## Примеры

### Базовый запрос

```bash
go run main.go https://httpbin.org/ip
```

### Запрос с аутентификацией прокси

В `.env` файле или переменной окружения:

```env
SOCKS5_PROXY=socks5://username:password@proxy.example.com:1080
```

Или:

```bash
export SOCKS5_PROXY=socks5://username:password@proxy.example.com:1080
go run main.go https://example.com
```

### Формат URL прокси

- Без аутентификации: `socks5://host:port` или `host:port`
- С аутентификацией: `socks5://username:password@host:port`

## Сборка

Для создания исполняемого файла:

```bash
go build -o http-proxy main.go
```

Затем запуск:

```bash
./http-proxy https://httpbin.org/ip
```

## Требования

- Прокси **обязателен** - программа не будет работать без него
- Переменная окружения `SOCKS5_PROXY` или файл `.env` должны быть настроены
