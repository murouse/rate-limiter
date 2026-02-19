# RateLimiter

`rate-limiter` — это гибкая библиотека для **fixed-window rate limiting** в gRPC-сервисах на Go.  
Она позволяет ограничивать количество запросов на уровне метода и глобально, используя кастомные хранилища (в памяти, Redis и др.), с поддержкой protobuf-опций для RPC.

---

## Особенности

- Fixed-window семантика (TTL устанавливается только при первом инкременте ключа)
- Поддержка нескольких правил на метод (например: "1 запрос в минуту" + "10 запросов в день")
- Глобальные правила, действующие на все методы
- Простое подключение через gRPC interceptor
- Расширяемый storage backend через интерфейс `Cache`
- Поддержка Redis с атомарным INCR+EXPIRE
- Настраиваемый ключ (user ID, IP, tenant и т.д.)
- Легко интегрируется с protobuf-опциями для методов RPC

---

## Установка

```bash
go get github.com/murouse/rate-limiter


---

## Быстрый старт

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/murouse/rate-limiter"
	"google.golang.org/grpc"
)

func main() {
	// Инициализируем RateLimiter с кастомными опциями
	rl := ratelimiter.New(
		ratelimiter.WithNamespace("my-service"),
		ratelimiter.WithGlobalLimitRules([]ratelimiter.RateLimitRule{
			{Name: "global_per_minute", Limit: 1000, Window: time.Minute},
		}),
	)

	// Создаём gRPC сервер с interceptor
	server := grpc.NewServer(
		grpc.UnaryInterceptor(rl.UnaryServerInterceptor()),
	)

	_ = server // регистрируем сервисы, запускаем сервер
}
```

---

## Использование с protobuf

В `.proto` можно задавать лимиты прямо на методах RPC:

```proto
syntax = "proto3";

package myservice;

import "google/protobuf/duration.proto";
import "google/protobuf/descriptor.proto";
import "rate_limiter/rate_limiter.proto";

service MyService {
  rpc MyMethod(Request) returns (Response) {
    option (rate_limiter.rate_limits) = {
      name: "per_minute"
      limit: 1
      window: { seconds: 60 }
    };
    option (rate_limiter.rate_limits) = {
      name: "per_day"
      limit: 10
      window: { seconds: 86400 }
    };
  }
}

message Request {}
message Response {}
```

Интерсептор автоматически извлекает правила из protobuf и применяет их.

---

## Кастомизация

### Кастомный `Cache`

Интерфейс `Cache` позволяет использовать любые storage backend:

```go
type Cache interface {
    Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}
```

* TTL должен устанавливаться **только при первом инкременте** (fixed-window semantics).
* Atomicity обязательна (Redis Lua script, CAS и др.).

### Кастомный ключ

Можно передавать кастомный `RateKeyExtractor`:

```go
type RateKeyExtractor interface {
    ExtractRateKey(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error)
}
```

Пример: извлечение user ID из токена.

### Формат ключа

Можно изменить формат ключа с помощью `WithKeyFormatterFunc`:

```go
func customFormatter(namespace, rateKey, fullMethod, ruleName string) string {
    return fmt.Sprintf("%s|%s|%s|%s", namespace, rateKey, fullMethod, ruleName)
}
```

---

## Интерсептор

`UnaryServerInterceptor` проверяет:

1. Глобальные правила
2. Правила метода (из protobuf)

Если лимит превышен — возвращает `ResourceExhausted`.

---

## Fixed-window semantics

* Счётчик увеличивается при каждом запросе
* TTL устанавливается **только при первом increment**
* После окончания окна счётчик сбрасывается
* Гарантирует predictable, fixed-time rate limiting, без скользящего окна

---

## Пример Redis Cache

```go
import (
    "context"
    "time"

    "github.com/go-redis/redis/v9"
)

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

redisCache := NewRedisCache(rdb) // Реализация с INCR + PEXPIRE
rl := ratelimiter.New(ratelimiter.WithCache(redisCache))
```

---

## Лицензия

MIT License © 2026

```
