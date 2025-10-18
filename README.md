# Winnipeg Tech Events Scraper & Telegram Sharing Web App

A production-grade web application that discovers, aggregates, and shares technology-related events happening in Winnipeg, Manitoba. The application reliably fetches events from multiple sources, handles failures gracefully, and enables seamless sharing of event digests to Telegram groups.

## Features

### 🔍 Multi-Source Event Aggregation
- **Meetup.com**: Scrapes tech events from Winnipeg area
- **Eventbrite**: Fetches tech events with smart datetime parsing
- **Dev.events**: Discovers developer events in Winnipeg/Manitoba
- **Smart Date Handling**: Intelligently parses various date formats including day names

### 🛡️ Robust Error Handling
- **Automatic Fallback**: Switches to sample data if scraping fails
- **Graceful Degradation**: Individual source failures don't break the app
- **Real-time Debug Console**: Comprehensive logging and diagnostics
- **UI Error Alerts**: Clear user notifications for all error states

### 📱 Telegram Integration
- **Bot API**: Direct messaging to configured Telegram groups
- **Share URLs**: Pre-filled messages for manual sharing
- **Character Limits**: Smart warnings for message length
- **Message Preview**: Real-time preview of formatted messages

### 🎨 Modern Web UI
- **Responsive Design**: Works on desktop and mobile
- **Dark Mode Support**: Automatic theme detection
- **Smart Filtering**: By date range, source, and search terms
- **Event Grouping**: Organizes events by "Today", "This Week", etc.

## Quick Start

### Prerequisites
- Go 1.24.1 or later
- Modern web browser with JavaScript enabled

### Installation

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd event_calendar
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run the application**:
   ```bash
   go run cmd/main.go
   ```

4. **Open your browser**:
   Navigate to `http://localhost:8080`

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `PERIOD_DAYS` | Event search period in days | `30` |

### Telegram Bot Setup (Optional)

1. **Create a Telegram Bot**:
   - Message [@BotFather](https://t.me/botfather) on Telegram
   - Create a new bot with `/newbot`
   - Save the bot token

2. **Get Chat ID**:
   - Add your bot to a group
   - Send a message in the group
   - Visit `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
   - Find the chat ID in the response

3. **Configure in UI**:
   - Enter bot token and chat ID in the Telegram panel
   - Use "Send to Telegram" for direct messaging
   - Use "Share via URL" for manual sharing

## Architecture

### Backend (Go)
- **Main Server**: `cmd/main.go` - HTTP server and API endpoints
- **Models**: `internal/models/event.go` - Event data structure
- **Scrapers**: `pkg/*/scraper.go` - Source-specific event scrapers
- **Aggregator**: `pkg/aggregator/` - Event collection and processing

### Frontend (JavaScript)
- **Web UI**: `web/index.html` - Main application interface
- **Styling**: `web/styles.css` - Modern, responsive CSS
- **Logic**: `web/app.js` - Client-side application logic

### Data Flow
1. **Frontend** requests events from `/api/events`
2. **Backend** aggregates events from multiple scrapers
3. **Scrapers** fetch and parse events from external sources
4. **Aggregator** removes duplicates and sorts by date
5. **Frontend** displays events with filtering and Telegram sharing

## Validation & Testing

### Manual Testing Checklist

#### ✅ Basic Functionality
- [ ] Application loads without errors
- [ ] Events display in grouped format (Today, This Week, etc.)
- [ ] Filters work correctly (date range, source, search)
- [ ] Event selection works for Telegram sharing

#### ✅ Error Handling
- [ ] Network failure shows warning banner
- [ ] Fallback to sample data works
- [ ] Debug console shows detailed logs
- [ ] "Try Again" button resets state

#### ✅ Telegram Integration
- [ ] Message preview updates when selecting events
- [ ] Character count shows and warns at limits
- [ ] Share via URL opens Telegram with pre-filled message
- [ ] Bot API works with valid credentials (if configured)

#### ✅ Date Handling
- [ ] Events are sorted chronologically
- [ ] "Today" shows only today's events
- [ ] "This Week" shows current week events
- [ ] "Next Week" shows upcoming week events

#### ✅ UI/UX
- [ ] Responsive design works on mobile
- [ ] Dark mode adapts to system preference
- [ ] Loading states show during data fetch
- [ ] Error banners are dismissible

### Automated Testing

Run the test suite:
```bash
go test ./...
```

### Load Testing

Test with multiple concurrent requests:
```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Run load test
hey -n 100 -c 10 http://localhost:8080/api/events
```

## Troubleshooting

### Common Issues

#### Events Not Loading
1. **Check browser console** for JavaScript errors
2. **Verify backend is running** on correct port
3. **Check network tab** for failed API requests
4. **Review debug console** for detailed error logs

#### Telegram Sharing Issues
1. **Verify bot token** is correct and active
2. **Check chat ID** is valid and bot is added to group
3. **Ensure message length** is under 4096 characters
4. **Test share URL** in incognito mode

#### Date Parsing Problems
1. **Check timezone** settings in browser
2. **Verify date formats** in debug console
3. **Test with sample data** to isolate issues

### Debug Mode

Enable detailed logging:
1. Click "Debug Console" button
2. Review real-time logs
3. Export logs for analysis
4. Check for specific error patterns

### Sample Data Mode

If all scrapers fail:
1. App automatically switches to sample data
2. Warning banner appears
3. Full functionality remains available
4. "Try Again" button re-attempts live fetch

## Development

### Adding New Event Sources

1. **Create scraper package**:
   ```go
   // pkg/newsource/scraper.go
   package newsource
   
   type Scraper struct {
       client  *resty.Client
       baseURL string
   }
   
   func (s *Scraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
       // Implementation
   }
   ```

2. **Register in main.go**:
   ```go
   newsourceScraper := newsource.NewScraper()
   agg := aggregator.NewAggregator(meetupScraper, eventbriteScraper, devEventsScraper, newsourceScraper)
   ```

3. **Update frontend**:
   ```javascript
   // Add to source filter options
   <option value="newsource">New Source</option>
   ```

### Customizing Event Display

Modify `renderEventCard()` in `web/app.js`:
```javascript
renderEventCard(event) {
    // Customize event card HTML
    return `<div class="event-card">...</div>`;
}
```

### Adding New Filters

1. **Add HTML control**:
   ```html
   <select id="newFilter">
       <option value="all">All</option>
       <!-- options -->
   </select>
   ```

2. **Update JavaScript**:
   ```javascript
   document.getElementById('newFilter').addEventListener('change', () => {
       this.applyFilters();
   });
   ```

## Deployment

### Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o main cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./main"]
```

### Environment Variables for Production

```bash
export PORT=8080
export PERIOD_DAYS=30
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For issues and questions:
1. Check the debug console for error details
2. Review this README for common solutions
3. Open an issue with detailed error logs
4. Include browser console output and network requests

## Changelog

### v1.0.0 (Current)
- Initial release with Meetup, Eventbrite, and Dev.events support
- Telegram integration with Bot API and share URLs
- Robust error handling with sample data fallback
- Modern responsive web UI with dark mode
- Real-time debug console and comprehensive logging
- Smart date parsing and event grouping
- Production-ready with graceful degradation
