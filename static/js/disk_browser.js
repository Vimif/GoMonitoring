// Navigateur de fichiers interactif

let currentPath = '/';
let pathHistory = [];

// Fonction appelÃ©e lors du clic sur un disque
function browseDisk(mountPoint) {
    currentPath = mountPoint;
    pathHistory = [mountPoint];
    document.getElementById('file-browser-section').style.display = 'block';
    loadDirectory(currentPath);

    // Scroll vers le navigateur
    document.getElementById('file-browser-section').scrollIntoView({ behavior: 'smooth' });
}

// Charge et affiche le contenu d'un rÃ©pertoire
async function loadDirectory(path) {
    const fileList = document.getElementById('file-list');
    const currentPathSpan = document.getElementById('current-path');
    const btnBack = document.getElementById('btn-back');

    // Afficher le loading
    fileList.innerHTML = '<div class="loading"><div class="spinner"></div></div>';
    currentPathSpan.textContent = path;

    // Activer/dÃ©sactiver le bouton retour
    btnBack.disabled = pathHistory.length <= 1;

    try {
        const response = await fetch(`/api/machine/${machineId}/browse?path=${encodeURIComponent(path)}`);

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Erreur inconnue');
        }

        const listing = await response.json();
        renderDirectory(listing);
        currentPath = path;

    } catch (error) {
        fileList.innerHTML = `
            <div class="file-entry" style="justify-content: center; color: #e17055;">
                Erreur: ${error.message}
            </div>
        `;
    }
}

// Rend le contenu du rÃ©pertoire
function renderDirectory(listing) {
    const fileList = document.getElementById('file-list');

    if (!listing.Entries || listing.Entries.length === 0) {
        fileList.innerHTML = `
            <div class="file-entry" style="justify-content: center; color: #636e72;">
                RÃ©pertoire vide
            </div>
        `;
        return;
    }

    // Trier: dossiers d'abord, puis fichiers, alphabÃ©tiquement
    const entries = [...listing.Entries].sort((a, b) => {
        if (a.IsDir && !b.IsDir) return -1;
        if (!a.IsDir && b.IsDir) return 1;
        return a.Name.localeCompare(b.Name);
    });

    let html = '';
    for (const entry of entries) {
        const icon = entry.IsDir ? 'ðŸ“' : getFileIcon(entry.Name);
        const size = entry.IsDir ? '-' : formatSize(entry.Size);
        const className = entry.IsDir ? 'folder' : 'file';
        const onClick = entry.IsDir
            ? `onclick="navigateTo(this.dataset.path)"`
            : '';

        html += `
            <div class="file-entry ${className}" ${entry.IsDir ? `data-path="${escapeHtml(entry.Path)}" ${onClick}` : ''}>
                <span class="file-icon">${icon}</span>
                <span class="file-name">${escapeHtml(entry.Name)}</span>
                <span class="file-size">${size}</span>
                <span class="file-perms">${entry.Permissions || '-'}</span>
                <span class="file-owner">${entry.Owner || '-'}:${entry.Group || '-'}</span>
            </div>
        `;
    }

    fileList.innerHTML = html;
}

// Navigation vers un sous-rÃ©pertoire
function navigateTo(path) {
    pathHistory.push(path);
    loadDirectory(path);
}

// Retour au rÃ©pertoire parent
function goBack() {
    if (pathHistory.length > 1) {
        pathHistory.pop();
        const previousPath = pathHistory[pathHistory.length - 1];
        loadDirectory(previousPath);
    }
}

// Formate la taille en bytes
function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// Retourne une icÃ´ne basÃ©e sur l'extension du fichier
function getFileIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    const icons = {
        // Documents
        'txt': 'ðŸ“„', 'doc': 'ðŸ“„', 'docx': 'ðŸ“„', 'pdf': 'ðŸ“•',
        'xls': 'ðŸ“Š', 'xlsx': 'ðŸ“Š', 'csv': 'ðŸ“Š',
        'ppt': 'ðŸ“™', 'pptx': 'ðŸ“™',
        // Code
        'js': 'ðŸ“œ', 'ts': 'ðŸ“œ', 'py': 'ðŸ', 'go': 'ðŸ”·',
        'java': 'â˜•', 'c': 'ðŸ“', 'cpp': 'ðŸ“', 'h': 'ðŸ“',
        'html': 'ðŸŒ', 'css': 'ðŸŽ¨', 'json': 'ðŸ“‹', 'xml': 'ðŸ“‹',
        'yaml': 'ðŸ“‹', 'yml': 'ðŸ“‹', 'md': 'ðŸ“',
        'sh': 'âš¡', 'bash': 'âš¡',
        // Images
        'jpg': 'ðŸ–¼ï¸', 'jpeg': 'ðŸ–¼ï¸', 'png': 'ðŸ–¼ï¸', 'gif': 'ðŸ–¼ï¸',
        'svg': 'ðŸ–¼ï¸', 'ico': 'ðŸ–¼ï¸', 'bmp': 'ðŸ–¼ï¸',
        // Archives
        'zip': 'ðŸ“¦', 'tar': 'ðŸ“¦', 'gz': 'ðŸ“¦', 'rar': 'ðŸ“¦', '7z': 'ðŸ“¦',
        // Audio/Video
        'mp3': 'ðŸŽµ', 'wav': 'ðŸŽµ', 'flac': 'ðŸŽµ',
        'mp4': 'ðŸŽ¬', 'avi': 'ðŸŽ¬', 'mkv': 'ðŸŽ¬', 'mov': 'ðŸŽ¬',
        // Autres
        'log': 'ðŸ“‹', 'conf': 'âš™ï¸', 'cfg': 'âš™ï¸', 'ini': 'âš™ï¸',
        'sql': 'ðŸ—ƒï¸', 'db': 'ðŸ—ƒï¸',
        'key': 'ðŸ”‘', 'pem': 'ðŸ”‘', 'crt': 'ðŸ“œ',
    };
    return icons[ext] || 'ðŸ“„';
}

// Ã‰chappe les caractÃ¨res HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
