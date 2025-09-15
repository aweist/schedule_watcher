#!/bin/bash

# Deployment script for Synology NAS
set -e

echo "🚀 Starting deployment..."

# Check if running locally or on NAS
if [[ -n "$SSH_CLIENT" ]] || [[ -n "$SSH_TTY" ]] || [[ "$(hostname)" != *"synology"* && "$(hostname)" != *"nas"* ]]; then
    # Running via SSH - execute on remote
    echo "📡 Detected local execution - running on NAS via SSH..."
    ssh teknonas 'bash -s' << 'REMOTE_SCRIPT'
#!/bin/bash
set -e
echo "🚀 Starting deployment on NAS..."
cd /volume1/docker/schedule_watcher
docker compose pull
docker compose down
docker compose up -d
sleep 10
if docker compose ps | grep -q "Up"; then
    echo "✅ Deployment successful!"
    echo "🌐 Application is running at: http://$(hostname -I | awk '{print $1}'):8081"
else
    echo "❌ Deployment failed - container not running"
    docker compose logs --tail=50
    exit 1
fi
docker image prune -f
echo "🎉 Deployment completed!"
REMOTE_SCRIPT
    exit $?
fi

# Navigate to app directory (when running directly on NAS)
cd /volume1/docker/schedule_watcher

# Pull latest images
echo "📦 Pulling latest images..."
docker compose pull

# Stop current containers
echo "🛑 Stopping current containers..."
docker compose down

# Start updated containers
echo "▶️  Starting updated containers..."
docker compose up -d

# Wait for container to be ready
echo "⏳ Waiting for container to be ready..."
sleep 10

# Check if container is running
if docker compose ps | grep -q "Up"; then
    echo "✅ Deployment successful!"
    echo "🌐 Application is running at: http://$(hostname -I | awk '{print $1}'):8081"
else
    echo "❌ Deployment failed - container not running"
    echo "📋 Container logs:"
    docker compose logs --tail=50
    exit 1
fi

# Clean up old images to save space
echo "🧹 Cleaning up old images..."
docker image prune -f

echo "🎉 Deployment completed!"