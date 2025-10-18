// Winnipeg Tech Events Scraper & Telegram Sharing Web App
class EventScraperApp {
    constructor() {
        this.events = [];
        this.filteredEvents = [];
        this.selectedEvents = new Set();
        this.isOfflineMode = false;
        this.debugLogs = [];
        
        this.initializeApp();
    }

    initializeApp() {
        this.setupEventListeners();
        this.loadSampleData();
        this.loadEvents();
        this.log('info', 'App initialized successfully');
    }

    setupEventListeners() {
        // Refresh button
        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.refreshEvents();
        });

        // Debug toggle
        document.getElementById('debugToggle').addEventListener('click', () => {
            this.toggleDebugConsole();
        });

        // Telegram panel toggle
        document.getElementById('telegramToggle').addEventListener('click', () => {
            this.toggleTelegramPanel();
        });

        // Filter controls
        document.getElementById('dateFilter').addEventListener('change', () => {
            this.applyFilters();
        });
        document.getElementById('sourceFilter').addEventListener('change', () => {
            this.applyFilters();
        });
        document.getElementById('searchInput').addEventListener('input', () => {
            this.applyFilters();
        });

        // Telegram actions
        document.getElementById('selectAllBtn').addEventListener('click', () => {
            this.selectAllEvents();
        });
        document.getElementById('clearSelectionBtn').addEventListener('click', () => {
            this.clearSelection();
        });
        document.getElementById('sendTelegramBtn').addEventListener('click', () => {
            this.sendToTelegram();
        });
        document.getElementById('shareTelegramBtn').addEventListener('click', () => {
            this.shareViaUrl();
        });

        // Debug controls
        document.getElementById('clearLogsBtn').addEventListener('click', () => {
            this.clearDebugLogs();
        });
        document.getElementById('exportLogsBtn').addEventListener('click', () => {
            this.exportLogs();
        });

        // Alert dismiss buttons
        document.getElementById('errorDismiss').addEventListener('click', () => {
            this.hideAlert('error');
        });
        document.getElementById('warningDismiss').addEventListener('click', () => {
            this.hideAlert('warning');
        });
        document.getElementById('successDismiss').addEventListener('click', () => {
            this.hideAlert('success');
        });

        // Message preview updates
        document.getElementById('messagePreview').addEventListener('input', () => {
            this.updateCharCount();
        });
    }

    log(level, message, data = null) {
        const timestamp = new Date().toISOString();
        const logEntry = {
            timestamp,
            level,
            message,
            data
        };
        
        this.debugLogs.push(logEntry);
        this.updateDebugConsole();
        
        // Also log to browser console for development
        if (typeof console !== 'undefined') {
            console[level === 'error' ? 'error' : level === 'warning' ? 'warn' : 'log'](message, data);
        }
    }

    showAlert(type, message, duration = 0) {
        const banner = document.getElementById(`${type}Banner`);
        const messageEl = document.getElementById(`${type}Message`);
        
        if (banner && messageEl) {
            messageEl.textContent = message;
            banner.style.display = 'flex';
            
            if (duration > 0) {
                setTimeout(() => this.hideAlert(type), duration);
            }
        }
    }

    hideAlert(type) {
        const banner = document.getElementById(`${type}Banner`);
        if (banner) {
            banner.style.display = 'none';
        }
    }

    async loadEvents() {
        this.showLoading(true);
        this.log('info', 'Starting event loading process');
        
        try {
            const events = await this.fetchAllEvents();
            this.events = events;
            this.applyFilters();
            this.isOfflineMode = false;
            
            if (events.length === 0) {
                this.log('warning', 'No events fetched, switching to offline mode');
                this.switchToOfflineMode();
            } else {
                this.log('success', `Successfully loaded ${events.length} events`);
                this.showAlert('success', `Loaded ${events.length} tech events from Winnipeg!`, 3000);
            }
        } catch (error) {
            this.log('error', 'Failed to load events', error);
            this.showAlert('error', 'Could not access event data, using demo mode. Try Again...');
            this.switchToOfflineMode();
        } finally {
            this.showLoading(false);
        }
    }

    async fetchAllEvents() {
        try {
            this.log('info', 'Fetching events from backend API');
            const response = await fetch('/api/events?city=Winnipeg&categories=tech');
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const data = await response.json();
            this.log('success', `Fetched ${data.length} events from backend`);
            return data;
        } catch (error) {
            this.log('error', 'Failed to fetch from backend API', error.message);
            
            // Fallback to sample data
            return this.getSampleEvents();
        }
    }

    switchToOfflineMode() {
        this.isOfflineMode = true;
        this.events = this.getSampleEvents();
        this.applyFilters();
        this.showAlert('warning', 'Using demo data. Check your connection and try again.', 5000);
    }

    loadSampleData() {
        const sampleDataEl = document.getElementById('sampleData');
        if (sampleDataEl) {
            try {
                const sampleData = JSON.parse(sampleDataEl.textContent);
                this.sampleEvents = sampleData.events;
            } catch (error) {
                this.log('error', 'Failed to load sample data', error);
            }
        }
    }

    getSampleEvents() {
        return this.sampleEvents || [];
    }

    refreshEvents() {
        this.log('info', 'Manual refresh triggered');
        this.loadEvents();
    }

    showLoading(show) {
        const spinner = document.getElementById('loadingSpinner');
        const eventsList = document.getElementById('eventsList');
        const noEvents = document.getElementById('noEventsMessage');
        
        if (show) {
            spinner.style.display = 'flex';
            eventsList.style.display = 'none';
            noEvents.style.display = 'none';
        } else {
            spinner.style.display = 'none';
            eventsList.style.display = 'block';
        }
    }

    applyFilters() {
        const dateFilter = document.getElementById('dateFilter').value;
        const sourceFilter = document.getElementById('sourceFilter').value;
        const searchTerm = document.getElementById('searchInput').value.toLowerCase();

        this.filteredEvents = this.events.filter(event => {
            // Date filter
            if (dateFilter !== 'all') {
                const eventDate = new Date(event.startTime);
                const now = new Date();
                
                switch (dateFilter) {
                    case 'future':
                        if (eventDate < now) return false;
                        break;
                    case 'today':
                        if (!this.isSameDay(eventDate, now)) return false;
                        break;
                    case 'week':
                        if (!this.isThisWeek(eventDate)) return false;
                        break;
                    case 'nextweek':
                        if (!this.isNextWeek(eventDate)) return false;
                        break;
                }
            }

            // Source filter
            if (sourceFilter !== 'all' && event.source !== sourceFilter) {
                return false;
            }

            // Search filter
            if (searchTerm && !event.name.toLowerCase().includes(searchTerm) && 
                !event.description.toLowerCase().includes(searchTerm)) {
                return false;
            }

            return true;
        });

        this.renderEvents();
        this.updateTelegramPreview();
    }

    isSameDay(date1, date2) {
        return date1.getDate() === date2.getDate() &&
               date1.getMonth() === date2.getMonth() &&
               date1.getFullYear() === date2.getFullYear();
    }

    isThisWeek(date) {
        const now = new Date();
        const startOfWeek = new Date(now.setDate(now.getDate() - now.getDay()));
        const endOfWeek = new Date(now.setDate(now.getDate() - now.getDay() + 6));
        return date >= startOfWeek && date <= endOfWeek;
    }

    isNextWeek(date) {
        const now = new Date();
        const startOfNextWeek = new Date(now.setDate(now.getDate() - now.getDay() + 7));
        const endOfNextWeek = new Date(now.setDate(now.getDate() - now.getDay() + 13));
        return date >= startOfNextWeek && date <= endOfNextWeek;
    }

    renderEvents() {
        const eventsList = document.getElementById('eventsList');
        const noEvents = document.getElementById('noEventsMessage');

        if (this.filteredEvents.length === 0) {
            eventsList.style.display = 'none';
            noEvents.style.display = 'block';
            return;
        }

        eventsList.style.display = 'block';
        noEvents.style.display = 'none';

        // Group events by time period
        const groupedEvents = this.groupEventsByTime();
        
        let html = '';
        for (const [period, events] of Object.entries(groupedEvents)) {
            html += `<div class="event-group">
                <h3 class="event-group-title">${period}</h3>
                ${events.map(event => this.renderEventCard(event)).join('')}
            </div>`;
        }

        eventsList.innerHTML = html;

        // Add event listeners to checkboxes
        this.filteredEvents.forEach(event => {
            const checkbox = document.getElementById(`event-${event.id}`);
            if (checkbox) {
                checkbox.addEventListener('change', (e) => {
                    if (e.target.checked) {
                        this.selectedEvents.add(event.id);
                    } else {
                        this.selectedEvents.delete(event.id);
                    }
                    this.updateTelegramPreview();
                });
            }
        });
    }

    groupEventsByTime() {
        const now = new Date();
        const groups = {
            'Today': [],
            'This Week': [],
            'Next Week': [],
            'Later': []
        };

        this.filteredEvents.forEach(event => {
            const eventDate = new Date(event.startTime);
            
            if (this.isSameDay(eventDate, now)) {
                groups['Today'].push(event);
            } else if (this.isThisWeek(eventDate)) {
                groups['This Week'].push(event);
            } else if (this.isNextWeek(eventDate)) {
                groups['Next Week'].push(event);
            } else {
                groups['Later'].push(event);
            }
        });

        // Remove empty groups
        Object.keys(groups).forEach(key => {
            if (groups[key].length === 0) {
                delete groups[key];
            }
        });

        return groups;
    }

    renderEventCard(event) {
        const isSelected = this.selectedEvents.has(event.id);
        const startDate = new Date(event.startTime);
        const endDate = new Date(event.endTime);
        
        return `
            <div class="event-card ${isSelected ? 'selected' : ''}">
                <div class="event-header">
                    <div>
                        <h3 class="event-title">${this.escapeHtml(event.name)}</h3>
                        <div class="event-date">
                            <i class="fas fa-calendar"></i>
                            ${startDate.toLocaleDateString('en-US', { 
                                weekday: 'long', 
                                year: 'numeric', 
                                month: 'long', 
                                day: 'numeric' 
                            })}
                            <i class="fas fa-clock"></i>
                            ${startDate.toLocaleTimeString('en-US', { 
                                hour: 'numeric', 
                                minute: '2-digit',
                                hour12: true 
                            })} - ${endDate.toLocaleTimeString('en-US', { 
                                hour: 'numeric', 
                                minute: '2-digit',
                                hour12: true 
                            })}
                        </div>
                    </div>
                    <div class="event-source">${event.source}</div>
                </div>
                
                <p class="event-description">${this.escapeHtml(event.description)}</p>
                
                <div class="event-details">
                    ${event.venue ? `<div class="event-detail">
                        <i class="fas fa-map-marker-alt"></i>
                        <span>${this.escapeHtml(event.venue)}</span>
                    </div>` : ''}
                    ${event.group ? `<div class="event-detail">
                        <i class="fas fa-users"></i>
                        <span>${this.escapeHtml(event.group)}</span>
                    </div>` : ''}
                    ${event.attendeeCount ? `<div class="event-detail">
                        <i class="fas fa-user-plus"></i>
                        <span>${event.attendeeCount} attendees</span>
                    </div>` : ''}
                    ${event.price ? `<div class="event-detail">
                        <i class="fas fa-tag"></i>
                        <span>${this.escapeHtml(event.price)}</span>
                    </div>` : ''}
                </div>
                
                <div class="event-actions">
                    <input type="checkbox" 
                           id="event-${event.id}" 
                           class="event-checkbox"
                           ${isSelected ? 'checked' : ''}>
                    <label for="event-${event.id}">Select for sharing</label>
                    <a href="${event.url}" target="_blank" class="event-link">
                        <i class="fas fa-external-link-alt"></i> View Event
                    </a>
                </div>
            </div>
        `;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    selectAllEvents() {
        this.selectedEvents.clear();
        this.filteredEvents.forEach(event => {
            this.selectedEvents.add(event.id);
        });
        this.renderEvents();
        this.updateTelegramPreview();
        this.log('info', `Selected all ${this.filteredEvents.length} events`);
    }

    clearSelection() {
        this.selectedEvents.clear();
        this.renderEvents();
        this.updateTelegramPreview();
        this.log('info', 'Cleared event selection');
    }

    updateTelegramPreview() {
        const selectedEvents = this.events.filter(event => this.selectedEvents.has(event.id));
        const messagePreview = document.getElementById('messagePreview');
        
        if (selectedEvents.length === 0) {
            messagePreview.value = '';
            messagePreview.placeholder = 'Select events to see preview...';
        } else {
            const message = this.generateTelegramMessage(selectedEvents);
            messagePreview.value = message;
        }
        
        this.updateCharCount();
    }

    generateTelegramMessage(events) {
        const now = new Date();
        const dateStr = now.toLocaleDateString('en-US', { 
            weekday: 'long', 
            year: 'numeric', 
            month: 'long', 
            day: 'numeric' 
        });
        
        let message = `🚀 *Winnipeg Tech Events - ${dateStr}*\n\n`;
        
        const grouped = this.groupEventsByTime();
        for (const [period, periodEvents] of Object.entries(grouped)) {
            const selectedPeriodEvents = periodEvents.filter(event => this.selectedEvents.has(event.id));
            if (selectedPeriodEvents.length > 0) {
                message += `*${period}:*\n`;
                selectedPeriodEvents.forEach(event => {
                    const startDate = new Date(event.startTime);
                    const timeStr = startDate.toLocaleTimeString('en-US', { 
                        hour: 'numeric', 
                        minute: '2-digit',
                        hour12: true 
                    });
                    
                    message += `• ${event.name}\n`;
                    message += `  📅 ${startDate.toLocaleDateString('en-US', { 
                        month: 'short', 
                        day: 'numeric' 
                    })} at ${timeStr}\n`;
                    if (event.venue) {
                        message += `  📍 ${event.venue}\n`;
                    }
                    if (event.price && event.price !== 'Free') {
                        message += `  💰 ${event.price}\n`;
                    }
                    message += `  🔗 [View Event](${event.url})\n\n`;
                });
            }
        }
        
        message += `\n_Shared via Winnipeg Tech Events Tracker_`;
        
        return message;
    }

    updateCharCount() {
        const messagePreview = document.getElementById('messagePreview');
        const charCount = document.getElementById('charCount');
        const charWarning = document.getElementById('charWarning');
        
        const count = messagePreview.value.length;
        charCount.textContent = count;
        
        if (count > 3800) {
            charWarning.style.display = 'inline';
            charWarning.style.color = 'var(--error-color)';
            charWarning.textContent = '⚠️ Over limit!';
        } else if (count > 3500) {
            charWarning.style.display = 'inline';
            charWarning.style.color = 'var(--warning-color)';
            charWarning.textContent = '⚠️ Approaching limit';
        } else {
            charWarning.style.display = 'none';
        }
    }

    async sendToTelegram() {
        const botToken = document.getElementById('botToken').value.trim();
        const chatId = document.getElementById('chatId').value.trim();
        const message = document.getElementById('messagePreview').value;

        if (!botToken || !chatId) {
            this.showAlert('error', 'Please provide both Bot Token and Chat ID to send via Telegram Bot API');
            return;
        }

        if (!message.trim()) {
            this.showAlert('error', 'Please select events to share');
            return;
        }

        try {
            this.log('info', 'Sending message to Telegram via Bot API');
            
            const response = await fetch(`https://api.telegram.org/bot${botToken}/sendMessage`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    chat_id: chatId,
                    text: message,
                    parse_mode: 'Markdown',
                    disable_web_page_preview: true
                })
            });

            const result = await response.json();

            if (result.ok) {
                this.showAlert('success', 'Message sent to Telegram successfully!', 3000);
                this.log('success', 'Message sent to Telegram successfully');
            } else {
                throw new Error(result.description || 'Unknown error');
            }
        } catch (error) {
            this.log('error', 'Failed to send to Telegram', error.message);
            this.showAlert('error', `Failed to send to Telegram: ${error.message}`);
        }
    }

    shareViaUrl() {
        const message = document.getElementById('messagePreview').value;
        
        if (!message.trim()) {
            this.showAlert('error', 'Please select events to share');
            return;
        }

        const encodedMessage = encodeURIComponent(message);
        const shareUrl = `https://t.me/share/url?text=${encodedMessage}`;
        
        window.open(shareUrl, '_blank');
        this.log('info', 'Opened Telegram share URL');
    }

    toggleDebugConsole() {
        const console = document.getElementById('debugConsole');
        const isVisible = console.style.display !== 'none';
        console.style.display = isVisible ? 'none' : 'block';
    }

    toggleTelegramPanel() {
        const content = document.getElementById('telegramContent');
        const toggle = document.getElementById('telegramToggle');
        const isVisible = content.style.display !== 'none';
        
        content.style.display = isVisible ? 'none' : 'block';
        toggle.innerHTML = isVisible ? '<i class="fas fa-chevron-down"></i>' : '<i class="fas fa-chevron-up"></i>';
    }

    updateDebugConsole() {
        const logsContainer = document.getElementById('debugLogs');
        if (!logsContainer) return;

        const logsHtml = this.debugLogs.slice(-50).map(log => `
            <div class="debug-log-entry ${log.level}">
                <span class="debug-log-timestamp">${new Date(log.timestamp).toLocaleTimeString()}</span>
                <strong>[${log.level.toUpperCase()}]</strong> ${log.message}
                ${log.data ? `<pre>${JSON.stringify(log.data, null, 2)}</pre>` : ''}
            </div>
        `).join('');

        logsContainer.innerHTML = logsHtml;
        logsContainer.scrollTop = logsContainer.scrollHeight;
    }

    clearDebugLogs() {
        this.debugLogs = [];
        this.updateDebugConsole();
        this.log('info', 'Debug logs cleared');
    }

    exportLogs() {
        const logsData = {
            timestamp: new Date().toISOString(),
            logs: this.debugLogs,
            appState: {
                eventsCount: this.events.length,
                filteredEventsCount: this.filteredEvents.length,
                selectedEventsCount: this.selectedEvents.size,
                isOfflineMode: this.isOfflineMode
            }
        };

        const blob = new Blob([JSON.stringify(logsData, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `winnipeg-tech-events-logs-${new Date().toISOString().split('T')[0]}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        this.log('info', 'Debug logs exported');
    }
}

// Scraper Classes
class BaseScraper {
    constructor() {
        this.baseUrl = '';
    }

    async fetchEvents() {
        throw new Error('fetchEvents must be implemented by subclass');
    }

    parseEvent(eventData) {
        throw new Error('parseEvent must be implemented by subclass');
    }

    parseDate(dateString) {
        // Handle various date formats
        if (!dateString) return null;
        
        // Handle day names (e.g., "Thu", "Saturday")
        const dayNames = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];
        const shortDayNames = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];
        
        const lowerDateStr = dateString.toLowerCase().trim();
        const dayIndex = dayNames.indexOf(lowerDateStr) !== -1 ? dayNames.indexOf(lowerDateStr) : shortDayNames.indexOf(lowerDateStr);
        
        if (dayIndex !== -1) {
            // Find next occurrence of this day after today
            const today = new Date();
            const todayDay = today.getDay();
            let daysUntilTarget = dayIndex - todayDay;
            
            if (daysUntilTarget <= 0) {
                daysUntilTarget += 7; // Next week
            }
            
            const targetDate = new Date(today);
            targetDate.setDate(today.getDate() + daysUntilTarget);
            return targetDate;
        }
        
        // Try parsing as regular date
        const parsed = new Date(dateString);
        return isNaN(parsed.getTime()) ? null : parsed;
    }
}

class MeetupScraper extends BaseScraper {
    constructor() {
        super();
        this.baseUrl = 'https://www.meetup.com/find/?location=ca--mb--Winnipeg&source=EVENTS&categoryId=546';
    }

    async fetchEvents() {
        // Note: This is a simplified implementation
        // In a real implementation, you'd need to handle CORS and scraping
        // For now, return sample data
        return [
            {
                id: 'meetup-1',
                name: 'Winnipeg AI & Machine Learning Meetup',
                description: 'Monthly meetup discussing AI trends and applications in business.',
                source: 'meetup',
                url: 'https://meetup.com/example',
                venue: 'Innovation Hub',
                group: 'Winnipeg AI Community',
                attendeeCount: 45,
                price: 'Free',
                startTime: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
                endTime: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000).toISOString(),
                dateString: 'Next Thursday'
            }
        ];
    }
}

class EventbriteScraper extends BaseScraper {
    constructor() {
        super();
        this.baseUrl = 'https://www.eventbrite.ca/d/canada--winnipeg/tech-event/';
    }

    async fetchEvents() {
        // Note: This is a simplified implementation
        // In a real implementation, you'd need to handle CORS and scraping
        return [
            {
                id: 'eventbrite-1',
                name: 'Winnipeg Tech Conference 2025',
                description: 'Annual technology conference featuring local and international speakers.',
                source: 'eventbrite',
                url: 'https://eventbrite.com/example',
                venue: 'Convention Centre',
                group: 'Winnipeg Tech Events',
                attendeeCount: 200,
                price: '$50',
                startTime: '2025-11-05T17:00:00-06:00',
                endTime: '2025-11-05T21:00:00-06:00',
                dateString: 'November 5, 2025'
            }
        ];
    }
}

class DevEventsScraper extends BaseScraper {
    constructor() {
        super();
        this.baseUrl = 'https://dev.events/NA/CA';
    }

    async fetchEvents() {
        // Note: This is a simplified implementation
        // In a real implementation, you'd need to handle CORS and scraping
        return [
            {
                id: 'devevents-1',
                name: 'Winnipeg Developer Workshop',
                description: 'Hands-on coding workshop for developers of all levels.',
                source: 'devevents',
                url: 'https://dev.events/example',
                venue: 'TechSpace Winnipeg',
                group: 'Winnipeg Developers',
                attendeeCount: 30,
                price: 'Free',
                startTime: '2025-02-25T09:00:00-06:00',
                endTime: '2025-02-27T17:00:00-06:00',
                dateString: 'Feb 25-27, 2025'
            }
        ];
    }
}

// Initialize the app when the page loads
document.addEventListener('DOMContentLoaded', () => {
    window.eventApp = new EventScraperApp();
});
