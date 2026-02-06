/**
 * ChartManager - Gestion des graphiques historiques
 * Graphiques : CPU, Mémoire
 */
class ChartManager {
    constructor(machineId, totalMemory) {
        this.machineId = machineId;
        this.totalMemory = totalMemory;
        this.charts = {};
        this.currentPeriod = '24h';
        this.data = null;
        this.pendingPeriodChange = null; // Pour le debounce

        // Couleurs
        this.colors = {
            cpu: '#3b82f6',
            memory: '#10b981',
            memoryTotal: '#6b7280'
        };
    }

    // Utilitaire debounce
    debounce(func, wait) {
        let timeout;
        return (...args) => {
            clearTimeout(timeout);
            timeout = setTimeout(() => func.apply(this, args), wait);
        };
    }

    // Mapping des périodes vers durées API
    getPeriodDuration(period) {
        const map = {
            '1h': '1h',
            '6h': '6h',
            '24h': '24h',
            '7d': '168h',
            '30d': '720h'
        };
        return map[period] || '24h';
    }

    // Charger les données depuis l'API
    async loadData(period) {
        this.currentPeriod = period;
        const duration = this.getPeriodDuration(period);

        try {
            const response = await fetch(`/api/machine/${this.machineId}/history?duration=${duration}`);
            if (!response.ok) throw new Error('Erreur API');
            this.data = await response.json();
            return this.data;
        } catch (e) {
            console.error('Erreur chargement données:', e);
            return null;
        }
    }

    // Formater les bytes
    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    // Formater le timestamp pour les labels
    formatTimestamp(timestamp, period) {
        const date = new Date(timestamp);
        if (period === '1h' || period === '6h') {
            return date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
        } else if (period === '24h') {
            return date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
        } else {
            return date.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' }) +
                   ' ' + date.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' });
        }
    }

