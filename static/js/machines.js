// Gestion des machines (ajout/suppression/filtrage) - V2 Dynamic Updates

let machineToDelete = null;
let searchTimeout = null;

// === Filtrage (Recherche) avec debounce ===

function filterMachines() {
    if (searchTimeout) clearTimeout(searchTimeout);

    searchTimeout = setTimeout(() => {
        const input = document.getElementById('machine-search');
        const filter = input.value.toLowerCase();
        const wrappers = document.querySelectorAll('.machine-card-wrapper');
        const sections = document.querySelectorAll('.group-section');
        const emptyNoResults = document.getElementById('empty-no-results');

        let visibleCount = 0;

        wrappers.forEach(wrapper => {
            const name = wrapper.getAttribute('data-name')?.toLowerCase() || '';
            const ip = wrapper.getAttribute('data-ip')?.toLowerCase() || '';

            if (name.includes(filter) || ip.includes(filter)) {
                wrapper.style.display = "";
                visibleCount++;
            } else {
                wrapper.style.display = "none";
            }
        });

        // Masquer les sections vides
        sections.forEach(section => {
            const visibleCards = section.querySelectorAll('.machine-card-wrapper:not([style*="display: none"])');
            if (visibleCards.length === 0) {
                section.style.display = "none";
            } else {
                section.style.display = "";
            }
        });

        // Afficher/masquer l'état vide selon les résultats
        if (emptyNoResults) {
            if (visibleCount === 0 && filter.length > 0) {
                emptyNoResults.style.display = "flex";
            } else {
                emptyNoResults.style.display = "none";
            }
        }
    }, 200);
}

// === Effacer la recherche ===

function clearSearch() {
    const input = document.getElementById('machine-search');
    if (input) {
        input.value = '';
        filterMachines();
        input.focus();
    }
}

// === Utilitaire: État de chargement bouton ===

function setButtonLoading(button, loading) {
    if (loading) {
        button.classList.add('loading');
        button.disabled = true;
    } else {
        button.classList.remove('loading');
        button.disabled = false;
    }
}

// === Fonctions de mise à jour dynamique du DOM ===

/**
 * Supprime une carte machine du DOM avec animation
 */
function removeMachineCard(machineId) {
    const wrapper = document.querySelector(`.machine-card-wrapper[data-id="${machineId}"]`);
    if (!wrapper) return;

    // Animation de sortie
    wrapper.style.transition = 'all 0.3s ease-out';
    wrapper.style.transform = 'scale(0.9)';
    wrapper.style.opacity = '0';

    setTimeout(() => {
        const section = wrapper.closest('.group-section');
        wrapper.remove();

        // Vérifier si la section est maintenant vide
        if (section) {
            const remainingCards = section.querySelectorAll('.machine-card-wrapper');
            if (remainingCards.length === 0) {
                section.style.transition = 'all 0.3s ease-out';
                section.style.opacity = '0';
                setTimeout(() => section.remove(), 300);
            }
        }

        // Mettre à jour le compteur de machines
        updateMachineCount(-1);

        // Vérifier s'il reste des machines
        checkEmptyState();
    }, 300);
}

/**
 * Met à jour le compteur de machines dans la navbar
 */
function updateMachineCount(delta) {
    const countBadge = document.querySelector('.machine-count, .nav-badge');
    if (countBadge) {
        const currentCount = parseInt(countBadge.textContent) || 0;
        const newCount = Math.max(0, currentCount + delta);
        countBadge.textContent = newCount;

        // Animation du badge
        countBadge.style.transform = 'scale(1.2)';
        setTimeout(() => {
            countBadge.style.transform = 'scale(1)';
        }, 200);
    }
}

/**
 * Vérifie s'il faut afficher l'état vide (aucune machine)
 */
function checkEmptyState() {
    const wrappers = document.querySelectorAll('.machine-card-wrapper');
    const emptyState = document.getElementById('empty-state');
    const machinesGrid = document.querySelector('.machines-grid, .group-section');

    if (wrappers.length === 0 && emptyState) {
        emptyState.style.display = 'flex';
    }
}

/**
 * Rafraîchit les données des machines depuis le serveur
 * Utilisé après ajout/modification pour obtenir le HTML mis à jour
 */
