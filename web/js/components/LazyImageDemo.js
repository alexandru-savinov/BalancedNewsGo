// LazyImage Demo Component
// Demonstrates image lazy loading with performance monitoring

class LazyImageDemo extends HTMLElement {
  constructor() {
    super();
    this.attachShadow({ mode: 'open' });
    this.render();
  }

  render() {
    this.shadowRoot.innerHTML = `
      <style>
        :host {
          display: block;
          margin: 20px 0;
        }

        .image-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
          gap: 15px;
        }

        .image-card {
          background: white;
          border-radius: 8px;
          overflow: hidden;
          box-shadow: 0 2px 8px rgba(0,0,0,0.1);
          transition: transform 0.2s ease;
        }

        .image-card:hover {
          transform: translateY(-2px);
        }

        .image-container {
          position: relative;
          width: 100%;
          height: 200px;
          background: #f0f0f0;
          overflow: hidden;
        }

        .lazy-image {
          width: 100%;
          height: 100%;
          object-fit: cover;
          transition: opacity 0.3s ease;
          opacity: 0;
        }

        .lazy-image.loaded {
          opacity: 1;
        }

        .lazy-image.error {
          background: #ffebee;
        }

        .image-placeholder {
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          color: #999;
          font-size: 14px;
        }

        .loading-spinner {
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          width: 24px;
          height: 24px;
          border: 2px solid #e0e0e0;
          border-top: 2px solid #007acc;
          border-radius: 50%;
          animation: spin 1s linear infinite;
        }

        @keyframes spin {
          to { transform: translate(-50%, -50%) rotate(360deg); }
        }

        .image-info {
          padding: 15px;
        }

        .image-title {
          font-weight: 600;
          margin-bottom: 5px;
          color: #333;
        }

        .image-description {
          color: #666;
          font-size: 14px;
          line-height: 1.4;
        }

        .performance-badge {
          position: absolute;
          top: 10px;
          right: 10px;
          background: rgba(0, 122, 204, 0.9);
          color: white;
          padding: 4px 8px;
          border-radius: 12px;
          font-size: 12px;
          font-weight: 500;
        }
      </style>

      <div class="image-grid">
        ${this.generateImageCards()}
      </div>
    `;

    this.initializeLazyLoading();
  }

  generateImageCards() {
    const images = [
      {
        title: "Mountain Landscape",
        description: "Beautiful mountain vista with morning light",
        src: "https://picsum.photos/400/300?random=1",
        placeholder: "🏔️"
      },
      {
        title: "Ocean Waves",
        description: "Peaceful ocean scene with rolling waves",
        src: "https://picsum.photos/400/300?random=2",
        placeholder: "🌊"
      },
      {
        title: "Forest Path",
        description: "Serene forest trail in autumn colors",
        src: "https://picsum.photos/400/300?random=3",
        placeholder: "🌲"
      },
      {
        title: "City Skyline",
        description: "Modern cityscape at golden hour",
        src: "https://picsum.photos/400/300?random=4",
        placeholder: "🏙️"
      },
      {
        title: "Desert Sunset",
        description: "Dramatic sunset over desert dunes",
        src: "https://picsum.photos/400/300?random=5",
        placeholder: "🌅"
      },
      {
        title: "Arctic Landscape",
        description: "Pristine arctic wilderness with ice",
        src: "https://picsum.photos/400/300?random=6",
        placeholder: "🧊"
      }
    ];

    return images.map((image, index) => `
      <div class="image-card">
        <div class="image-container">
          <img
            class="lazy-image"
            data-src="${image.src}"
            alt="${image.title}"
            loading="lazy"
            data-index="${index}"
          />
          <div class="image-placeholder">${image.placeholder}</div>
          <div class="loading-spinner" style="display: none;"></div>
          <div class="performance-badge" id="badge-${index}" style="display: none;">0ms</div>
        </div>
        <div class="image-info">
          <div class="image-title">${image.title}</div>
          <div class="image-description">${image.description}</div>
        </div>
      </div>
    `).join('');
  }

  async initializeLazyLoading() {
    // Wait for LazyLoader to be available
    if (typeof LazyLoader === 'undefined') {
      await new Promise(resolve => {
        const checkForLazyLoader = () => {
          if (typeof LazyLoader !== 'undefined') {
            resolve();
          } else {
            setTimeout(checkForLazyLoader, 100);
          }
        };
        checkForLazyLoader();
      });
    }

    const images = this.shadowRoot.querySelectorAll('.lazy-image');
    const loader = new LazyLoader();

    images.forEach((img, index) => {
      const startTime = performance.now();
      const spinner = img.parentElement.querySelector('.loading-spinner');
      const badge = img.parentElement.querySelector('.performance-badge');
      const placeholder = img.parentElement.querySelector('.image-placeholder');

      // Show loading state
      img.addEventListener('loadstart', () => {
        spinner.style.display = 'block';
        placeholder.style.display = 'none';
      });

      // Handle successful load
      img.addEventListener('load', () => {
        const loadTime = performance.now() - startTime;
        img.classList.add('loaded');
        spinner.style.display = 'none';
        badge.textContent = `${Math.round(loadTime)}ms`;
        badge.style.display = 'block';

        // Log to performance console
        if (typeof window.logToConsole === 'function') {
          window.logToConsole(`Image ${index + 1} loaded in ${Math.round(loadTime)}ms`, 'success');
        }

        // Track with performance monitor
        if (window.PerformanceMonitor) {
          window.PerformanceMonitor.trackEvent('image-load', {
            index,
            loadTime,
            src: img.src
          });
        }
      });

      // Handle load error
      img.addEventListener('error', () => {
        img.classList.add('error');
        spinner.style.display = 'none';
        placeholder.style.display = 'block';
        placeholder.textContent = '❌ Failed to load';

        if (typeof window.logToConsole === 'function') {
          window.logToConsole(`Image ${index + 1} failed to load`, 'error');
        }
      });

      // Initialize lazy loading for this image
      loader.observe(img);
    });

    if (typeof window.logToConsole === 'function') {
      window.logToConsole(`Initialized lazy loading for ${images.length} images`, 'info');
    }
  }
}

// Register the custom element
customElements.define('lazy-image-demo', LazyImageDemo);

export default LazyImageDemo;
