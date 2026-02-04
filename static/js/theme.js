document.addEventListener('DOMContentLoaded', () => {
    // Charger le thÃ¨me sauvegardÃ© ou utiliser light par dÃ©faut
    const savedTheme = localStorage.getItem('theme') || 'light';
    setTheme(savedTheme);

    const toggleBtn = document.getElementById('theme-toggle');
    if (toggleBtn) {
        toggleBtn.addEventListener('click', () => {
            const current = document.documentElement.getAttribute('data-theme');
            const newTheme = current === 'dark' ? 'light' : 'dark';
            setTheme(newTheme);
        });
    }
});

function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('theme', theme);

    // Mettre Ã  jour l'icÃ´ne
    const icon = document.querySelector('#theme-toggle .icon');
    if (icon) {
        icon.textContent = theme === 'dark' ? 'â˜€ï¸' : 'ðŸŒ™';
    }
}
