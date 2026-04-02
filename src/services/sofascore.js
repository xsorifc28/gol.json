import axios from 'axios';

const PROXY_URL = 'https://cors-anywhere.com/';
const HOSTS = ['api.sofascore.com', 'www.sofascore.com', 'api.sofascore.app'];

export class SofascoreService {
  async fetchWithFallback(endpoint) {
    let lastError = null;
    for (const host of HOSTS) {
      try {
        const url = `https://${host}/api/v1${endpoint}`;
        const response = await axios.get(PROXY_URL + url, {
          headers: { 'X-Requested-With': 'XMLHttpRequest' }
        });
        return response.data;
      } catch (err) {
        lastError = err;
      }
    }
    throw lastError;
  }

  async getLiveEvents() {
    try {
      const data = await this.fetchWithFallback('/sport/football/events/live');
      if (data.events && data.events.length > 0) return data.events;
    } catch (e) {}
    const today = new Date().toISOString().split('T')[0];
    const data = await this.fetchWithFallback(`/sport/football/scheduled-events/${today}`);
    return data.events || [];
  }

  async getEventDetails(eventId) {
    const data = await this.fetchWithFallback(`/event/${eventId}`);
    return data.event;
  }

  async getIncidents(eventId) {
    const data = await this.fetchWithFallback(`/event/${eventId}/incidents`);
    return data.incidents || [];
  }
}

export const sofascore = new SofascoreService();
