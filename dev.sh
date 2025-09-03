#!/bin/bash

# Development script to rebuild and restart the service

echo "🔄 Stopping existing containers..."
docker-compose down

echo "🔨 Building with no cache..."
docker-compose build --no-cache

echo "🚀 Starting services..."
docker-compose up -d

echo "📊 Showing logs..."
docker-compose logs -f