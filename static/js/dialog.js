/**
 * Professional Dialog System
 * Remplace les alert() et confirm() natifs par des modals stylÃ©s
 */

class DialogManager {
    constructor() {
        this.container = null;
        this.activeDialog = null;
        this.focusableElements = [];
        this.previouslyFocused = null;
        this.init();
    }

    init() {
        // CrÃ©er le conteneur de dialogue
        this.container = document.createElement('div');
        this.container.id = 'dialog-container';
        this.container.innerHTML = `
            <div class="dialog-backdrop" data-dialog-close></div>
            <div class="dialog-box" role="dialog" aria-modal="true" aria-labelledby="dialog-title">
                <div class="dialog-icon"></div>
                <h3 class="dialog-title" id="dialog-title"></h3>
                <p class="dialog-message" id="dialog-message"></p>
                <div class="dialog-actions"></div>
            </div>
        `;
        document.body.appendChild(this.container);

        // Ã‰vÃ©nements
        this.container.addEventListener('click', (e) => {
            if (e.target.hasAttribute('data-dialog-close')) {
                this.handleCancel();
            }
        });

        document.addEventListener('keydown', (e) => {
            if (!this.activeDialog) return;

            if (e.key === 'Escape') {
                e.preventDefault();
                this.handleCancel();
            }

            if (e.key === 'Tab') {
                this.handleTabKey(e);
            }
        });
    }

    // IcÃ´nes SVG pour chaque type
    getIcon(type) {
        const icons = {
            info: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="12" y1="16" x2="12" y2="12"/>
                <line x1="12" y1="8" x2="12.01" y2="8"/>
            </svg>`,
            success: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <path d="M9 12l2 2 4-4"/>
            </svg>`,
            warning: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                <line x1="12" y1="9" x2="12" y2="13"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>`,
            error: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="15" y1="9" x2="9" y2="15"/>
                <line x1="9" y1="9" x2="15" y2="15"/>
            </svg>`,
            question: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>`
        };
        return icons[type] || icons.info;
    }

    /**
     * Affiche une alerte (remplace alert())
     * @param {string} message - Le message Ã  afficher
     * @param {Object} options - Options de configuration
     * @returns {Promise<void>}
     */
    alert(message, options = {}) {
        return new Promise((resolve) => {
            const config = {
                type: options.type || 'info',
                title: options.title || 'Information',
                message: message,
                confirmText: options.confirmText || 'OK',
                onConfirm: resolve,
                onCancel: resolve
            };
            this.show(config, false);
        });
    }

    /**
     * Affiche une confirmation (remplace confirm())
     * @param {string} message - Le message Ã  afficher
     * @param {Object} options - Options de configuration
     * @returns {Promise<boolean>}
     */
    confirm(message, options = {}) {
        return new Promise((resolve) => {
            const config = {
                type: options.type || 'question',
                title: options.title || 'Confirmation',
                message: message,
                confirmText: options.confirmText || 'Confirmer',
                cancelText: options.cancelText || 'Annuler',
                danger: options.danger || false,
                onConfirm: () => resolve(true),
                onCancel: () => resolve(false)
            };
            this.show(config, true);
        });
    }

    show(config, isConfirm) {
        this.previouslyFocused = document.activeElement;
        this.activeDialog = config;

        const box = this.container.querySelector('.dialog-box');
        const iconEl = this.container.querySelector('.dialog-icon');
        const titleEl = this.container.querySelector('.dialog-title');
        const messageEl = this.container.querySelector('.dialog-message');
        const actionsEl = this.container.querySelector('.dialog-actions');

        // DÃ©finir le type pour le style
        box.className = `dialog-box dialog-${config.type}`;

        // IcÃ´ne
        iconEl.innerHTML = this.getIcon(config.type);

        // Contenu
        titleEl.textContent = config.title;
        messageEl.textContent = config.message;

        // Boutons
        let buttonsHTML = '';
        if (isConfirm) {
            buttonsHTML = `
                <button type="button" class="btn btn-secondary" data-action="cancel">${config.cancelText}</button>
                <button type="button" class="btn ${config.danger ? 'btn-danger' : 'btn-primary'}" data-action="confirm">${config.confirmText}</button>
            `;
        } else {
            buttonsHTML = `
                <button type="button" class="btn btn-primary" data-action="confirm">${config.confirmText}</button>
            `;
        }
        actionsEl.innerHTML = buttonsHTML;

        // Gestionnaires de boutons
        actionsEl.querySelectorAll('button').forEach(btn => {
            btn.addEventListener('click', () => {
                const action = btn.getAttribute('data-action');
                if (action === 'confirm') {
                    this.handleConfirm();
                } else {
                    this.handleCancel();
                }
            });
        });

        // Afficher avec animation
        this.container.classList.add('open');

        // Focus sur le premier bouton
        requestAnimationFrame(() => {
            this.updateFocusableElements();
            const confirmBtn = actionsEl.querySelector('[data-action="confirm"]');
            if (confirmBtn) confirmBtn.focus();
        });
    }

    hide() {
        this.container.classList.remove('open');
        this.activeDialog = null;

        // Restaurer le focus
        if (this.previouslyFocused) {
            this.previouslyFocused.focus();
        }
    }

    handleConfirm() {
        if (this.activeDialog && this.activeDialog.onConfirm) {
            this.activeDialog.onConfirm();
        }
        this.hide();
    }

    handleCancel() {
        if (this.activeDialog && this.activeDialog.onCancel) {
            this.activeDialog.onCancel();
        }
        this.hide();
    }

    updateFocusableElements() {
        const box = this.container.querySelector('.dialog-box');
        this.focusableElements = box.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
    }

    handleTabKey(e) {
        if (this.focusableElements.length === 0) return;

        const firstElement = this.focusableElements[0];
        const lastElement = this.focusableElements[this.focusableElements.length - 1];

        if (e.shiftKey) {
            if (document.activeElement === firstElement) {
                e.preventDefault();
                lastElement.focus();
            }
        } else {
            if (document.activeElement === lastElement) {
                e.preventDefault();
                firstElement.focus();
            }
        }
    }
}

// Instance globale
window.dialog = new DialogManager();
