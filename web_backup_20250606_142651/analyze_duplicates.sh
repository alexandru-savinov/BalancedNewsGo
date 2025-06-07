#!/bin/bash
# Web Folder Duplicate Analysis Script
# Analyzes and reports duplicate and redundant files

echo "🔍 Web Folder Duplicate Analysis"
echo "================================"
echo ""

# Function to calculate directory size
get_dir_size() {
    if [ -d "$1" ]; then
        du -sh "$1" 2>/dev/null | cut -f1
    else
        echo "N/A"
    fi
}

# Function to count files in directory
count_files() {
    if [ -d "$1" ]; then
        find "$1" -type f | wc -l
    else
        echo "0"
    fi
}

echo "📁 Directory Size Analysis:"
echo "├── js/          $(get_dir_size js/) ($(count_files js/) files)"
echo "├── static/js/   $(get_dir_size static/js/) ($(count_files static/js/) files)"
echo "├── css/         $(get_dir_size css/) ($(count_files css/) files)"
echo "├── static/css/  $(get_dir_size static/css/) ($(count_files static/css/) files)"
echo "├── dist/        $(get_dir_size dist/) ($(count_files dist/) files)"
echo "└── coverage/    $(get_dir_size coverage/) ($(count_files coverage/) files)"
echo ""

# Check for JavaScript duplicates
echo "🔄 JavaScript File Duplicates:"
if [ -d "js" ] && [ -d "static/js" ]; then
    js_duplicates=$(find js/ static/js/ -name "*.js" -exec basename {} \; | sort | uniq -d | wc -l)
    echo "Found $js_duplicates duplicate JavaScript file names"
    
    if [ $js_duplicates -gt 0 ]; then
        echo "Duplicate files:"
        find js/ static/js/ -name "*.js" -exec basename {} \; | sort | uniq -d | head -10
        if [ $js_duplicates -gt 10 ]; then
            echo "... and $((js_duplicates - 10)) more"
        fi
    fi
else
    echo "No duplicate JavaScript directories found"
fi
echo ""

# Check for CSS duplicates
echo "🎨 CSS File Duplicates:"
if [ -d "css" ] && [ -d "static/css" ]; then
    css_duplicates=$(find css/ static/css/ -name "*.css" -exec basename {} \; | sort | uniq -d | wc -l)
    echo "Found $css_duplicates duplicate CSS file names"
    
    if [ $css_duplicates -gt 0 ]; then
        echo "Duplicate files:"
        find css/ static/css/ -name "*.css" -exec basename {} \; | sort | uniq -d
    fi
else
    echo "No duplicate CSS directories found"
fi
echo ""

# Check for backup files
echo "📋 Backup Files:"
backup_files=$(find . -name "*.backup" -o -name "*.bak" -o -name "*~" 2>/dev/null)
if [ -n "$backup_files" ]; then
    echo "$backup_files" | while read -r file; do
        if [ -f "$file" ]; then
            size=$(ls -lh "$file" | awk '{print $5}')
            echo "├── $file ($size)"
        fi
    done
else
    echo "No backup files found"
fi
echo ""

# Check for demo/test HTML files
echo "🎭 Demo/Test HTML Files:"
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

total_demo_size=0
demo_count=0
for file in "${demo_files[@]}"; do
    if [ -f "$file" ]; then
        size_bytes=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
        size_human=$(ls -lh "$file" | awk '{print $5}')
        echo "├── $file ($size_human)"
        total_demo_size=$((total_demo_size + size_bytes))
        ((demo_count++))
    fi
done

if [ $demo_count -eq 0 ]; then
    echo "No demo files found"
else
    total_demo_mb=$(echo "scale=1; $total_demo_size / 1024 / 1024" | bc 2>/dev/null || echo "~$(($total_demo_size / 1024 / 1024))")
    echo "└── Total: $demo_count files (~${total_demo_mb}MB)"
fi
echo ""

# Calculate potential savings
echo "💾 Potential Space Savings:"
js_size=0
css_size=0

if [ -d "static/js" ]; then
    js_size=$(du -sb static/js 2>/dev/null | cut -f1 || echo "0")
fi

if [ -d "static/css" ]; then
    css_size=$(du -sb static/css 2>/dev/null | cut -f1 || echo "0")
fi

total_duplicate_size=$((js_size + css_size))
if [ $total_duplicate_size -gt 0 ]; then
    total_mb=$(echo "scale=1; $total_duplicate_size / 1024 / 1024" | bc 2>/dev/null || echo "~$(($total_duplicate_size / 1024 / 1024))")
    echo "├── Duplicate static/ directory: ~${total_mb}MB"
fi

if [ $total_demo_size -gt 0 ]; then
    demo_mb=$(echo "scale=1; $total_demo_size / 1024 / 1024" | bc 2>/dev/null || echo "~$(($total_demo_size / 1024 / 1024))")
    echo "├── Demo files (can be organized): ~${demo_mb}MB"
fi

if [ -d "dist" ]; then
    dist_size=$(du -sb dist 2>/dev/null | cut -f1 || echo "0")
    if [ $dist_size -gt 0 ]; then
        dist_mb=$(echo "scale=1; $dist_size / 1024 / 1024" | bc 2>/dev/null || echo "~$(($dist_size / 1024 / 1024))")
        echo "├── Build artifacts (dist/): ~${dist_mb}MB"
    fi
fi

echo ""
echo "🛠️  Recommended Actions:"
echo "1. Run './cleanup_duplicates.sh' to clean up duplicates"
echo "2. Verify functionality after cleanup"
echo "3. Update .gitignore to exclude build artifacts"
echo "4. Consider moving demos to separate directory"
