// Gestion des utilisateurs

document.addEventListener('DOMContentLoaded', () => {
    console.log("Users Module Loaded - V4 Dialog System");
    loadUsers();
});

async function loadUsers() {
    const tableBody = document.querySelector('#users-table tbody');
    if (!tableBody) return;

    try {
        const response = await fetch('/api/users');
        if (!response.ok) throw new Error('Erreur chargement utilisateurs');
        const users = await response.json();

        tableBody.innerHTML = users.map(user => {
            let statusBadge = '';
            let lockInfo = '';

            if (user.is_locked) {
                statusBadge = `<span class="status-badge status-critical">Verrouillé</span>`;
            } else if (user.is_active) {
                statusBadge = `<span class="status-badge status-online">Actif</span>`;
            } else {
                statusBadge = `<span class="status-badge status-offline">Inactif</span>`;
            }

            return `
            <tr>
                <td>
                    <div style="display: flex; align-items: center; gap: 0.75rem;">
                        <span style="font-weight: 600; color: var(--text-color);">${user.username}</span>
                        ${user.username === 'admin' ? '<span class="status-badge status-warning system-badge">SYSTEM</span>' : ''}
                    </div>
                </td>
                <td>
                    <div class="badge-role"
                         ${user.username !== 'admin' ? `onclick="openRoleModal('${user.username}', '${user.role}')"` : ''}
                         style="cursor: ${user.username !== 'admin' ? 'pointer' : 'default'}; display: inline-flex; align-items: center; gap: 0.5rem; color: var(--text-muted); font-weight: 500;">
                        <span>${user.role}</span>
                        ${user.username !== 'admin' ? '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path></svg>' : ''}
                    </div>
                </td>
                <td>${statusBadge}</td>
                <td class="actions-cell">
                        ${user.is_locked ? `
                        <button class="btn-action btn-restart" title="Déverrouiller" onclick="unlockUser('${user.username}')">
                            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 9.9-1"></path></svg>
                        </button>` : ''}

                        ${user.username !== 'admin' ? `
                        <button class="btn-action ${user.is_active ? 'btn-stop' : 'btn-start'}" title="${user.is_active ? 'Désactiver' : 'Activer'}" onclick="toggleUserStatus('${user.username}', ${!user.is_active})">
                            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                                <path d="M18.36 6.64a9 9 0 1 1-12.73 0"></path>
                                <line x1="12" y1="2" x2="12" y2="12"></line>
                            </svg>
                        </button>` : ''}

                        <button class="btn-action btn-edit" title="Changer mot de passe" onclick="openPasswordModal('${user.username}')">
                            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-1.95-2.04l.11-.11.55-.55a2 2 0 0 1 2.83 2.83l-.7.7m-4.25-4.24l1.42 1.41"></path></svg>
                        </button>

                        ${user.username !== 'admin' ? `
                        <button class="btn-action btn-stop" title="Supprimer" onclick="deleteUser('${user.username}')">
                            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                        </button>` : ''}
                </td>
            </tr>
            `;
        }).join('');
    } catch (error) {
        console.error(error);
        tableBody.innerHTML = `<tr><td colspan="4" class="text-error">Erreur de chargement: ${error.message}</td></tr>`;
    }
}

// Add User Modal
function openAddUserModal() {
    document.getElementById('add-user-modal').classList.add('open');
    document.getElementById('add-user-form').reset();
}

function closeAddUserModal() {
    document.getElementById('add-user-modal').classList.remove('open');
}