async function refreshMachinesData() {
    try {
        // Récupérer la page actuelle et extraire le contenu des machines
        const response = await fetch(window.location.href);
        if (!response.ok) throw new Error('Erreur de chargement');

        const html = await response.text();
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');

        // Trouver le conteneur des machines dans la nouvelle page
        const newContent = doc.querySelector('.content-wrapper');
        const currentContent = document.querySelector('.content-wrapper');

        if (newContent && currentContent) {
            // Animation de transition
            currentContent.style.opacity = '0.5';

            setTimeout(() => {
                // Remplacer le contenu
                currentContent.innerHTML = newContent.innerHTML;
                currentContent.style.opacity = '1';

                // Ré-appliquer le filtre de recherche si actif
                const searchInput = document.getElementById('machine-search');
                if (searchInput && searchInput.value) {
                    filterMachines();
                }
            }, 150);
        }
    } catch (error) {
        console.error('Erreur lors du rafraîchissement:', error);
        // En cas d'erreur, recharger la page
        location.reload();
    }
}

// === Modal Ajouter ===

function openAddModal() {
    const modal = document.getElementById('add-modal');
    modal.classList.add('open');
    document.getElementById('add-machine-form').reset();
    switchAuthTab('key');
}

function closeAddModal() {
    const modal = document.getElementById('add-modal');
    modal.classList.remove('open');
}

function switchAuthTab(tab) {
    const tabs = document.querySelectorAll('#add-modal .auth-tab');
    tabs.forEach(t => t.classList.remove('active'));

    if (tab === 'key') {
        tabs[0].classList.add('active');
        document.getElementById('auth-key').style.display = 'block';
        document.getElementById('auth-password').style.display = 'none';
        document.getElementById('machine-password').value = '';
    } else {
        tabs[1].classList.add('active');
        document.getElementById('auth-key').style.display = 'none';
        document.getElementById('auth-password').style.display = 'block';
        document.getElementById('machine-keypath').value = '';
    }
}

