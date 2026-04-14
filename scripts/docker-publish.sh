#!/bin/bash
set -e

# Hector Docker Image Publishing Script
# Usage: ./scripts/docker-publish.sh [registry] [version]
#   registry: dockerhub, ghcr, or both (default: both)
#   version:  semantic version like v1.0.0 (default: latest)

# Configuration
REGISTRY=${1:-both}
VERSION=${2:-latest}

# Docker Hub configuration
# IMPORTANT: Set DOCKERHUB_USER environment variable or edit this line
DOCKERHUB_USER=${DOCKERHUB_USER:-}
DOCKERHUB_IMAGE="${DOCKERHUB_USER}/hector"

# GitHub Container Registry configuration
GITHUB_ORG=${GITHUB_ORG:-verikod}
GHCR_IMAGE="ghcr.io/$GITHUB_ORG/hector"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Validate Docker Hub username if publishing to Docker Hub
if [ "$REGISTRY" = "dockerhub" ] || [ "$REGISTRY" = "both" ]; then
    if [ -z "$DOCKERHUB_USER" ]; then
        print_error "DOCKERHUB_USER environment variable not set!"
        echo ""
        echo "To publish to Docker Hub, you must set your Docker Hub username:"
        echo "  export DOCKERHUB_USER=your-docker-username"
        echo ""
        echo "Or use GHCR instead:"
        echo "  ./scripts/docker-publish.sh ghcr"
        exit 1
    fi
fi

# Change to hector directory (script can be run from anywhere)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
HECTOR_DIR="$(dirname "$SCRIPT_DIR")"
cd "$HECTOR_DIR"

print_header "Hector Docker Publishing"
echo "Registry: $REGISTRY"
echo "Version: $VERSION"
echo "Location: $HECTOR_DIR"
echo ""

# Extract version info if it's a git tag
if [ "$VERSION" = "latest" ] && git describe --tags --exact-match 2>/dev/null; then
    GIT_VERSION=$(git describe --tags --exact-match)
    print_info "Git tag detected: $GIT_VERSION"
    read -p "Use git tag version instead of 'latest'? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        VERSION=$GIT_VERSION
    fi
fi

# Function to build and push to Docker Hub
publish_dockerhub() {
    print_header "Publishing to Docker Hub"

    # Check if logged in (check config.json for docker.io auth)
    if ! grep -q "index.docker.io" ~/.docker/config.json 2>/dev/null; then
        print_error "Not logged in to Docker Hub"
        echo "Please run: docker login"
        return 1
    fi

    print_info "Building image: $DOCKERHUB_IMAGE:$VERSION"

    # Build the image
    docker build \
        -t "$DOCKERHUB_IMAGE:$VERSION" \
        -t "$DOCKERHUB_IMAGE:latest" \
        --label "org.opencontainers.image.source=https://github.com/$GITHUB_ORG/hector" \
        --label "org.opencontainers.image.description=Hector RAG Server" \
        --label "org.opencontainers.image.version=$VERSION" \
        .

    print_success "Build complete"

    # Push the images
    print_info "Pushing to Docker Hub..."
    docker push "$DOCKERHUB_IMAGE:$VERSION"

    if [ "$VERSION" != "latest" ]; then
        docker push "$DOCKERHUB_IMAGE:latest"
        print_success "Pushed: $DOCKERHUB_IMAGE:$VERSION and $DOCKERHUB_IMAGE:latest"
    else
        print_success "Pushed: $DOCKERHUB_IMAGE:latest"
    fi

    echo ""
    print_success "Published to Docker Hub!"
    echo "Pull with: docker pull $DOCKERHUB_IMAGE:$VERSION"
    echo ""
}

# Function to build and push to GHCR
publish_ghcr() {
    print_header "Publishing to GitHub Container Registry"

    # Check if logged in to GHCR
    if ! grep -q "ghcr.io" ~/.docker/config.json 2>/dev/null; then
        print_error "Not logged in to GHCR"
        echo "Please run: echo \$GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin"
        return 1
    fi

    print_info "Building image: $GHCR_IMAGE:$VERSION"

    # Build the image
    docker build \
        -t "$GHCR_IMAGE:$VERSION" \
        -t "$GHCR_IMAGE:latest" \
        --label "org.opencontainers.image.source=https://github.com/$GITHUB_ORG/hector" \
        --label "org.opencontainers.image.description=Hector RAG Server" \
        --label "org.opencontainers.image.version=$VERSION" \
        .

    print_success "Build complete"

    # Push the images
    print_info "Pushing to GHCR..."
    docker push "$GHCR_IMAGE:$VERSION"

    if [ "$VERSION" != "latest" ]; then
        docker push "$GHCR_IMAGE:latest"
        print_success "Pushed: $GHCR_IMAGE:$VERSION and $GHCR_IMAGE:latest"
    else
        print_success "Pushed: $GHCR_IMAGE:latest"
    fi

    echo ""
    print_success "Published to GitHub Container Registry!"
    echo "Pull with: docker pull $GHCR_IMAGE:$VERSION"
    echo ""
    print_info "To make the image public:"
    echo "  1. Go to https://github.com/$GITHUB_ORG?tab=packages"
    echo "  2. Click on 'hector' package"
    echo "  3. Package settings → Change visibility → Public"
    echo ""
}

# Main execution
case "$REGISTRY" in
    dockerhub)
        publish_dockerhub
        ;;
    ghcr)
        publish_ghcr
        ;;
    both)
        publish_dockerhub
        echo ""
        publish_ghcr
        ;;
    *)
        print_error "Unknown registry: $REGISTRY"
        echo "Usage: $0 [dockerhub|ghcr|both] [version]"
        echo ""
        echo "Examples:"
        echo "  $0 dockerhub          # Publish latest to Docker Hub"
        echo "  $0 ghcr v1.0.0        # Publish v1.0.0 to GHCR"
        echo "  $0 both v1.0.0        # Publish v1.0.0 to both registries"
        exit 1
        ;;
esac

# Summary
print_header "Summary"
echo "Registry: $REGISTRY"
echo "Version: $VERSION"
echo ""
echo "Update your hector-cloud .env file with:"
if [ "$REGISTRY" = "ghcr" ]; then
    echo "  HECTOR_IMAGE=$GHCR_IMAGE:$VERSION"
elif [ "$REGISTRY" = "dockerhub" ]; then
    echo "  HECTOR_IMAGE=$DOCKERHUB_IMAGE:$VERSION"
else
    echo "  HECTOR_IMAGE=$GHCR_IMAGE:$VERSION  (GHCR - recommended)"
    echo "  or"
    echo "  HECTOR_IMAGE=$DOCKERHUB_IMAGE:$VERSION  (Docker Hub)"
fi
echo ""
print_success "Done! 🎉"
