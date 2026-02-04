/**
 * Module API rÃ©utilisable
 * GÃ¨re les requÃªtes HTTP avec CSRF, erreurs, etc.
 */

import { showError } from './notifications.js';

/**
 * RÃ©cupÃ¨re le token CSRF
 */
function getCSRFToken() {
  const meta = document.querySelector('meta[name="csrf-token"]');
  return meta ? meta.getAttribute('content') : '';
}

/**
 * Configuration par dÃ©faut pour fetch
 */
const defaultConfig = {
  headers: {
    'Content-Type': 'application/json',
  },
  credentials: 'same-origin',
};

/**
 * Classe pour gÃ©rer les requÃªtes API
 */
export class APIClient {
  constructor(baseURL = '/api') {
    this.baseURL = baseURL;
  }

  /**
   * RequÃªte gÃ©nÃ©rique
   */
  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;

    // Fusionner les options
    const config = {
      ...defaultConfig,
      ...options,
      headers: {
        ...defaultConfig.headers,
        ...options.headers,
      },
    };

    // Ajouter le token CSRF pour les mÃ©thodes modifiant les donnÃ©es
    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(config.method?.toUpperCase())) {
      config.headers['X-CSRF-Token'] = getCSRFToken();
    }

    try {
      const response = await fetch(url, config);

      // Gestion des erreurs HTTP
      if (!response.ok) {
        const error = await this.handleError(response);
        throw error;
      }

      // Parser la rÃ©ponse JSON si applicable
      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      }

      return response;
    } catch (error) {
      // Afficher l'erreur Ã  l'utilisateur
      if (error.message && !error.userNotified) {
        showError(error.message);
      }
      throw error;
    }
  }

  /**
   * GÃ¨re les erreurs de rÃ©ponse
   */
  async handleError(response) {
    let message = `Erreur ${response.status}`;

    try {
      const data = await response.json();
      message = data.error || data.message || message;
    } catch (e) {
      message = response.statusText || message;
    }

    const error = new Error(message);
    error.status = response.status;
    error.response = response;
    return error;
  }

  /**
   * GET request
   */
  async get(endpoint, params = {}) {
    const queryString = new URLSearchParams(params).toString();
    const url = queryString ? `${endpoint}?${queryString}` : endpoint;

    return this.request(url, {
      method: 'GET',
    });
  }

  /**
   * POST request
   */
  async post(endpoint, data = {}) {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  /**
   * PUT request
   */
  async put(endpoint, data = {}) {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  /**
   * DELETE request
   */
  async delete(endpoint) {
    return this.request(endpoint, {
      method: 'DELETE',
    });
  }

  /**
   * PATCH request
   */
  async patch(endpoint, data = {}) {
    return this.request(endpoint, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  }
}

// Instance singleton
export const api = new APIClient();

// Exports des mÃ©thodes raccourcies
export const get = (endpoint, params) => api.get(endpoint, params);
export const post = (endpoint, data) => api.post(endpoint, data);
export const put = (endpoint, data) => api.put(endpoint, data);
export const del = (endpoint) => api.delete(endpoint);
export const patch = (endpoint, data) => api.patch(endpoint, data);

/**
 * API spÃ©cifique machines
 */
export const machinesAPI = {
  getAll: () => get('/machines'),
  getById: (id) => get(`/machines/${id}`),
  create: (data) => post('/machines', data),
  update: (id, data) => put(`/machines/${id}`, data),
  delete: (id) => del(`/machines/${id}`),
  getHistory: (id, duration = '24h') => get(`/machine/${id}/history`, { duration }),
  getStatus: (id) => get(`/machine/${id}/status`),
};

/**
 * API spÃ©cifique utilisateurs
 */
export const usersAPI = {
  getAll: () => get('/users'),
  create: (data) => post('/users', data),
  update: (username, data) => put(`/users/${username}`, data),
  delete: (username) => del(`/users/${username}`),
  changePassword: (data) => post('/users/password', data),
};

/**
 * WebSocket helper
 */
export class WebSocketClient {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000;
    this.handlers = {
      open: [],
      message: [],
      error: [],
      close: [],
    };
  }

  /**
   * Connecte le WebSocket
   */
  connect() {
    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = (event) => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        this.handlers.open.forEach(handler => handler(event));
      };

      this.ws.onmessage = (event) => {
        this.handlers.message.forEach(handler => handler(event));
      };

      this.ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        this.handlers.error.forEach(handler => handler(event));
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket closed');
        this.handlers.close.forEach(handler => handler(event));
        this.attemptReconnect();
      };
    } catch (error) {
      console.error('Failed to connect WebSocket:', error);
      this.attemptReconnect();
    }
  }

  /**
   * Tente une reconnexion avec backoff exponentiel
   */
  attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

      console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

      setTimeout(() => this.connect(), delay);
    } else {
      console.error('Max reconnection attempts reached');
      showError('Connexion WebSocket perdue. Rechargez la page.');
    }
  }

  /**
   * Envoie un message
   */
  send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(typeof data === 'string' ? data : JSON.stringify(data));
    } else {
      console.warn('WebSocket not connected');
    }
  }

  /**
   * Ferme la connexion
   */
  close() {
    this.maxReconnectAttempts = 0; // DÃ©sactiver reconnexion
    if (this.ws) {
      this.ws.close();
    }
  }

  /**
   * Ajoute un gestionnaire d'Ã©vÃ©nement
   */
  on(event, handler) {
    if (this.handlers[event]) {
      this.handlers[event].push(handler);
    }
  }

  /**
   * Retire un gestionnaire d'Ã©vÃ©nement
   */
  off(event, handler) {
    if (this.handlers[event]) {
      const index = this.handlers[event].indexOf(handler);
      if (index > -1) {
        this.handlers[event].splice(index, 1);
      }
    }
  }
}
