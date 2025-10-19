# Monthly Meetup Poll Integration

This document describes the new Telegram poll integration that posts a monthly meetup poll on the 20th of every month.

## Overview

The system now supports two separate Telegram integrations:

1. **Event Notifications** - Posts tech events to the main chat (existing functionality)
2. **Monthly Meetup Polls** - Posts a poll on the 20th of every month to a different chat asking users to select their preferred meeting day

## Features

- **Automatic Poll Posting**: Posts a poll on the 20th of every month
- **Multiple Choice**: Users can select multiple days of the week
- **Ukrainian Language**: Poll question and options are in Ukrainian
- **Separate Chat**: Uses a different chat ID from the main event notifications
- **Test Mode**: Supports testing without actually sending polls

## Configuration

### Environment Variables

Add these environment variables to your configuration:

```bash
# Main Telegram Bot Configuration (for event notifications)
TELEGRAM_BOT_TOKEN=your_main_bot_token_here
TELEGRAM_CHAT_ID=your_main_chat_id_here

# Poll Bot Configuration (for monthly meetup polls)
TELEGRAM_POLL_BOT_TOKEN=your_poll_bot_token_here
TELEGRAM_POLL_CHAT_ID=your_poll_chat_id_here

# Poll Configuration
ENABLE_POLL=true
```

### Configuration Options

- `TELEGRAM_BOT_TOKEN`: The main bot token for event notifications
- `TELEGRAM_CHAT_ID`: The main chat ID for event notifications
- `TELEGRAM_POLL_BOT_TOKEN`: The separate bot token for polls
- `TELEGRAM_POLL_CHAT_ID`: The chat ID where polls should be posted
- `ENABLE_POLL`: Set to `false` to disable poll functionality (default: `true` if both poll bot token and chat ID are configured)
- `TEST_MODE`: Set to `true` to test without actually sending polls

## Usage

### Option 1: Integrated with Main Scheduler

The poll functionality is integrated into the main scheduler (`cmd/scheduler/main.go`). When the scheduler runs on the 20th of the month, it will automatically send the poll if:

- `ENABLE_POLL` is `true` (or not set and both poll bot token and chat ID are configured)
- `TELEGRAM_POLL_BOT_TOKEN` is configured
- `TELEGRAM_POLL_CHAT_ID` is configured
- It's the 20th of the month

### Option 2: Standalone Poll Scheduler

Use the dedicated poll scheduler (`cmd/poll-scheduler/main.go`) for more control:

```bash
# Build the poll scheduler
go build -o poll-scheduler ./cmd/poll-scheduler/

# Run the poll scheduler
./poll-scheduler
```

## Poll Content

The poll asks: **"Є бажаючі зустрітись - виберіть день тижня"** (Are there people who want to meet - choose the day of the week)

Options:
- Понеділок (Monday)
- Вівторок (Tuesday)
- Середа (Wednesday)
- Четвер (Thursday)
- П'ятниця (Friday)
- Субота (Saturday)
- Неділя (Sunday)

## Testing

### Test Scripts

Two test scripts are provided:

1. **`test_poll.sh`** - Tests the poll scheduler on a regular day (should skip)
2. **`test_poll_20th.sh`** - Tests the poll scheduler simulating the 20th of the month

```bash
# Test on regular day
./test_poll.sh

# Test simulating 20th of month
./test_poll_20th.sh
```

### Manual Testing

To test with real Telegram:

1. Set your real `TELEGRAM_POLL_BOT_TOKEN` and `TELEGRAM_POLL_CHAT_ID`
2. Set `TEST_MODE=false`
3. Temporarily modify the `is20thOfMonth()` function to return `true`
4. Run the poll scheduler

## API Reference

### Telegram Service Methods

#### `SendPoll(chatID, question string, options []string, allowMultiple bool) error`

Sends a poll to the specified chat.

**Parameters:**
- `chatID`: The chat ID to send the poll to
- `question`: The poll question
- `options`: Array of poll options
- `allowMultiple`: Whether users can select multiple options

#### `SendMonthlyMeetupPoll(chatID string) error`

Sends the predefined monthly meetup poll.

**Parameters:**
- `chatID`: The chat ID to send the poll to

## Scheduling

### Cron Job Setup

To run the poll scheduler automatically, set up a cron job:

```bash
# Run daily at 9:00 AM, but only sends poll on the 20th
0 9 * * * /path/to/your/poll-scheduler

# Or run only on the 20th of each month at 9:00 AM
0 9 20 * * /path/to/your/poll-scheduler
```

### GitHub Actions

Add to your GitHub Actions workflow:

```yaml
- name: Run Monthly Poll Scheduler
  run: |
    go build -o poll-scheduler ./cmd/poll-scheduler/
    ./poll-scheduler
  env:
    TELEGRAM_BOT_TOKEN: ${{ secrets.TELEGRAM_BOT_TOKEN }}
    TELEGRAM_POLL_CHAT_ID: ${{ secrets.TELEGRAM_POLL_CHAT_ID }}
    TEST_MODE: false
```

## Troubleshooting

### Common Issues

1. **Poll not sending**: Check that `TELEGRAM_POLL_CHAT_ID` is configured and `ENABLE_POLL` is not set to `false`
2. **Wrong date**: The poll only sends on the 20th of the month
3. **Test mode**: Make sure `TEST_MODE` is set to `false` for production

### Logs

The scheduler provides detailed logging:

```
📊 Monthly Meetup Poll Scheduler Starting...
📋 Poll Configuration loaded: TestMode=false
📝 Today is the 20th of the month, proceeding with poll
📤 Sending monthly meetup poll to Telegram...
✅ Monthly meetup poll sent successfully
```

## Security

- Bot tokens should be stored as environment variables or secrets
- Never commit bot tokens to version control
- Use test mode during development

## Future Enhancements

Potential improvements:

1. **Configurable poll date**: Allow setting a different day of the month
2. **Custom poll questions**: Support for different poll types
3. **Poll results tracking**: Store and analyze poll results
4. **Multiple poll types**: Support for different types of polls
5. **Poll scheduling**: Allow scheduling polls for specific dates
