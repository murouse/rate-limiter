# Архитектура Protobuf/gRPC контрактов в Go-сервисах

В данном документе описана архитектура организации API-контрактов, конфигурация инструмента **Buf** и процесс кодогенерации для экосистемы, использующей кастомные библиотеки расширений (на примере `rate_limiter`).

---

## 1. Архитектура директорий и Buf Workspace

В основе проектирования лежит концепция **Buf Workspace (рабочего пространства)**. Вместо плоской структуры или ручного управления путями через флаги `-I` (`--proto_path`) компилятора, репозиторий делится на изолированные логические модули.

### Структура проекта (`hookah-culture`)

```text
.
├── api/                                      # Модуль бизнес-логики
│   └── auth/
│       └── v1/
│           └── service.proto
├── third_party/                              # Модуль внешних зависимостей
│   └── murouse/
│       └── rate_limiter/
│           └── v1/
│               └── rate_limiter.proto
├── pkg/
│   └── api/                                  # Сюда складывается сгенерированный Go-код
├── buf.yaml                                  # Конфигурация воркспейса
└── buf.gen.yaml                              # Конфигурация плагинов генерации

```

### Описание модулей в `buf.yaml`

Файл `buf.yaml` регистрирует корни для поиска Protobuf-файлов.

```yaml
version: v2
modules:
  - path: api          # Собственные контракты сервиса
  - path: third_party  # Контракты внешних библиотек

```

> **Почему именно так?**
> При компиляции `buf` виртуально объединяет папки `api/` и `third_party/` в единый корень. Это избавляет от необходимости писать относительные пути вроде `import "../../../third_party/..."` и позволяет использовать чистые, переносимые импорты.

---

## 2. Проектирование импортов и разрешение типов

### Файл расширения (Библиотека `rate_limiter`)

Файл расположен по пути `third_party/murouse/rate_limiter/v1/rate_limiter.proto`. Имя корневой папки внутри `third_party` (`murouse`) выступает в роли пространства имен (Namespace) для защиты от коллизий.

```protobuf
syntax = "proto3";

package murouse.rate_limiter.v1;

// Указывает Go-плагину, где искать готовый скомпилированный код этой либы
option go_package = "github.com/murouse/rate-limiter;rate_limiter";

import "google/protobuf/descriptor.proto";
import "google/protobuf/duration.proto";

message Rule { ... }

// Расширяем стандартные опции Protobuf
extend google.protobuf.MethodOptions {
  repeated Rule rules = 51234;
}
extend google.protobuf.FieldOptions {
  string rate_key = 51235;
}

```

### Файл бизнес-логики (`service.proto`)

Расположен в `api/auth/v1/service.proto`. Благодаря Buf Workspace, импорт внешней библиотеки пишется от виртуального корня:

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "gitlab.com/murouse/hookah-culture/pkg/api/auth/v1;authv1";

// Импорт без префикса "third_party/"
import "murouse/rate_limiter/v1/rate_limiter.proto"; 

service AuthService {
  rpc SendCode(SendCodeRequest) returns (SendCodeResponse) {
    // Использование кастомной опции rate_limiter
    option (murouse.rate_limiter.v1.rules) = {
      name: "10_per_min"
      limit: 2
      window: { seconds: 60 }
    };
  }
}

message SendCodeRequest {
  // Использование опции для поля
  string phone = 1 [(murouse.rate_limiter.v1.rate_key) = "phone"];
}

```

---

## 3. Процесс кодогенерации (`buf.gen.yaml`)

Конфигурация определяет, как виртуальные схемы преобразуются в физические Go-файлы.

```yaml
version: v2
managed:
  enabled: false # Выключено, так как go_package управляется вручную в .proto

inputs:
  - directory: api # Генерируем код ТОЛЬКО для файлов из папки api

plugins:
  - local: bin/protoc-gen-go
    out: pkg/api
    opt:
      - paths=source_relative

```

### Как плагины обрабатывают пути (Механизм `source_relative`)

Для генерации используется связка параметра `out: pkg/api` и флага `paths=source_relative`.

1. `buf` берет файл `api/auth/v1/service.proto`.
2. Так как корень инпута — папка `api`, относительный путь файла равен `auth/v1/service.proto`.
3. Флаг `paths=source_relative` приказывает плагину: *«Сохрани этот относительный путь при создании `.pb.go` файла»*.
4. На выходе получаем: `pkg/api/` + `auth/v1/service.proto` $\rightarrow$ `pkg/api/auth/v1/service.pb.go`.

### Как разрешаются Go-импорты внешних библиотек

В процессе генерации `service.pb.go` плагин встречает типы из `rate_limiter.proto`. Ему нужно знать, какой Go-пакет импортировать в коде.

1. Плагин заглядывает в `third_party/murouse/rate_limiter/v1/rate_limiter.proto`.
2. Находит строку `option go_package = "[github.com/murouse/rate-limiter;rate_limiter](https://github.com/murouse/rate-limiter;rate_limiter)";`.
3. В сгенерированный файл `service.pb.go` в блок `import` автоматически подставляется внешняя зависимость:

```go
// pkg/api/auth/v1/service.pb.go
import (
    // ...
    rate_limiter "github.com/murouse/rate-limiter"
)

```

> **Важно:** Файлы из `third_party` **не генерируются** локально в вашем сервисе. Воркспейс лишь считывает их схемы для валидации. Сам исполняемый код `rate-limiter` подтягивается стандартным Go-менеджером зависимостей: `go get [github.com/murouse/rate-limiter](https://github.com/murouse/rate-limiter)`.

---

## 4. Почему сделано именно так (Rationale)

### 1. Отсутствие флагов маппинга (`-M` или `Mfilename=package`)

В старых подходах приходилось в `buf.gen.yaml` для каждого плагина дублировать длинные строки переопределения пакетов:
`opt: ["Mmurouse/rate_limiter/v1/rate_limiter.proto=[github.com/murouse/rate-limiter;rate_limiter](https://github.com/murouse/rate-limiter;rate_limiter)"]`.

Текущая схема использует `go_package` внутри самого `.proto`-файла в `third_party`. Это единый источник правды (Single Source of Truth), который поддерживается компилятором автоматически.

### 2. Совместимость с gRPC Reflection и рантаймом

В рантайме Go (при парсинге опций в middleware или при работе утилит вроде Evans/Postman) поиск дескрипторов файлов происходит по их **логическому пути импорта внутри Protobuf**.

* Твой сервис думает, что файл называется: `"murouse/rate_limiter/v1/rate_limiter.proto"`.
* Оригинальная библиотека `rate-limiter` при компиляции зарегистрировала себя под именем: `"murouse/rate_limiter/v1/rate_limiter.proto"`.

Поскольку пути совпали символ-в-символ, gRPC рефлексия и `proto.GetExtension` работают без ошибок «unknown extension».

### 3. Защита от коллизий имен (Namespacing)

Если в будущем в проект потребуется внедрить еще один rate limiter от другой команды или вендора, они не столкнутся в папке `third_party`, так как изолированы на уровне файловой структуры и Protobuf-пакетов:

* `third_party/murouse/rate_limiter/...` (`package murouse.rate_limiter.v1;`)
* `third_party/vendor/rate_limiter/...` (`package vendor.rate_limiter.v1;`)