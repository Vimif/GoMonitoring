// Dashboard WebSocket avec reconnexion exponentielle
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 10;
const BASE_DELAY = 1000; // 1 seconde
const MAX_DELAY = 30000; // 30 secondes max

document.addEventListener('DOMContentLoaded', () => {
    initWebSocket();
});

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;

    console.log(`Connexion WebSocket: ${wsUrl}`);
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log("WebSocket connecté");
        reconnectAttempts = 0; // Reset on successful connection

        // Supprimer la notification de déconnexion si elle existe
        const disconnectNotif = document.querySelector('.ws-disconnect-notification');
        if (disconnectNotif) disconnectNotif.remove();
    };

    ws.onmessage = (event) => {
        try {
            const machines = JSON.parse(event.data);
            updateDashboardUI(machines);
        } catch (e) {
            console.error("Erreur parsing WS", e);
        }
    };

    ws.onclose = () => {
        console.log("WebSocket déconnecté");

        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
            // Calcul du délai avec backoff exponentiel
            const delay = Math.min(BASE_DELAY * Math.pow(2, reconnectAttempts), MAX_DELAY);
            reconnectAttempts++;

            console.log(`Reconnexion dans ${delay / 1000}s (tentative ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);

            // Afficher un indicateur discret si plusieurs tentatives
            if (reconnectAttempts >= 5) {
                showReconnectingIndicator(delay);
            }

            setTimeout(initWebSocket, delay);
        } else {
            console.error("Nombre maximum de tentatives de reconnexion atteint");
            showDisconnectedNotification();
        }
    };

    ws.onerror = (err) => {
        console.error("Erreur WebSocket", err);
        ws.close();
    };
}

/**
 * Affiche un indicateur de reconnexion discret
 */
function showReconnectingIndicator(delay) {
    // Supprimer l'indicateur existant
    let indicator = document.querySelector('.ws-reconnect-indicator');
    if (indicator) indicator.remove();

    indicator = document.createElement('div');
    indicator.className = 'ws-reconnect-indicator';
    indicator.innerHTML = `
        <span class="reconnect-spinner"></span>
        <span>Reconnexion en cours...</span>
    `;
    indicator.style.cssText = `
        position: fixed;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%);
        background: var(--card-bg, #1e293b);
        color: var(--text-muted, #94a3b8);
        padding: 8px 16px;
        border-radius: 20px;
        font-size: 0.85rem;
        display: flex;
        align-items: center;
        gap: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.2);
        z-index: 9999;
        animation: fadeIn 0.3s ease-out;
    `;

    // Ajouter le style du spinner
    const style = document.createElement('style');
    style.textContent = `
        .reconnect-spinner {
            width: 14px;
            height: 14px;
            border: 2px solid var(--text-light, #64748b);
            border-top-color: var(--primary-color, #3b82f6);
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
    `;
    document.head.appendChild(style);

    document.body.appendChild(indicator);

    // Supprimer après le délai
    setTimeout(() => {
        if (indicator.parentElement) {
            indicator.style.opacity = '0';
            setTimeout(() => indicator.remove(), 300);
        }
    }, delay - 500);
}

/**
 * Affiche une notification de déconnexion permanente
 */
function showDisconnectedNotification() {
    // Supprimer les indicateurs temporaires
    const indicator = document.querySelector('.ws-reconnect-indicator');
    if (indicator) indicator.remove();

    // Créer la notification de déconnexion
    const notification = document.createElement('div');
    notification.className = 'ws-disconnect-notification';
    notification.innerHTML = `
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        <span>Connexion perdue</span>
        <button onclick="location.reload()" style="
            background: var(--primary-color, #3b82f6);
            color: white;
            border: none;
            padding: 4px 12px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.8rem;
        ">Rafraîchir</button>
    `;
    notification.style.cssText = `
        position: fixed;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%);
        background: var(--danger-light, rgba(239, 68, 68, 0.1));
        color: var(--danger-color, #ef4444);
        padding: 10px 20px;
        border-radius: 8px;
        font-size: 0.9rem;
        display: flex;
        align-items: center;
        gap: 10px;
        box-shadow: 0 4px 12px rgba(239, 68, 68, 0.2);
        z-index: 9999;
        border: 1px solid var(--danger-color, #ef4444);
    `;

    document.body.appendChild(notification);
}

function updateDashboardUI(machines) {
    machines.forEach(m => {
        const card = document.getElementById(`card-${m.id}`);
        if (!card) return;

        // Vérifier si le statut a changé pour l'animation
        const previousStatus = card.classList.contains('status-online') ? 'online' :
            card.classList.contains('status-offline') ? 'offline' : 'unknown';
        const statusChanged = previousStatus !== m.status;

        // Mise à jour de la classe de statut
        card.classList.forEach(cls => {
            if (cls.startsWith('status-') && cls !== `status-${m.status}`) {
                card.classList.remove(cls);
            }
        });
        card.classList.add(`status-${m.status}`);

        // Animation de transition si le statut a changé
        if (statusChanged) {
            card.classList.add('fade-in');
            setTimeout(() => card.classList.remove('fade-in'), 300);
        }

        // Mise à jour des métriques si présentes
        if (m.status === 'online') {
            const metricsDiv = card.querySelector('.machine-metrics');
            if (metricsDiv) {
                // CPU
                if (m.cpu) {
                    const cpuBar = metricsDiv.querySelector('.metric-cpu .progress-fill');
                    const cpuText = metricsDiv.querySelector('.metric-cpu .metric-value');
                    if (cpuBar && cpuText) {
                        const cpuVal = m.cpu.usage_percent;
                        cpuBar.style.width = `${cpuVal.toFixed(1)}%`;
                        cpuText.textContent = `${cpuVal.toFixed(1)}%`;

                        cpuBar.className = 'progress-fill';
                        if (cpuVal > 80) cpuBar.classList.add('danger');
                        else if (cpuVal > 60) cpuBar.classList.add('warning');
                    }
                }

                // RAM
                if (m.memory) {
                    const memBar = metricsDiv.querySelector('.metric-mem .progress-fill');
                    const memText = metricsDiv.querySelector('.metric-mem .metric-value');
                    if (memBar && memText) {
                        const memVal = m.memory.used_percent;
                        memBar.style.width = `${memVal.toFixed(1)}%`;
                        memText.textContent = `${memVal.toFixed(1)}%`;

                        memBar.className = 'progress-fill';
                        if (memVal > 80) memBar.classList.add('danger');
                        else if (memVal > 60) memBar.classList.add('warning');
                    }
                }
            }
        }
    });

    // Mise à jour des compteurs globaux (défini dans dashboard.html)
    if (typeof updateMachineCounts === 'function') {
        updateMachineCounts();
    }
}
