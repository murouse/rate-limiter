# Архитектура Protobuf-опций: Использование внешней библиотеки без паники рантайма

При использовании кастомных Protobuf-опций (аннотаций) из общей библиотеки (например, `rate-limiter`), компиляция их `.proto`-файла внутри бизнес-сервиса приводит к ошибке:

```text
panic: proto: extension number 51234 is already registered...

```

Это происходит из-за **двойной регистрации расширения** в глобальном реестре Go: один раз из кода библиотеки (`go.mod`), второй раз — из локально сгенерированного кода проекта.

---

## Суть решения

1. **В проекте:** Оставляем только `.proto`-файл опций. Он нужен исключительно как декларация для компилятора и IDE.
2. **В генераторе (`buf`):** Запрещаем компилировать этот файл в локальный Go-код, но заставляем подменять ссылки на внешнюю библиотеку.
3. **В рантайме:** Проект использует скомпилированные структуры напрямую из библиотеки через `go.mod`. Регистрация происходит один раз, паника исчезает.

---

## Реализация

### 1. Подготовка `.proto`-файлов

Разместите копию `.proto`-файла из библиотеки в папку `api/third_party/`. **Пакет и `go_package` должны строго совпадать с библиотечными.**

```protobuf
// api/third_party/rate_limiter/v1/rate_limiter.proto
syntax = "proto3";

package rate_limiter; // Оригинальный пакет библиотеки
option go_package = "github.com/murouse/rate-limiter;rate_limiter"; // Ссылка на репозиторий либы

import "google/protobuf/descriptor.proto";
import "google/protobuf/duration.proto";

message Rule {
  string name = 1;
  int32 limit = 2;
  google.protobuf.Duration window = 3;
}

extend google.protobuf.MethodOptions { repeated Rule rules = 51234; }
extend google.protobuf.FieldOptions { string rate_key = 51235; }

```

Теперь импортируйте этот файл в свой сервис и вешайте аннотации, используя префикс пакета `rate_limiter`:

```protobuf
// api/auth/v1/service.proto
syntax = "proto3";
package auth.v1;

option go_package = "gitlab.com/murouse/hookah-culture/pkg/api/auth/v1;authv1";

import "third_party/rate_limiter/v1/rate_limiter.proto"; // Импорт локальной копии

service AuthService {
  rpc SendCode(SendCodeRequest) returns (SendCodeResponse) {
    // Применяем опцию через оригинальный пакет rate_limiter
    option (rate_limiter.rules) = {
      name: "10_per_min"
      limit: 2
      window: { seconds: 60 }
    };
  };
}

message SendCodeRequest {
  string phone = 1 [(rate_limiter.rate_key) = "phone"];
}

```

---

### 2. Настройка генерации (`buf.gen.yaml` v2)

Используем секцию `inputs` для отсечения лишнего кода и флаг `-M` для подмены Go-импортов в сгенерированных файлах.

```yaml
version: v2
managed:
  enabled: false

inputs:
  - directory: .
    exclude_paths:
      - api/third_party # Запрещаем генерировать rate_limiter.pb.go локально

plugins:
  - local: bin/protoc-gen-go
    out: pkg/api
    opt: 
      - paths=source_relative
      # Формат: -M{путь_импорта_в_proto}={путь_импорта_в_go};{имя_пакета}
      - Mthird_party/rate_limiter/v1/rate_limiter.proto=github.com/murouse/rate-limiter;rate_limiter

  - local: bin/protoc-gen-go-grpc
    out: pkg/api
    opt: 
      - paths=source_relative
      - Mthird_party/rate_limiter/v1/rate_limiter.proto=github.com/murouse/rate-limiter;rate_limiter

  - local: bin/protoc-gen-grpc-gateway
    out: pkg/api
    opt:
      - paths=source_relative
      - logtostderr=true
      - generate_unbound_methods=true
      - Mthird_party/rate_limiter/v1/rate_limiter.proto=github.com/murouse/rate-limiter;rate_limiter

```

---

## Как это работает при сборке

1. **`buf generate`** читает `service.proto` и видит импорт `third_party/.../rate_limiter.proto`.
2. Благодаря правильному `package` и наличию файла, линтер и компилятор успешно валидируют структуру аннотаций.
3. Блок `exclude_paths` предотвращает создание локального файла `pkg/api/third_party/rate_limiter.pb.go`.
4. Флаг `-M` перехватывает генерацию `service.pb.go`. Вместо локального пути он прописывает в Go-импорты внешнюю зависимость:
```go
// pkg/api/auth/v1/service.pb.go
import rate_limiter "github.com/murouse/rate-limiter"

```
5. При запуске `go run` рантайм загружает код опций один раз — из библиотеки, подключенной через `go.mod`. Конфликт номеров полностью исключен.
