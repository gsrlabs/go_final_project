#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups"
DB_FILE="data/scheduler.db"

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

error() { echo -e "${RED}❌ $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️ $1${NC}"; }
info() { echo -e "${BLUE}🔹 $1${NC}"; }

# Проверка аргументов
if [ -z "$1" ]; then
    error "Укажите файл бекапа"
    echo "Использование: $0 <backup_file.db>"
    echo ""
    
    # Показываем доступные бекапы
    if [ -d "$BACKUP_DIR" ]; then
        info "Доступные бекапы:"
        sudo find "$BACKUP_DIR" -name "*.db" -type f -printf "  %f\n" 2>/dev/null | sort -r | head -10
        echo ""
        LATEST_BACKUP=$(sudo find "$BACKUP_DIR" -name "*.db" -type f -printf "%T@ %f\n" 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
        if [ -n "$LATEST_BACKUP" ]; then
            info "Пример: sudo $0 $LATEST_BACKUP"
        fi
    else
        info "Директория бекапов не существует: $BACKUP_DIR"
    fi
    exit 1
fi

BACKUP_FILE="$1"
[[ "$BACKUP_FILE" != *"/"* ]] && BACKUP_FILE="$BACKUP_DIR/$BACKUP_FILE"

# Проверка файла
if [ ! -f "$BACKUP_FILE" ]; then
    error "Файл не найден: $BACKUP_FILE"
    
    if [ ! -d "$BACKUP_DIR" ]; then
        error "Директория бекапов не существует: $BACKUP_DIR"
    fi
    
    # Показываем похожие файлы
    SIMILAR_FILES=$(sudo find "$BACKUP_DIR" -name "*$(basename "$BACKUP_FILE")*" -type f 2>/dev/null | head -5)
    if [ -n "$SIMILAR_FILES" ]; then
        info "Возможно вы имели в виду:"
        echo "$SIMILAR_FILES"
    fi
    exit 1
fi

# Проверяем права sudo
if [ "$EUID" -ne 0 ]; then
    error "Требуются права root для доступа к базе данных"
    info "Запустите: sudo $0 $1"
    exit 1
fi

# Подтверждение
echo ""
warning "╔══════════════════════════════════════════════════╗"
warning "║               ВНИМАНИЕ! ОПАСНО!                 ║"
warning "║      Все текущие задачи будут удалены!          ║"
warning "╚══════════════════════════════════════════════════╝"
echo ""
read -p "Вы уверены, что хотите продолжить? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    info "Восстановление отменено"
    exit 0
fi

# Останавливаем приложение
info "Останавливаем приложение..."
docker compose down

# Создаем резервную копию текущей базы (на всякий случай)
if [ -f "$DB_FILE" ]; then
    info "Создаю резервную копию текущей базы..."
    cp "$DB_FILE" "$DB_FILE.backup.$(date +%Y%m%d_%H%M%S)"
    success "Резервная копия создана"
fi

# Восстановление
info "Восстановление данных..."
START_TIME=$(date +%s)

if cp "$BACKUP_FILE" "$DB_FILE"; then
    DURATION=$(( $(date +%s) - START_TIME ))
    success "Данные восстановлены за ${DURATION}с"
    
    # Устанавливаем правильные права
    chmod 644 "$DB_FILE"
    chown $SUDO_USER:$SUDO_USER "$DB_FILE" 2>/dev/null || true
    
    # Проверяем что файл не пустой
    if [ -s "$DB_FILE" ]; then
        success "Файл базы данных проверен (не пустой)"
    else
        warning "Восстановленный файл базы данных пуст"
    fi
else
    error "Ошибка при восстановлении данных"
    # Пытаемся восстановить из backup
    LATEST_BACKUP=$(find . -name "scheduler.db.backup.*" -type f -printf "%T@ %p\n" 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
    if [ -n "$LATEST_BACKUP" ]; then
        warning "Восстанавливаю из резервной копии..."
        cp "$LATEST_BACKUP" "$DB_FILE"
    fi
    exit 1
fi

# Запускаем приложение
info "Запускаем приложение..."
docker compose up -d

# Финальная информация
echo ""
success "╔══════════════════════════════════════════════════╗"
success "║           ВОССТАНОВЛЕНИЕ ЗАВЕРШЕНО!             ║"
success "╚══════════════════════════════════════════════════╝"
echo ""
info "📁 Файл: $(basename "$BACKUP_FILE")"
info "📏 Размер: $(du -h "$BACKUP_FILE" | cut -f1)"
info "⏱️ Время восстановления: ${DURATION}с"
echo ""
info "🌐 Приложение запущено и доступно на http://localhost:7540"