/**
 * Module de notifications rÃ©utilisable
 * Ã‰limine la duplication de code des notifications
 */

export class NotificationManager {
  constructor() {
    this.container = null;
    this.init();
  }

  /**
   * Initialise le conteneur de notifications
   */
  init() {
    // CrÃ©er le conteneur s'il n'existe pas
    if (!document.getElementById('notification-container')) {
      this.container = document.createElement('div');
      this.container.id = 'notification-container';
      this.container.className = 'notification-container';
      document.body.appendChild(this.container);
    } else {
      this.container = document.getElementById('notification-container');
    }
  }

  /**
   * Affiche une notification
   * @param {string} message - Message Ã  afficher
   * @param {string} type - Type: 'success', 'error', 'warning', 'info'
   * @param {number} duration - DurÃ©e en ms (0 = permanent)
   */
  show(message, type = 'info', duration = 3000) {
    const notification = document.createElement('div');
    notification.className = `notification notification-${type} notification-enter`;

    const icon = this.getIcon(type);
    notification.innerHTML = `
      <span class="notification-icon">${icon}</span>
      <span class="notification-message">${this.escapeHtml(message)}</span>
      <button class="notification-close" aria-label="Fermer">Ã—</button>
    `;

    // Ajouter au conteneur
    this.container.appendChild(notification);

    // Animation d'entrÃ©e
    requestAnimationFrame(() => {
      notification.classList.add('notification-show');
    });

    // Bouton fermer
    const closeBtn = notification.querySelector('.notification-close');
    closeBtn.addEventListener('click', () => this.hide(notification));

    // Auto-fermeture
    if (duration > 0) {
      setTimeout(() => this.hide(notification), duration);
    }

    return notification;
  }

  /**
   * Masque une notification
   * @param {HTMLElement} notification
   */
  hide(notification) {
    notification.classList.remove('notification-show');
    notification.classList.add('notification-exit');

    setTimeout(() => {
      if (notification.parentNode) {
        notification.parentNode.removeChild(notification);
      }
    }, 300);
  }

  /**
   * Affiche une notification de succÃ¨s
   */
  success(message, duration = 3000) {
    return this.show(message, 'success', duration);
  }

  /**
   * Affiche une notification d'erreur
   */
  error(message, duration = 5000) {
    return this.show(message, 'error', duration);
  }

  /**
   * Affiche une notification d'avertissement
   */
  warning(message, duration = 4000) {
    return this.show(message, 'warning', duration);
  }

  /**
   * Affiche une notification d'information
   */
  info(message, duration = 3000) {
    return this.show(message, 'info', duration);
  }

  /**
   * Retourne l'icÃ´ne selon le type
   */
  getIcon(type) {
    const icons = {
      success: 'âœ“',
      error: 'âœ•',
      warning: 'âš ',
      info: 'â„¹'
    };
    return icons[type] || icons.info;
  }

  /**
   * Ã‰chappe le HTML pour prÃ©venir XSS
   */
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Efface toutes les notifications
   */
  clearAll() {
    const notifications = this.container.querySelectorAll('.notification');
    notifications.forEach(n => this.hide(n));
  }
}

// Instance singleton
export const notifications = new NotificationManager();

// Export des mÃ©thodes raccourcies
export const showSuccess = (msg, duration) => notifications.success(msg, duration);
export const showError = (msg, duration) => notifications.error(msg, duration);
export const showWarning = (msg, duration) => notifications.warning(msg, duration);
export const showInfo = (msg, duration) => notifications.info(msg, duration);
