#!/bin/bash

# Deployment script - builds, pushes, and deploys to Synology NAS
set -e

echo "🚀 Starting deployment..."

# First, build and push the image
echo "🔨 Building and pushing Docker image..."
if [[ -f ./scripts/build-and-push.sh ]]; then
    # Run build script but skip the interactive deploy prompt
    echo "n" | ./scripts/build-and-push.sh
else
    echo "❌ build-and-push.sh not found. Please run from project root."
    exit 1
fi

# Check if build was successful
if [[ $? -ne 0 ]]; then
    echo "❌ Build and push failed. Aborting deployment."
    exit 1
fi

echo "📡 Deploying to NAS via SSH..."
ssh teknonas 'bash -s' << 'REMOTE_SCRIPT'
#!/bin/bash
set -e

# Add /usr/local/bin to PATH for Docker
export PATH="/usr/local/bin:$PATH"

echo "🚀 Starting deployment on NAS..."
cd /volume1/docker/schedule_watcher

echo "📦 Pulling latest images..."
docker compose pull

echo "🛑 Stopping current containers..."
docker compose down

echo "▶️  Starting updated containers..."
docker compose up -d

echo "⏳ Waiting for container to be ready..."
sleep 10

if docker compose ps | grep -q "Up"; then
    echo "✅ Deployment successful!"
    echo "🌐 Application is running at: http://$(hostname -I | awk '{print $1}'):8081"
else
    echo "❌ Deployment failed - container not running"
    docker compose logs --tail=50
    exit 1
fi

echo "🧹 Cleaning up old images..."
docker image prune -f

echo "🎉 Deployment completed!"
REMOTE_SCRIPT