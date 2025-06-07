// Form Validation Utilities
/**
 * Comprehensive form validation with accessibility and security features
 * Features:
 * - Real-time field validation with customizable rules
 * - Accessibility-compliant error display
 * - CSRF protection integration
 * - Input sanitization using DOMPurify fallback
 */

export class FormValidator {
  static #errorMessages = {
    required: 'This field is required',
    minLength: 'Must be at least {min} characters',
    maxLength: 'Must not exceed {max} characters',
    min: 'Must be at least {min}',
    max: 'Must not exceed {max}',
    email: 'Please enter a valid email address',
    url: 'Please enter a valid URL',
    numeric: 'Please enter a valid number',
    integer: 'Please enter a whole number',
    biasScore: 'Bias score must be between -1 and 1',
    feedback: 'Feedback must be under 1000 characters'
  };

  static validateField(value, rules = {}, fieldName = 'Field') {
    const errors = [];
    const str = String(value ?? '').trim();

    // Required check
    if (rules.required && !str) {
      errors.push(this.#formatMessage('required', rules, fieldName));
    }
    if (!str && !rules.required) return { isValid: true, errors: [] };

    // Length checks
    if (rules.minLength && str.length < rules.minLength) {
      errors.push(this.#formatMessage('minLength', rules, fieldName));
    }
    if (rules.maxLength && str.length > rules.maxLength) {
      errors.push(this.#formatMessage('maxLength', rules, fieldName));
    }

    // Numeric checks
    if (rules.numeric || rules.min != null || rules.max != null) {
      const num = parseFloat(str);
      if (isNaN(num)) {
        errors.push(this.#formatMessage('numeric', rules, fieldName));
      } else {
        if (rules.min != null && num < rules.min) errors.push(this.#formatMessage('min', rules, fieldName));
        if (rules.max != null && num > rules.max) errors.push(this.#formatMessage('max', rules, fieldName));
        if (rules.integer && !Number.isInteger(num)) errors.push(this.#formatMessage('integer', rules, fieldName));
      }
    }

    // Email pattern
    if (rules.email) {
      const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!re.test(str)) errors.push(this.#formatMessage('email', rules, fieldName));
    }

    // URL validity
    if (rules.url) {
      try { new URL(str); } catch { errors.push(this.#formatMessage('url', rules, fieldName)); }
    }

    // Custom pattern
    if (rules.pattern) {
      const re = typeof rules.pattern === 'string' ? new RegExp(rules.pattern) : rules.pattern;
      if (!re.test(str)) errors.push(this.#formatMessage('pattern', rules, fieldName));
    }

    // Custom function
    if (rules.custom) {
      const res = rules.custom(value, str);
      if (res !== true) errors.push(typeof res === 'string' ? res : `Invalid ${fieldName.toLowerCase()}`);
    }

    return { isValid: errors.length === 0, errors };
  }

  static validateForm(formData, schema) {
    const results = {};
    let allErrors = [];
    for (const [field, rules] of Object.entries(schema)) {
      const value = formData[field];
      const res = this.validateField(value, rules, field);
      results[field] = res;
      allErrors = allErrors.concat(res.errors);
    }
    return { isValid: allErrors.length === 0, errors: allErrors, fieldResults: results };
  }

  static sanitizeInput(input) {
    const str = String(input ?? '');
    if (typeof DOMPurify !== 'undefined') {
      return DOMPurify.sanitize(str, { ALLOWED_TAGS: [], ALLOWED_ATTR: [] });
    }
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#x27;');
  }

  static getCsrfToken() {
    const meta = document.querySelector('meta[name="csrf-token"]');
    return meta ? meta.getAttribute('content') : null;
  }

  static #formatMessage(key, rules, field) {
    let msg = this.#errorMessages[key] || 'Invalid value';
    msg = msg.replace(/\{min\}/g, rules.min ?? rules.minLength ?? '')
             .replace(/\{max\}/g, rules.max ?? rules.maxLength ?? '')
             .replace(/\{field\}/g, field);
    return msg;
  }

  static getSchemas() {
    return {
      biasScoring: { score: { required: true, numeric: true, min: -1, max: 1 }, notes: { maxLength: 1000 } },
      feedback: { feedback: { required: true, minLength: 1, maxLength: 1000 } },
      feedManagement: { url: { required: true, url: true }, name: { required: true } }
    };
  }
}

export default FormValidator;
