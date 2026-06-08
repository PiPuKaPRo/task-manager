# Task Manager API

## Описание

REST API для управления задачами на Go. Реализовано 10 эндпоинтов для CRUD операций, фильтрации и назначения исполнителей.

## Технологии

- Go 1.21
- PostgreSQL
- Стандартная библиотека + errgroup

## Требования

- Go 1.21 или выше
- PostgreSQL 12 или выше

## Инструкция по запуску

### 1. Установка PostgreSQL

Скачайте и установите PostgreSQL с официального сайта:
https://www.postgresql.org/download/windows/

При установке задайте пароль: `0000`

### 2. Создание базы данных

Откройте **SQL Shell (psql)** или **pgAdmin** и выполните команду:

```sql
CREATE DATABASE taskdb;
```

### 3. Клонирование проекта

```bash
git clone https://github.com/PiPuKaPRo/task-manager.git
cd task-manager
```

### 4. Установка зависимостей

```bash
go mod download
go mod tidy
```

### 5. Создание таблицы

Подключитесь к базе данных и выполните SQL:

```sql
CREATE TABLE IF NOT EXISTS tasks (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    assigned_to VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### 6. Запуск сервера

```bash
go run cmd/server/main.go
```

После запуска вы увидите:

```
[APP] Starting Task Manager API...
[APP] Database connected successfully
[APP] Server starting on http://localhost:8080
```

### 1. Проверка здоровья сервера

```bash
curl http://localhost:8080/health
```

**Ответ:** `{"data":{"status":"ok"}}`

### 2. Создание задачи

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy milk","description":"2 liters","assigned_to":"ivan@mail.com"}'
```

### 3. Получение всех задач

```bash
curl http://localhost:8080/tasks
```

### 4. Получение задачи по ID

```bash
curl http://localhost:8080/tasks/1
```

### 5. Обновление задачи

```bash
curl -X PUT http://localhost:8080/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy bread"}'
```

### 6. Отметка как выполненной

```bash
curl -X POST http://localhost:8080/tasks/1/done
```

### 7. Отметка как невыполненной

```bash
curl -X POST http://localhost:8080/tasks/1/undone
```

### 8. Назначение исполнителя

```bash
curl -X POST http://localhost:8080/tasks/1/assign \
  -H "Content-Type: application/json" \
  -d '{"assigned_to":"petr@mail.com"}'
```

### 9. Фильтрация по статусу

```bash
curl "http://localhost:8080/tasks?status=pending"
```

### 10. Удаление задачи

```bash
curl -X DELETE http://localhost:8080/tasks/1
```

## Запуск тестов

```bash
# Запуск всех тестов
go test -v ./...

# Проверка покрытия кода
go test -cover ./...
```

## Остановка сервера

Нажмите `Ctrl+C` в терминале с сервером. Сервер завершит работу корректно (graceful shutdown).

## Список всех эндпоинтов

| Метод | Эндпоинт | Описание |
|-------|----------|----------|
| GET | `/health` | Проверка здоровья |
| POST | `/tasks` | Создать задачу |
| GET | `/tasks` | Список задач |
| GET | `/tasks/{id}` | Получить задачу |
| PUT | `/tasks/{id}` | Обновить задачу |
| DELETE | `/tasks/{id}` | Удалить задачу |
| POST | `/tasks/{id}/done` | Выполнить |
| POST | `/tasks/{id}/undone` | Отменить выполнение |
| POST | `/tasks/{id}/assign` | Назначить исполнителя |
| GET | `/tasks/status/{status}` | Фильтр по статусу |

## Возможные ошибки

### Ошибка подключения к БД

В файле `cmd/server/main.go` найдите строку:

```go
connStr := "host=localhost port=5432 user=postgres password=0000 dbname=taskdb sslmode=disable"
```
Измените `0000` на ваш пароль.

### Порт 8080 занят

Измените порт в файле `.env`:

```env
SERVER_PORT=8081
```