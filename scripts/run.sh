#!/bin/bash

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
COMPOSE="docker compose"
APP="todo-scheduler"

# Функции
success() { echo -e "${GREEN} $1${NC}"; }
info()    { echo -e "${BLUE} $1${NC}"; }
warning() { echo -e "${YELLOW} $1${NC}"; }
error()   { echo -e "${RED} $1${NC}"; }

# Получить порт из .env или использовать по умолчанию
get_port() {
    if [ -f ".env" ]; then
        # Ищем TODO_PORT в .env файле
        PORT=$(grep -E "^TODO_PORT=" .env | cut -d'=' -f2)
        if [ -n "$PORT" ]; then
            echo "$PORT"
            return
        fi
    fi
    # Если не нашли - используем 7540
    echo "7540"
}

# Показать статус
show_status() {
    echo ""
    info "Статус контейнеров:"
    $COMPOSE ps
    echo ""
}

# Показать подсказку
show_help() {
    echo ""
    info "Управление проектом Todo Scheduler"
    echo ""
    echo -e "  ${YELLOW}run.sh start${NC}          — Запустить контейнер"
    echo -e "  ${YELLOW}run.sh stop${NC}           — Остановить контейнер"
    echo -e "  ${YELLOW}run.sh restart${NC}        — Перезапустить"
    echo -e "  ${YELLOW}run.sh rebuild${NC}        — Пересобрать и запустить"
    echo -e "  ${YELLOW}run.sh logs${NC}           — Логи в реальном времени"
    echo -e "  ${YELLOW}run.sh status${NC}         — Показать статус"
    echo ""
    echo -e "  ${YELLOW}run.sh dev${NC}            — Запуск без Docker (go run)"
    echo -e "  ${YELLOW}run.sh test${NC}           — Запуск тестов"
    echo ""
    echo -e "  ${YELLOW}run.sh down-clean${NC}     — УДАЛИТЬ данные (ОСТОРОЖНО!)"
    echo ""
    info "Пример: ./scripts/run.sh start"
    echo ""
}

# === ОСНОВНАЯ ЛОГИКА ===
case "$1" in

    "start")
        PORT=$(get_port)
        echo "Запуск Todo Scheduler на порту $PORT..."
        $COMPOSE up -d
        sleep 2
        show_status
        success "Приложение запущено на http://localhost:$PORT"
        ;;

    "stop")
        echo "Остановка приложения (данные сохранены)..."
        $COMPOSE stop
        success "Приложение остановлено"
        ;;

    "restart")
        PORT=$(get_port)
        echo "Перезапуск приложения на порту $PORT..."
        $COMPOSE restart
        sleep 2
        show_status
        success "Перезапуск завершён"
        ;;

    "rebuild")
        PORT=$(get_port)
        echo "Пересборка и запуск на порту $PORT..."
        $COMPOSE down
        $COMPOSE up --build -d
        sleep 2
        show_status
        success "Приложение пересобрано и запущено на http://localhost:$PORT"
        ;;

    "logs")
        echo "Логи приложения (Ctrl+C для выхода)..."
        $COMPOSE logs -f "$APP"
        ;;

    "status")
        show_status
        ;;

    "dev")
        echo "Запуск в режиме разработки (без Docker)..."
        if [ ! -f ".env" ]; then
            warning "Файл .env не найден, создаю базовый..."
            cp .env.example .env 2>/dev/null || echo "TODO_PORT=7540" > .env
        fi
        PORT=$(get_port)
        echo "Порт: $PORT"
        go run main.go
        ;;

    "test")
        echo "Запуск тестов..."
        
        # Автоматическое исправление прав для тестов
        if [ -d data ]; then
            echo "🔧 Временное расширение прав для тестов..."
            sudo chmod -R 777 data/
        fi
        
        # Запускаем тесты
        go test ./tests/...
        TEST_RESULT=$?
        
        # Восстанавливаем безопасные права (опционально)
        # if [ -d data ]; then
        #     echo "🔧 Восстановление прав доступа..."
        #     sudo chmod -R 755 data/
        #     if [ -f data/scheduler.db ]; then
        #         sudo chmod 644 data/scheduler.db
        #     fi
        # fi
        
        exit $TEST_RESULT
        ;;

    # === ОПАСНАЯ КОМАНДА ===
    "down-clean")
        warning "ВНИМАНИЕ! Это УДАЛИТ ВСЕ ДАННЫЕ (базу задач)!"
        warning "Все задачи будут потеряны!"
        read -p "Введите 'YES' для подтверждения: " confirm
        if [[ "$confirm" == "YES" ]]; then
            echo "Удаление контейнеров и данных..."
            $COMPOSE down -v
            sudo rm -rf data/
            success "Данные полностью очищены"
        else
            error "Отменено"
        fi
        ;;

    "")
        show_help
        ;;

    *)
        error "Неизвестная команда: $1"
        show_help
        exit 1
        ;;

esac