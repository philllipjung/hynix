#!/bin/bash
# Create GitHub Personal Access Token and Repository

set -e

USERNAME="yohaim1511"
EMAIL="yohaim1511@naver.com"
REPO_NAME="hynix"

echo "=========================================="
echo "  GitHub Repository Setup for Hynix"
echo "=========================================="
echo ""
echo "GitHub deprecated password authentication in 2021."
echo "You need to create a Personal Access Token (PAT)."
echo ""
echo "STEPS:"
echo ""
echo "1. Open: https://github.com/settings/tokens"
echo "2. Click: 'Generate new token' → 'Generate new token (classic)'"
echo "3. Set:"
echo "   - Note: hynix-server"
echo "   - Expiration: No expiration (or your choice)"
echo "   - Scopes: ✅ repo (full control)"
echo "4. Click 'Generate token'"
echo "5. Copy the token (starts with ghp_...)"
echo ""
echo "=========================================="
echo ""

read -p "Enter your GitHub Personal Access Token: " TOKEN

if [ -z "$TOKEN" ]; then
    echo "Error: Token is required"
    exit 1
fi

echo ""
echo "Creating repository..."

# Create repository
RESPONSE=$(curl -s -X POST \
  -H "Authorization: token $TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/user/repos \
  -d "{
    \"name\": \"$REPO_NAME\",
    \"description\": \"Hynix Microservice Project with CI/CD\",
    \"private\": false,
    \"auto_init\": false
  }")

# Check for errors
if echo "$RESPONSE" | grep -q "Bad credentials"; then
    echo "❌ Error: Invalid token"
    exit 1
fi

if echo "$RESPONSE" | grep -q "name already exists"; then
    echo "⚠️  Repository already exists, continuing..."
else
    echo "✅ Repository created!"
fi

echo ""
echo "Configuring git..."

cd /root/hynix

# Configure git
git config user.email "$EMAIL"
git config user.name "$USERNAME"

# Add remote (or update if exists)
if git remote get-url origin &>/dev/null; then
    git remote set-url origin "git@github.com:$USERNAME/$REPO_NAME.git"
    echo "✅ Remote updated"
else
    git remote add origin "git@github.com:$USERNAME/$REPO_NAME.git"
    echo "✅ Remote added"
fi

# Setup SSH agent
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/github_hynix 2>/dev/null || true

echo ""
echo "Pushing to GitHub..."

# Push to GitHub
git push -u origin main

echo ""
echo "=========================================="
echo "  ✅ Setup Complete!"
echo "=========================================="
echo ""
echo "Repository URL: https://github.com/$USERNAME/$REPO_NAME"
echo ""
echo "GitHub Actions will run on push and increment"
echo "the build number in config/config.json"
echo ""
