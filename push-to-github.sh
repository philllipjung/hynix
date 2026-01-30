#!/bin/bash
# Simple script to push using Personal Access Token

read -p "Enter GitHub Personal Access Token: " TOKEN

cd /root/hynix

# Configure git
git config user.email "yohaim1511@naver.com"
git config user.name "yohaim1511"

# Add remote with token (or create repo first)
echo ""
echo "First, create repository at: https://github.com/new"
echo "Name: hynix"
echo "Then press Enter to continue..."
read

# Add remote
git remote add origin https://yohaim1511@naver.com:$TOKEN@github.com/yohaim1511/hynix.git 2>/dev/null || \
git remote set-url origin https://yohaim1511@naver.com:$TOKEN@github.com/yohaim1511/hynix.git

# Push
git push -u origin main
