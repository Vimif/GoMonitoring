/**
 * Point d'entrÃ©e principal de l'application
 * Initialise les modules communs
 */

// Import des utilities
import { notifications } from './utils/notifications.js';
import { modal } from './utils/modal.js';
import { api } from './utils/api.js';

// Exports globaux pour compatibilitÃ©
window.notifications = notifications;
window.modal = modal;
window.api = api;

// Initialisation au chargement du DOM
document.addEventListener('DOMContentLoaded', () => {
  console.log('Go Monitoring - Frontend initialisÃ©');

  // Configuration globale des erreurs
  window.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled promise rejection:', event.reason);
    notifications.error('Une erreur inattendue est survenue');
  });

  // Initialiser le thÃ¨me
  initTheme();

  // Initialiser la navigation
  initNavigation();
});

/**
 * Initialise le systÃ¨me de thÃ¨me
 */
function initTheme() {
  const savedTheme = localStorage.getItem('theme') || 'dark';
  document.documentElement.setAttribute('data-theme', savedTheme);

  // Ã‰couteur pour le bouton de thÃ¨me
  const themeToggle = document.getElementById('theme-toggle');
  if (themeToggle) {
    themeToggle.addEventListener('click', () => {
      const currentTheme = document.documentElement.getAttribute('data-theme');
      const newTheme = currentTheme === 'dark' ? 'light' : 'dark';

      document.documentElement.setAttribute('data-theme', newTheme);
      localStorage.setItem('theme', newTheme);
    });
  }
}

/**
 * Initialise la navigation
 */
function initNavigation() {
  // Marquer le lien actif dans la sidebar
  const currentPath = window.location.pathname;
  const navLinks = document.querySelectorAll('.sidebar-nav a');

  navLinks.forEach(link => {
    if (link.getAttribute('href') === currentPath) {
      link.classList.add('active');
    }
  });

  // Collapse sidebar sur mobile
  const sidebarToggle = document.getElementById('sidebar-toggle');
  const sidebar = document.querySelector('.sidebar');

  if (sidebarToggle && sidebar) {
    sidebarToggle.addEventListener('click', () => {
      sidebar.classList.toggle('collapsed');
    });
  }
}

// Export pour utilisation dans d'autres modules
export { notifications, modal, api };
