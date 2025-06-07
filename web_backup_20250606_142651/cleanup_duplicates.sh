#!/bin/bash
# Web Folder Cleanup Script
# Removes duplicate and redundant files identified in the analysis

set -e  # Exit on any error

echo "🧹 Starting Web Folder Cleanup..."
echo "This script will:"
echo "1. Remove duplicate static/ directory"
echo "2. Remove backup files"
echo "3. Organize demo files"
echo "4. Clean build artifacts"
echo ""

# Confirm before proceeding
read -p "Do you want to proceed? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Cleanup cancelled."
    exit 1
fi

# Create backup directory first
echo "📦 Creating backup of current state..."
mkdir -p ../web_backup_$(date +%Y%m%d_%H%M%S)
cp -r . ../web_backup_$(date +%Y%m%d_%H%M%S)/

# 1. Remove duplicate static/ directory
echo "🗑️  Removing duplicate static/ directory..."
if [ -d "static" ]; then
    du -sh static/
    rm -rf static/
    echo "✅ Removed static/ directory"
else
    echo "ℹ️  static/ directory not found"
fi

# 2. Remove backup files
echo "🗑️  Removing backup files..."
if [ -f "js/components/Modal.js.backup" ]; then
    ls -la js/components/Modal.js.backup
    rm js/components/Modal.js.backup
    echo "✅ Removed Modal.js.backup"
else
    echo "ℹ️  Modal.js.backup not found"
fi

# 3. Organize demo files
echo "📁 Organizing demo files..."
if [ ! -d "demos" ]; then
    mkdir demos
fi

# Move demo files to demos directory
demo_files=(
    "bias-slider-demo.html"
    "debug-test.html"
    "performance-demo.html"
    "performance-test.html"
    "performance-validation.html"
    "performance-validation-complete.html"
    "progress-indicator-demo.html"
    "test-enhanced-articles.html"
    "test-navigation-icons.html"
    "test-performance-fix.html"
)

moved_count=0
for file in "${demo_files[@]}"; do
    if [ -f "$file" ]; then
        mv "$file" demos/
        echo "✅ Moved $file to demos/"
        ((moved_count++))
    fi
done

if [ $moved_count -eq 0 ]; then
    echo "ℹ️  No demo files found to move"
    rmdir demos 2>/dev/null || true
else
    echo "✅ Moved $moved_count demo files to demos/ directory"
fi

# 4. Clean build artifacts
echo "🗑️  Cleaning build artifacts..."
if [ -d "dist" ]; then
    du -sh dist/
    rm -rf dist/
    echo "✅ Removed dist/ directory (will be regenerated during next build)"
else
    echo "ℹ️  dist/ directory not found"
fi

if [ -d "coverage" ]; then
    du -sh coverage/
    rm -rf coverage/
    echo "✅ Removed coverage/ directory (will be regenerated during tests)"
else
    echo "ℹ️  coverage/ directory not found"
fi

# 5. Clean temporary files
echo "🗑️  Cleaning temporary files..."
find . -name "*.log" -type f -delete 2>/dev/null || true
find . -name ".DS_Store" -type f -delete 2>/dev/null || true
find . -name "Thumbs.db" -type f -delete 2>/dev/null || true

echo ""
echo "🎉 Cleanup completed successfully!"
echo ""
echo "📊 Summary:"
echo "- Removed duplicate static/ directory"
echo "- Removed backup files"
if [ $moved_count -gt 0 ]; then
    echo "- Organized $moved_count demo files into demos/ directory"
fi
echo "- Cleaned build artifacts"
echo ""
echo "📁 Updated web folder structure:"
ls -la
echo ""
echo "💡 Next steps:"
echo "1. Run 'npm run build' to regenerate dist/ directory"
echo "2. Run 'npm test' to regenerate coverage reports"
echo "3. Verify everything works correctly"
