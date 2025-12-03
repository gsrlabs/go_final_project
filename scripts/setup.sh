#!/bin/bash

echo "🔧 Настройка Todo Scheduler..."

# Создаем .env файл если не существует
if [ ! -f .env ]; then
    echo "📝 Создаю .env файл..."
    cat > .env << EOF
TODO_PORT=7540
TODO_DBFILE=data/scheduler.db
TODO_PASSWORD=mysecretpassword123
EOF
    echo "✅ .env файл создан"
else
    echo "✅ .env файл уже существует"
fi

# Даем права на скрипты
echo "🔐 Настраиваю права доступа..."
chmod +x scripts/*.sh

# Опциональная сборка Docker
echo ""
read -p "Собрать Docker образ? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🐳 Сборка Docker образа..."
    if docker compose build --no-cache; then
        echo "✅ Docker образ собран"
    else
        echo "⚠️ Не удалось собрать Docker образ"
    fi
else
    echo "ℹ️ Docker образ можно собрать позже: docker compose build"
fi

echo ""
echo "🎉 Настройка завершена!"
echo "🚀 Запустите приложение: ./scripts/run.sh dev"
echo "🐳 Или через Docker: ./scripts/run.sh start"