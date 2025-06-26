// Critical HTMX loader - minimal implementation for deferred loading
(function() {
  // Basic HTMX functionality placeholder until full library loads
  window.htmx = window.htmx || {
    _placeholder: true,
    process: function() {},
    on: function() {},
    trigger: function() {}
  };

  // Defer loading of full HTMX library
  function loadHTMX() {
    if (window.htmx && !window.htmx._placeholder) return;
    
    const script = document.createElement('script');
    script.src = 'https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js';
    script.onload = function() {
      if (window.htmx && window.htmx.process) {
        window.htmx.process(document.body);
      }
    };
    document.head.appendChild(script);
  }

  // Load HTMX after critical content is painted
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      setTimeout(loadHTMX, 100);
    });
  } else {
    setTimeout(loadHTMX, 100);
  }
})();