    // Configuration commune des graphiques
    getCommonOptions(tooltipCallback) {
        const style = getComputedStyle(document.body);
        const borderColor = style.getPropertyValue('--border-color').trim() || '#e5e7eb';
        const mutedColor = style.getPropertyValue('--text-muted').trim() || '#6b7280';

        return {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                mode: 'index',
                intersect: false
            },
            plugins: {
                legend: {
                    display: true,
                    position: 'top',
                    align: 'end',
                    labels: {
                        boxWidth: 12,
                        padding: 15,
                        color: mutedColor,
                        font: { size: 12 }
                    },
                    onClick: (e, legendItem, legend) => {
                        const index = legendItem.datasetIndex;
                        const chart = legend.chart;
                        chart.getDatasetMeta(index).hidden = !chart.getDatasetMeta(index).hidden;
                        chart.update();
                    }
                },
                tooltip: {
                    backgroundColor: 'rgba(0, 0, 0, 0.85)',
                    titleColor: '#fff',
                    bodyColor: '#fff',
                    borderColor: borderColor,
                    borderWidth: 1,
                    padding: 12,
                    displayColors: true,
                    callbacks: tooltipCallback
                }
            },
            scales: {
                x: {
                    display: true,
                    grid: {
                        display: false
                    },
                    ticks: {
                        color: mutedColor,
                        maxRotation: 0,
                        maxTicksLimit: 8,
                        font: { size: 10 }
                    }
                }
            }
        };
    }

    // Détruire un graphique existant
    destroyChart(chartId) {
        if (this.charts[chartId]) {
            this.charts[chartId].destroy();
            delete this.charts[chartId];
        }
    }

    // Rendre tous les graphiques
    renderAllCharts(data) {
        if (!data || data.length === 0) return;

        const timestamps = data.map(d => this.formatTimestamp(d.timestamp, this.currentPeriod));

        this.renderCPUChart(data, timestamps);
        this.renderMemoryChart(data, timestamps);
    }

    // Graphique CPU
    renderCPUChart(data, timestamps) {
        const ctx = document.getElementById('cpuChart');
        if (!ctx) return;

        this.destroyChart('cpu');

        const cpuData = data.map(d => d.cpu || 0);
        const style = getComputedStyle(document.body);
        const borderColor = style.getPropertyValue('--border-color').trim() || '#e5e7eb';
        const mutedColor = style.getPropertyValue('--text-muted').trim() || '#6b7280';

        const options = this.getCommonOptions({
            label: (ctx) => `CPU: ${ctx.parsed.y.toFixed(1)}%`
        });

        options.scales.y = {
            min: 0,
            max: 100,
            grid: {
                color: borderColor,
                borderDash: [3, 3]
            },
            ticks: {
                color: mutedColor,
                callback: v => v + '%',
                stepSize: 25
            }
        };

        this.charts.cpu = new Chart(ctx, {
            type: 'line',
            data: {
                labels: timestamps,
                datasets: [{
                    label: 'CPU',
                    data: cpuData,
                    borderColor: this.colors.cpu,
                    backgroundColor: this.colors.cpu + '15',
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0,
                    pointHoverRadius: 5,
                    borderWidth: 2
                }]
            },
            options: options
        });
    }

    // Graphique Mémoire
    renderMemoryChart(data, timestamps) {
        const ctx = document.getElementById('memoryChart');
        if (!ctx) return;

        this.destroyChart('memory');

        const memoryUsed = data.map(d => d.memory_used || 0);
        const style = getComputedStyle(document.body);
        const borderColor = style.getPropertyValue('--border-color').trim() || '#e5e7eb';
        const mutedColor = style.getPropertyValue('--text-muted').trim() || '#6b7280';

        const self = this;
        const options = this.getCommonOptions({
            label: function(ctx) {
                const label = ctx.dataset.label || '';
                const value = self.formatBytes(ctx.parsed.y);
                return `${label}: ${value}`;
            }
        });

        options.scales.y = {
            min: 0,
            max: this.totalMemory,
            grid: {
                color: borderColor,
                borderDash: [3, 3]
            },
            ticks: {
                color: mutedColor,
                callback: v => this.formatBytes(v),
                maxTicksLimit: 5
            }
        };

        this.charts.memory = new Chart(ctx, {
            type: 'line',
            data: {
                labels: timestamps,
                datasets: [
                    {
                        label: 'Utilisée',
                        data: memoryUsed,
                        borderColor: this.colors.memory,
                        backgroundColor: this.colors.memory + '15',
                        fill: true,
                        tension: 0.4,
                        pointRadius: 0,
                        pointHoverRadius: 5,
                        borderWidth: 2
                    },
                    {
                        label: 'Total',
                        data: data.map(() => this.totalMemory),
                        borderColor: this.colors.memoryTotal,
                        borderDash: [5, 5],
                        fill: false,
                        pointRadius: 0,
                        borderWidth: 1
                    }
                ]
            },
            options: options
        });
    }

    // Changer de période
    async changePeriod(period) {
        // Mise à jour UI boutons
        document.querySelectorAll('.period-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.period === period);
        });

        // Afficher loading
        document.querySelectorAll('.chart-container').forEach(container => {
            container.classList.add('loading');
        });

        // Charger les données
        const data = await this.loadData(period);

        // Masquer loading
        document.querySelectorAll('.chart-container').forEach(container => {
            container.classList.remove('loading');
        });

        if (data) {
            this.renderAllCharts(data);
        }
    }

    // Initialisation
    async init() {
        // Créer une version debounced de changePeriod (300ms)
        const debouncedChange = this.debounce((period) => {
            this.changePeriod(period);
        }, 300);

        // Configurer les boutons de période avec debounce
        document.querySelectorAll('.period-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                // Mise à jour UI immédiate pour le feedback
                document.querySelectorAll('.period-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');

                // Appel debounced pour le chargement des données
                debouncedChange(btn.dataset.period);
            });
        });

        // Charger les données initiales
        await this.changePeriod(this.currentPeriod);
    }
}

// Export pour utilisation globale
window.ChartManager = ChartManager;
