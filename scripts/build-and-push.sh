#!/bin/bash

# Build and push script for schedule-watcher
set -e

# Load environment variables from .env file if it exists
if [[ -f .env ]]; then
    echo "Loading environment variables from .env file..."
    # Export variables from .env file
    set -a  # automatically export all variables
    source .env
    set +a  # stop automatically exporting
fi

# Configuration
REGISTRY=${REGISTRY:-"docker.io"}  # Default to Docker Hub
USERNAME=${DOCKER_USERNAME:-"your-username"}
IMAGE_NAME="schedule-watcher"
VERSION=${VERSION:-"latest"}

# Full image name
FULL_IMAGE_NAME="${REGISTRY}/${USERNAME}/${IMAGE_NAME}:${VERSION}"

echo "Building multi-architecture image: ${FULL_IMAGE_NAME}"

# Check if buildx is available and create builder if needed
if ! docker buildx ls | grep -q multiarch; then
    echo "Creating multi-architecture builder..."
    docker buildx create --name multiarch --use
    docker buildx inspect --bootstrap
else
    docker buildx use multiarch
fi

# Build multi-architecture image (AMD64 and ARM64)
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -t "${FULL_IMAGE_NAME}" \
    --push \
    .

echo "Image built successfully!"

# Multi-arch build automatically pushes, so we're done
echo "Multi-architecture image pushed successfully!"

# Also tag and push as latest if version is specified
if [[ "${VERSION}" != "latest" ]]; then
    LATEST_IMAGE="${REGISTRY}/${USERNAME}/${IMAGE_NAME}:latest"
    echo "Building and pushing latest tag: ${LATEST_IMAGE}"
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --build-arg BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        -t "${LATEST_IMAGE}" \
        --push \
        .
    echo "Latest tag pushed!"
fi

# Verify image was pushed successfully
echo "🔍 Verifying image push..."
if docker manifest inspect "${FULL_IMAGE_NAME}" > /dev/null 2>&1; then
    echo "✅ Image verified in registry!"
    
    # Ask if user wants to deploy to NAS
    read -p "Deploy to NAS now? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🚀 Deploying to NAS..."
        ssh teknonas 'bash -s' < ./scripts/deploy.sh
    fi
else
    echo "❌ Failed to verify image in registry!"
    exit 1
fi

echo "Done!"