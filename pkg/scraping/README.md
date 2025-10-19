# Scraping Service

A comprehensive, modular scraping service for collecting events from multiple sources.

## Architecture

The scraping service is built with a clean, extensible architecture:

- **Service Layer**: `ScrapingService` manages multiple scrapers
- **Base Scraper**: `BaseScraper` provides common functionality
- **Individual Scrapers**: Specific implementations for each event source
- **Factory Pattern**: Easy creation and configuration of services

## Components

### Core Components

1. **ScrapingService** (`service.go`)
   - Manages multiple event scrapers
   - Provides concurrent scraping capabilities
   - Handles health monitoring and error management

2. **BaseScraper** (`base.go`)
   - Common functionality for all scrapers
   - HTTP client configuration
   - Health checking and validation
   - Event filtering and deduplication

3. **Individual Scrapers**
   - **MeetupScraper** (`meetup.go`): Scrapes events from Meetup.com
   - **EventbriteScraper** (`eventbrite.go`): Scrapes events from Eventbrite.com

4. **Factory** (`factory.go`)
   - Creates pre-configured scraping services
   - Supports different scraper combinations

## Usage

### Basic Usage

```go
// Create a service with all default scrapers
factory := scraping.NewScrapingServiceFactory()
service := factory.CreateDefaultService()

// Scrape events from all sources
events, err := service.ScrapeEvents("Winnipeg", "tech", 30*24*time.Hour)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d events\n", len(events))
```

### Custom Configuration

```go
// Create service with specific scrapers
service := factory.CreateServiceWithScrapers([]string{"meetup", "eventbrite"})

// Or create service with only one scraper
meetupService := factory.CreateMeetupOnlyService()
```

### Individual Scraper Usage

```go
// Scrape from a specific source
events, err := service.ScrapeEventsFromSource("meetup", "Winnipeg", "tech", 7*24*time.Hour)
if err != nil {
    log.Printf("Error: %v", err)
}
```

### Health Monitoring

```go
// Check health status of all scrapers
healthStatus := service.GetHealthStatus()
for name, healthy := range healthStatus {
    fmt.Printf("Scraper %s: %v\n", name, healthy)
}

// Get list of registered scrapers
scrapers := service.GetRegisteredScrapers()
fmt.Printf("Registered scrapers: %v\n", scrapers)
```

## API Endpoints

The service integrates with the main application and provides these endpoints:

- `GET /api/events` - Scrape events from all sources
- `GET /api/scrapers/health` - Get health status of all scrapers
- `GET /api/scrapers` - Get information about registered scrapers

## Adding New Scrapers

To add a new scraper:

1. Create a new scraper struct that embeds `BaseScraper`
2. Implement the `EventScraper` interface:
   ```go
   type EventScraper interface {
       GetEvents(city, category string, period time.Duration) ([]models.Event, error)
       GetName() string
       IsHealthy() bool
   }
   ```
3. Register the scraper in the factory

Example:

```go
type NewSourceScraper struct {
    *BaseScraper
}

func NewNewSourceScraper() *NewSourceScraper {
    base := NewBaseScraper("newsource", "https://api.newsource.com")
    return &NewSourceScraper{BaseScraper: base}
}

func (n *NewSourceScraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
    // Implementation here
    return events, nil
}
```

## Features

### Concurrent Scraping
- All scrapers run concurrently for better performance
- Graceful error handling - if one scraper fails, others continue

### Health Monitoring
- Automatic health checks for all scrapers
- Configurable health check intervals
- Health status available via API

### Error Handling
- Robust error handling with fallback mechanisms
- Detailed logging for debugging
- Graceful degradation when scrapers fail

### Event Validation
- Built-in event validation
- Duplicate removal
- Time period filtering

### Extensibility
- Easy to add new scrapers
- Pluggable architecture
- Factory pattern for easy configuration

## Configuration

### HTTP Client Configuration
- Realistic browser headers to avoid detection
- Configurable timeouts
- Automatic retry logic

### Anti-Scraping Measures
- User-Agent rotation
- Request rate limiting
- Respectful scraping practices

## Testing

The service includes comprehensive testing capabilities:

```go
// Test individual scrapers
meetupScraper := NewMeetupScraper()
events, err := meetupScraper.GetEvents("Winnipeg", "tech", 7*24*time.Hour)

// Test service integration
service := factory.CreateDefaultService()
events, err := service.ScrapeEvents("Winnipeg", "tech", 30*24*time.Hour)
```

## Error Handling

The service implements multiple layers of error handling:

1. **Individual Scraper Errors**: Each scraper handles its own errors
2. **Service Level Errors**: Service continues even if some scrapers fail
3. **Fallback Mechanisms**: Sample data returned when scraping fails
4. **Health Monitoring**: Automatic detection of unhealthy scrapers

## Performance

- **Concurrent Execution**: All scrapers run in parallel
- **Efficient Parsing**: Optimized HTML parsing with goquery
- **Memory Management**: Proper cleanup and resource management
- **Caching**: Health check results cached to avoid excessive requests

## Security

- **Rate Limiting**: Respectful request patterns
- **User-Agent Rotation**: Realistic browser headers
- **Error Sanitization**: Sensitive information not exposed in logs
- **Input Validation**: All inputs validated before processing