async function submitAddUser(e) {
    e.preventDefault();
    const formData = new FormData(e.target);
    const data = Object.fromEntries(formData.entries());

    try {
        const response = await fetch('/api/users', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (!response.ok) {
            const err = await response.text();
            throw new Error(err);
        }

        closeAddUserModal();
        loadUsers();
    } catch (error) {
        await dialog.alert(error.message, {
            title: 'Erreur',
            type: 'error'
        });
    }
}

// Role Modal
function openRoleModal(username, currentRole) {
    document.getElementById('role-modal').classList.add('open');
    document.getElementById('role-username').value = username;
    document.getElementById('role-username-display').textContent = username;
    document.getElementById('edit-role').value = currentRole;
    document.getElementById('role-form').reset();
    // Re-set value after reset
    document.getElementById('edit-role').value = currentRole;
}

function closeRoleModal() {
    document.getElementById('role-modal').classList.remove('open');
}

async function submitRoleChange(e) {
    e.preventDefault();
    const username = document.getElementById('role-username').value;
    const role = document.getElementById('edit-role').value;

    try {
        const response = await fetch(`/api/users/${username}/role`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ role })
        });

        if (!response.ok) {
            const err = await response.text();
            throw new Error(err);
        }

        closeRoleModal();
        loadUsers();
        // Feedback
        const badge = document.querySelector(`.badge-role[onclick*="${username}"]`);
        if (badge) {
            badge.style.animation = "pulse-green 0.5s";
        }
    } catch (error) {
        await dialog.alert(error.message, {
            title: 'Erreur',
            type: 'error'
        });
    }
}

// Password Modal
function openPasswordModal(username) {
    document.getElementById('password-modal').classList.add('open');
    document.getElementById('pwd-username').value = username;
    document.getElementById('pwd-username-display').textContent = username;
    document.getElementById('password-form').reset();
}

function closePasswordModal() {
    document.getElementById('password-modal').classList.remove('open');
}

async function submitPasswordChange(e) {
    e.preventDefault();
    const username = document.getElementById('pwd-username').value;
    const formData = new FormData(e.target);
    const password = formData.get('password');

    try {
        const response = await fetch(`/api/users/${username}/password`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ password })
        });

        if (!response.ok) {
            const err = await response.text();
            throw new Error(err);
        }

        closePasswordModal();
        await dialog.alert('Mot de passe mis à jour avec succès', {
            title: 'Succès',
            type: 'success'
        });
    } catch (error) {
        await dialog.alert(error.message, {
            title: 'Erreur',
            type: 'error'
        });
    }
}

// Toggle Status
async function toggleUserStatus(username, active) {
    const action = active ? 'activer' : 'désactiver';

    const confirmed = await dialog.confirm(
        `Voulez-vous vraiment ${action} le compte ${username} ?`,
        {
            title: active ? 'Activer le compte' : 'Désactiver le compte',
            type: 'question',
            confirmText: active ? 'Activer' : 'Désactiver',
            cancelText: 'Annuler',
            danger: !active
        }
    );

    if (!confirmed) return;

    try {
        const response = await fetch(`/api/users/${username}/toggle-status`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ active: active })
        });

        if (!response.ok) throw new Error(await response.text());
        loadUsers();
    } catch (error) {
        await dialog.alert(error.message, {
            title: 'Erreur',
            type: 'error'
        });
    }
}

// Unlock User
async function unlockUser(username) {
    const confirmed = await dialog.confirm(
        `Déverrouiller le compte ${username} ?`,
        {
            title: 'Déverrouiller le compte',
            type: 'question',
            confirmText: 'Déverrouiller',
            cancelText: 'Annuler'
        }
    );

    if (!confirmed) return;

    try {
        const response = await fetch(`/api/users/${username}/unlock`, {
            method: 'POST'
        });

        if (!response.ok) throw new Error(await response.text());
        loadUsers();
        await dialog.alert('Compte déverrouillé avec succès', {
            title: 'Succès',
            type: 'success'
        });
    } catch (error) {
        await dialog.alert(error.message, {
            title: 'Erreur',
            type: 'error'
        });
    }
}

// Delete User
async function deleteUser(username) {
    const confirmed = await dialog.confirm(
        `Êtes-vous sûr de vouloir supprimer l'utilisateur ${username} ?\n\nCette action est irréversible.`,
        {
            title: 'Supprimer l\'utilisateur',
            type: 'warning',
            confirmText: 'Supprimer',
            cancelText: 'Annuler',
            danger: true
        }
    );

    if (!confirmed) return;

    try {
        const response = await fetch(`/api/users/${username}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const err = await response.text();
            throw new Error(err);
        }

        loadUsers();
    } catch (error) {
        await dialog.alert(error.message, {
            title: 'Erreur',
            type: 'error'
        });
    }
}

// Close modals on outside click
window.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) {
        e.target.classList.remove('open');
    }
});
