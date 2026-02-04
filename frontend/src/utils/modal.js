/**
 * Module de modales rÃ©utilisable
 * Ã‰limine la duplication de code des modales
 */

export class ModalManager {
  constructor() {
    this.modals = new Map();
  }

  /**
   * CrÃ©e et affiche une modale
   * @param {Object} options - Options de la modale
   * @returns {HTMLElement} - L'Ã©lÃ©ment modal
   */
  create(options = {}) {
    const {
      id = `modal-${Date.now()}`,
      title = 'Modal',
      content = '',
      size = 'medium', // small, medium, large
      closeOnOverlay = true,
      buttons = []
    } = options;

    // CrÃ©er la structure HTML
    const modal = document.createElement('div');
    modal.id = id;
    modal.className = `modal modal-${size}`;
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-labelledby', `${id}-title`);

    modal.innerHTML = `
      <div class="modal-overlay"></div>
      <div class="modal-container">
        <div class="modal-header">
          <h2 id="${id}-title" class="modal-title">${this.escapeHtml(title)}</h2>
          <button class="modal-close" aria-label="Fermer">Ã—</button>
        </div>
        <div class="modal-body">
          ${typeof content === 'string' ? content : ''}
        </div>
        ${buttons.length > 0 ? `
          <div class="modal-footer">
            ${this.renderButtons(buttons)}
          </div>
        ` : ''}
      </div>
    `;

    // Si le contenu est un Ã©lÃ©ment DOM
    if (content instanceof HTMLElement) {
      modal.querySelector('.modal-body').appendChild(content);
    }

    // Ajouter au DOM
    document.body.appendChild(modal);

    // Ã‰vÃ©nements
    this.setupEvents(modal, closeOnOverlay, buttons);

    // Stocker
    this.modals.set(id, modal);

    // Afficher avec animation
    requestAnimationFrame(() => {
      modal.classList.add('modal-show');
    });

    // Focus sur le premier bouton ou input
    this.focusFirstElement(modal);

    return modal;
  }

  /**
   * Configure les Ã©vÃ©nements de la modale
   */
  setupEvents(modal, closeOnOverlay, buttons) {
    const closeBtn = modal.querySelector('.modal-close');
    const overlay = modal.querySelector('.modal-overlay');

    // Fermeture via bouton X
    closeBtn.addEventListener('click', () => this.close(modal.id));

    // Fermeture via overlay
    if (closeOnOverlay) {
      overlay.addEventListener('click', () => this.close(modal.id));
    }

    // Fermeture via Escape
    const escapeHandler = (e) => {
      if (e.key === 'Escape') {
        this.close(modal.id);
        document.removeEventListener('keydown', escapeHandler);
      }
    };
    document.addEventListener('keydown', escapeHandler);

    // Boutons d'action
    buttons.forEach((btn, index) => {
      const btnElement = modal.querySelector(`[data-button-index="${index}"]`);
      if (btnElement && btn.onClick) {
        btnElement.addEventListener('click', (e) => {
          btn.onClick(e, modal);
          if (btn.closeOnClick !== false) {
            this.close(modal.id);
          }
        });
      }
    });
  }

  /**
   * GÃ©nÃ¨re le HTML des boutons
   */
  renderButtons(buttons) {
    return buttons.map((btn, index) => {
      const className = btn.className || 'btn-secondary';
      const type = btn.type || 'button';
      return `
        <button
          type="${type}"
          class="btn ${className}"
          data-button-index="${index}"
        >
          ${this.escapeHtml(btn.text)}
        </button>
      `;
    }).join('');
  }

  /**
   * Ferme une modale
   */
  close(id) {
    const modal = this.modals.get(id);
    if (!modal) return;

    modal.classList.remove('modal-show');

    setTimeout(() => {
      if (modal.parentNode) {
        modal.parentNode.removeChild(modal);
      }
      this.modals.delete(id);
    }, 300);
  }

  /**
   * Ferme toutes les modales
   */
  closeAll() {
    this.modals.forEach((modal, id) => this.close(id));
  }

  /**
   * Met le focus sur le premier Ã©lÃ©ment focusable
   */
  focusFirstElement(modal) {
    setTimeout(() => {
      const focusable = modal.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      if (focusable.length > 0) {
        focusable[0].focus();
      }
    }, 100);
  }

  /**
   * Ã‰chappe le HTML
   */
  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Modale de confirmation
   */
  confirm(options = {}) {
    return new Promise((resolve) => {
      const modal = this.create({
        title: options.title || 'Confirmation',
        content: options.message || 'ÃŠtes-vous sÃ»r ?',
        size: options.size || 'small',
        buttons: [
          {
            text: options.cancelText || 'Annuler',
            className: 'btn-secondary',
            onClick: () => resolve(false)
          },
          {
            text: options.confirmText || 'Confirmer',
            className: options.dangerous ? 'btn-danger' : 'btn-primary',
            onClick: () => resolve(true)
          }
        ]
      });
    });
  }

  /**
   * Modale d'alerte
   */
  alert(options = {}) {
    return new Promise((resolve) => {
      const modal = this.create({
        title: options.title || 'Information',
        content: options.message || '',
        size: options.size || 'small',
        buttons: [
          {
            text: options.buttonText || 'OK',
            className: 'btn-primary',
            onClick: () => resolve(true)
          }
        ]
      });
    });
  }

  /**
   * Modale de prompt
   */
  prompt(options = {}) {
    return new Promise((resolve) => {
      const input = document.createElement('input');
      input.type = options.inputType || 'text';
      input.className = 'form-control';
      input.placeholder = options.placeholder || '';
      input.value = options.defaultValue || '';

      const container = document.createElement('div');
      if (options.message) {
        const p = document.createElement('p');
        p.textContent = options.message;
        container.appendChild(p);
      }
      container.appendChild(input);

      const modal = this.create({
        title: options.title || 'Saisie',
        content: container,
        size: options.size || 'small',
        buttons: [
          {
            text: options.cancelText || 'Annuler',
            className: 'btn-secondary',
            onClick: () => resolve(null)
          },
          {
            text: options.confirmText || 'OK',
            className: 'btn-primary',
            onClick: () => resolve(input.value)
          }
        ]
      });

      // Focus sur l'input
      setTimeout(() => input.focus(), 100);
    });
  }
}

// Instance singleton
export const modal = new ModalManager();

// Exports raccourcis
export const showModal = (options) => modal.create(options);
export const confirm = (options) => modal.confirm(options);
export const alert = (options) => modal.alert(options);
export const prompt = (options) => modal.prompt(options);