async function addMachine(event) {
    event.preventDefault();

    const form = document.getElementById('add-machine-form');
    const submitBtn = form.querySelector('button[type="submit"]');

    if (!form) {
        showNotification('Erreur: Formulaire introuvable', 'error');
        return;
    }

    const data = {
        id: form.querySelector('#machine-id').value.trim(),
        name: form.querySelector('#machine-name').value.trim(),
        group: form.querySelector('#machine-group').value.trim(),
        host: form.querySelector('#machine-host').value.trim(),
        port: parseInt(form.querySelector('#machine-port').value) || 22,
        user: form.querySelector('#machine-user').value.trim(),
        key_path: form.querySelector('#machine-keypath').value.trim(),
        password: form.querySelector('#machine-password').value
    };

    // Validation côté client
    if (!data.id || !data.name || !data.host || !data.user) {
        showNotification('Veuillez remplir tous les champs obligatoires', 'error');
        return;
    }

    if (!data.key_path && !data.password) {
        showNotification('Veuillez fournir une clé SSH ou un mot de passe', 'error');
        return;
    }

    setButtonLoading(submitBtn, true);

    try {
        const response = await fetch('/api/machines', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (response.ok) {
            showNotification('Machine ajoutée avec succès', 'success');
            closeAddModal();
            // Rafraîchir les données sans recharger la page
            await refreshMachinesData();
        } else {
            showNotification(result.error || 'Erreur lors de l\'ajout', 'error');
        }
    } catch (error) {
        showNotification('Erreur de connexion au serveur', 'error');
        console.error('Error:', error);
    } finally {
        setButtonLoading(submitBtn, false);
    }
}

// === Modal Supprimer ===

function deleteMachine(id, name, e) {
    if (e) {
        e.preventDefault();
        e.stopPropagation();
    }

    machineToDelete = id;
    document.getElementById('delete-machine-name').textContent = name;
    document.getElementById('delete-modal').classList.add('open');
}

function closeDeleteModal() {
    document.getElementById('delete-modal').classList.remove('open');
    machineToDelete = null;
}

async function confirmDelete() {
    if (!machineToDelete) return;

    const deleteBtn = document.querySelector('#delete-modal .btn-danger');
    const machineId = machineToDelete;
    setButtonLoading(deleteBtn, true);

    try {
        const response = await fetch(`/api/machines/${machineId}`, {
            method: 'DELETE'
        });

        const result = await response.json();

        if (response.ok) {
            showNotification('Machine supprimée avec succès', 'success');
            closeDeleteModal();
            // Supprimer la carte du DOM directement (pas de reload)
            removeMachineCard(machineId);
        } else {
            showNotification(result.error || 'Erreur lors de la suppression', 'error');
        }
    } catch (error) {
        showNotification('Erreur de connexion au serveur', 'error');
        console.error('Error:', error);
    } finally {
        setButtonLoading(deleteBtn, false);
    }
}

// === Modal Modifier ===

function openEditModal(id, name, group, host, port, user, keyPath) {
    document.getElementById('edit-modal').classList.add('open');

    document.getElementById('edit-machine-id').value = id;
    document.getElementById('edit-machine-name').value = name;
    document.getElementById('edit-machine-group').value = group;
    document.getElementById('edit-machine-host').value = host;
    document.getElementById('edit-machine-port').value = port;
    document.getElementById('edit-machine-user').value = user;
    document.getElementById('edit-machine-keypath').value = keyPath;
    document.getElementById('edit-machine-password').value = "";

    switchEditAuthTab(keyPath ? 'key' : 'key');
}

function closeEditModal() {
    document.getElementById('edit-modal').classList.remove('open');
}

function switchEditAuthTab(tab) {
    const tabs = document.querySelectorAll('#edit-modal .auth-tab');

    if (tab === 'key') {
        tabs[0].classList.add('active');
        tabs[1].classList.remove('active');
        document.getElementById('edit-auth-key').style.display = 'block';
        document.getElementById('edit-auth-password').style.display = 'none';
    } else {
        tabs[1].classList.add('active');
        tabs[0].classList.remove('active');
        document.getElementById('edit-auth-key').style.display = 'none';
        document.getElementById('edit-auth-password').style.display = 'block';
    }
}

async function updateMachine(event) {
    event.preventDefault();

    const form = document.getElementById('edit-machine-form');
    const submitBtn = form.querySelector('button[type="submit"]');
    const id = form.querySelector('#edit-machine-id').value;

    const data = {
        name: form.querySelector('#edit-machine-name').value.trim(),
        group: form.querySelector('#edit-machine-group').value.trim(),
        host: form.querySelector('#edit-machine-host').value.trim(),
        port: parseInt(form.querySelector('#edit-machine-port').value) || 22,
        user: form.querySelector('#edit-machine-user').value.trim(),
        key_path: form.querySelector('#edit-machine-keypath').value.trim(),
        password: form.querySelector('#edit-machine-password').value
    };

    setButtonLoading(submitBtn, true);

    try {
        const response = await fetch(`/api/machines/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        const result = await response.json();

        if (response.ok) {
            showNotification('Machine mise à jour avec succès', 'success');
            closeEditModal();
            // Rafraîchir les données sans recharger la page
            await refreshMachinesData();
        } else {
            showNotification(result.error || 'Erreur lors de la mise à jour', 'error');
        }
    } catch (error) {
        showNotification('Erreur de connexion au serveur', 'error');
        console.error('Error:', error);
    } finally {
        setButtonLoading(submitBtn, false);
    }
}

// === Notifications améliorées ===

const notificationIcons = {
    success: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>`,
    error: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>`,
    info: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>`,
    warning: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>`
};

const notificationTitles = {
    success: 'Succès',
    error: 'Erreur',
    info: 'Information',
    warning: 'Attention'
};

function showNotification(message, type = 'info') {
    // Supprimer les notifications existantes
    const existing = document.querySelector('.notification');
    if (existing) existing.remove();

    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.style.position = 'relative';
    notification.innerHTML = `
        <div class="notification-icon">${notificationIcons[type] || notificationIcons.info}</div>
        <div class="notification-content">
            <div class="notification-title">${notificationTitles[type] || 'Notification'}</div>
            <div class="notification-message">${message}</div>
        </div>
        <button class="notification-close" onclick="this.parentElement.remove()" aria-label="Fermer">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
        </button>
        <div class="notification-progress"></div>
    `;

    document.body.appendChild(notification);

    // Auto-hide après 5 secondes
    setTimeout(() => {
        if (notification.parentElement) {
            notification.style.animation = 'slideInRight 0.3s ease-out reverse forwards';
            setTimeout(() => notification.remove(), 300);
        }
    }, 5000);
}

// === Fermer les modals en cliquant à l'extérieur ===

document.addEventListener('click', function (event) {
    if (event.target.classList.contains('modal') && event.target.classList.contains('open')) {
        event.target.classList.remove('open');
    }
});

// === Fermer avec Escape ===

document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
        closeAddModal();
        closeDeleteModal();
        closeEditModal();
    }
});
