#!/bin/bash

set -e

# Get the level (major, minor, patch)
LEVEL=$1

if [ -z "$LEVEL" ]; then
    echo "Usage: $0 <major|minor|patch>"
    echo "Example: $0 patch"
    exit 1
fi

# Validate level
if [[ "$LEVEL" != "major" && "$LEVEL" != "minor" && "$LEVEL" != "patch" ]]; then
    echo "Error: Level must be 'major', 'minor', or 'patch'"
    exit 1
fi

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

echo "Current version: $LATEST_TAG"

# Extract version numbers
VERSION=${LATEST_TAG#v}
IFS='.' read -r -a VERSION_PARTS <<< "$VERSION"

MAJOR=${VERSION_PARTS[0]:-0}
MINOR=${VERSION_PARTS[1]:-0}
PATCH=${VERSION_PARTS[2]:-0}

# Bump the appropriate version
case $LEVEL in
    "major")
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    "minor")
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    "patch")
        PATCH=$((PATCH + 1))
        ;;
esac

NEW_VERSION="v${MAJOR}.${MINOR}.${PATCH}"

echo "New version: $NEW_VERSION"

# Check if we're on a clean working directory
if [[ -n $(git status --porcelain) ]]; then
    echo "Error: Working directory is not clean. Please commit or stash changes."
    exit 1
fi

# Update the version in the code
echo "Updating version in cmd/root.go..."
sed -i.bak "s/const Version = \"v[0-9]*\.[0-9]*\.[0-9]*\"/const Version = \"$NEW_VERSION\"/" cmd/root.go

# Verify the update was successful
if ! grep -q "const Version = \"$NEW_VERSION\"" cmd/root.go; then
    echo "Error: Failed to update version in cmd/root.go"
    # Restore backup
    mv cmd/root.go.bak cmd/root.go
    exit 1
fi

# Remove backup file
rm cmd/root.go.bak

# Commit the version update
echo "Committing version update..."
git add cmd/root.go
git commit -m "Bump version to $NEW_VERSION"

# Create and push the tag
echo "Creating tag $NEW_VERSION..."
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

echo "Pushing changes and tag to origin..."
git push origin HEAD
git push origin "$NEW_VERSION"

echo "✅ Successfully updated, committed, and pushed version $NEW_VERSION"