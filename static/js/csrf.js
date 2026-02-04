/**
 * CSRF Protection Helper
 * Ajoute automatiquement le token CSRF Ã  toutes les requÃªtes HTTP
 */

// RÃ©cupÃ©rer le token CSRF depuis la meta tag
function getCSRFToken() {
    const meta = document.querySelector('meta[name="csrf-token"]');
    return meta ? meta.getAttribute('content') : '';
}

// Wrapper autour de fetch() pour inclure automatiquement le token CSRF
const originalFetch = window.fetch;
window.fetch = function (url, options = {}) {
    // Pour les requÃªtes qui modifient l'Ã©tat (POST, PUT, DELETE, PATCH)
    const method = (options.method || 'GET').toUpperCase();
    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
        // Ajouter le header X-CSRF-Token
        options.headers = options.headers || {};
        if (typeof options.headers === 'object') {
            // Si c'est un objet Headers
            if (options.headers instanceof Headers) {
                options.headers.set('X-CSRF-Token', getCSRFToken());
            } else {
                // Si c'est un objet plain
                options.headers['X-CSRF-Token'] = getCSRFToken();
            }
        }
    }

    return originalFetch.call(this, url, options);
};

// Ajouter le token CSRF aux formulaires qui n'utilisent pas fetch
document.addEventListener('DOMContentLoaded', () => {
    // Pour les formulaires HTML classiques (method POST)
    document.querySelectorAll('form[method="post"], form[method="POST"]').forEach(form => {
        // VÃ©rifier si le formulaire n'a pas dÃ©jÃ  un input csrf_token
        if (!form.querySelector('input[name="csrf_token"]')) {
            const input = document.createElement('input');
            input.type = 'hidden';
            input.name = 'csrf_token';
            input.value = getCSRFToken();
            form.appendChild(input);
        }
    });
});

// Log pour debug (Ã  retirer en production)
if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
    console.log('[CSRF] Protection activÃ©e');
}
