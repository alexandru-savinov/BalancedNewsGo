const fs = require('fs');
const path = require('path');

// Icon mapping from class names to data-icon names
const iconMap = {
    'icon-home': 'home',
    'icon-newspaper': 'newspaper',
    'icon-balance': 'balance',
    'icon-chart': 'chart',
    'icon-sources': 'sources',
    'icon-clock': 'clock',
    'icon-search': 'search',
    'icon-filter': 'filter',
    'icon-alert': 'alert',
    'icon-chevron-left': 'chevronLeft',
    'icon-chevron-right': 'chevronRight',
    'icon-refresh': 'refresh',
    'icon-document': 'document',
    'icon-menu': 'menu',
    'icon-close': 'close'
};

function fixIconsInFile(filePath) {
    console.log(`Fixing icons in: ${filePath}`);
    let content = fs.readFileSync(filePath, 'utf8');
    let changed = false;

    for (const [oldClass, newIcon] of Object.entries(iconMap)) {
        const oldPattern = new RegExp(`<i class="${oldClass}[^"]*"></i>`, 'g');
        const newPattern = `<span data-icon="${newIcon}" class="icon"></span>`;

        if (content.includes(`<i class="${oldClass}`)) {
            content = content.replace(oldPattern, newPattern);
            changed = true;
            console.log(`  Replaced ${oldClass} with ${newIcon}`);
        }
    }

    if (changed) {
        fs.writeFileSync(filePath, content, 'utf8');
        console.log(`  Updated: ${filePath}`);
    } else {
        console.log(`  No changes needed: ${filePath}`);
    }
}

// Fix icons in template files
const templateFiles = [
    'web/templates/base.html',
    'web/templates/index.html'
];

templateFiles.forEach(file => {
    const fullPath = path.join(__dirname, file);
    if (fs.existsSync(fullPath)) {
        fixIconsInFile(fullPath);
    } else {
        console.log(`File not found: ${fullPath}`);
    }
});

console.log('Icon fixing complete!');
