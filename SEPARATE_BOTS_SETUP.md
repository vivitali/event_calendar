# Separate Bots Setup Guide

This document explains how to set up and use the separate bot configuration for the event calendar system.

## Overview

The system now supports **2 independent Telegram bots**:

1. **Main Bot** - Handles event notifications and general messages
2. **Poll Bot** - Handles monthly meetup polls only

This separation provides better organization, security, and allows for different permissions and chat configurations.

## Bot Configuration

### Main Bot (Event Notifications)
- **Purpose**: Sends tech event notifications
- **Environment Variable**: `TELEGRAM_BOT_TOKEN`
- **Chat**: `TELEGRAM_CHAT_ID`
- **Schedule**: Every Monday at 9:00 AM CST

### Poll Bot (Monthly Polls)
- **Purpose**: Sends monthly meetup polls
- **Environment Variable**: `TELEGRAM_POLL_BOT_TOKEN`
- **Chat**: `TELEGRAM_POLL_CHAT_ID`
- **Schedule**: 20th of every month at 9:00 AM CST

## Setup Instructions

### 1. Create Two Telegram Bots

#### Main Bot (for events):
1. Message @BotFather on Telegram
2. Send `/newbot`
3. Choose a name: e.g., "Winnipeg Tech Events Bot"
4. Choose a username: e.g., "winnipeg_tech_events_bot"
5. Save the bot token

#### Poll Bot (for polls):
1. Message @BotFather on Telegram
2. Send `/newbot`
3. Choose a name: e.g., "Winnipeg Meetup Poll Bot"
4. Choose a username: e.g., "winnipeg_meetup_poll_bot"
5. Save the bot token

### 2. Configure Bot Permissions

#### Main Bot:
- Add to main chat for event notifications
- Grant permissions to send messages
- Can be used for general community communication

#### Poll Bot:
- Add to poll chat (can be same or different chat)
- Grant permissions to send messages and polls
- Dedicated to poll functionality only

### 3. Get Chat IDs

For each chat where you want to send messages:

1. Add @userinfobot to the chat
2. Send any message
3. The bot will reply with the chat ID
4. Save the chat ID (format: `-1001234567890` for groups, `123456789` for users)

### 4. Environment Variables

Set up these environment variables:

```bash
# Main Bot Configuration
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
TELEGRAM_CHAT_ID=-1001234567890

# Poll Bot Configuration
TELEGRAM_POLL_BOT_TOKEN=987654321:XYZabcDEFghiJKLmnopqrsTUVwxyz
TELEGRAM_POLL_CHAT_ID=-1009876543210

# Optional Configuration
ENABLE_POLL=true
TEST_MODE=false
```

## GitHub Actions Setup

### Repository Secrets

Add these secrets to your GitHub repository:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Add these secrets:

   - `TELEGRAM_BOT_TOKEN` - Main bot token
   - `TELEGRAM_CHAT_ID` - Main chat ID
   - `TELEGRAM_POLL_BOT_TOKEN` - Poll bot token
   - `TELEGRAM_POLL_CHAT_ID` - Poll chat ID

### Workflow Files

The system includes two workflows:

1. **`.github/workflows/winnipeg-tech-events.yml`** - Main events workflow
2. **`.github/workflows/monthly-poll.yml`** - Dedicated poll workflow

Both workflows are configured to use the separate bot tokens.

## Usage Examples

### Local Development

```bash
# Set environment variables
export TELEGRAM_BOT_TOKEN="your_main_bot_token"
export TELEGRAM_CHAT_ID="your_main_chat_id"
export TELEGRAM_POLL_BOT_TOKEN="your_poll_bot_token"
export TELEGRAM_POLL_CHAT_ID="your_poll_chat_id"

# Run main scheduler (includes poll on 20th)
go run cmd/scheduler/main.go

# Run poll scheduler only
go run cmd/poll-scheduler/main.go
```

### Testing

```bash
# Test poll functionality
./test_poll.sh

# Test GitHub Actions simulation
./test_github_actions.sh

# Test with 20th simulation
./test_poll_20th.sh
```

## Benefits of Separate Bots

### 1. **Security**
- Different permissions for different functions
- Isolated access tokens
- Reduced risk if one bot is compromised

### 2. **Organization**
- Clear separation of concerns
- Different bot names and descriptions
- Easier to manage permissions

### 3. **Flexibility**
- Can use different chats for different purposes
- Independent bot configurations
- Easier to disable one function without affecting the other

### 4. **Scalability**
- Can add more specialized bots in the future
- Independent rate limiting
- Better monitoring and logging

## Troubleshooting

### Common Issues

1. **Bot not sending messages**:
   - Check bot permissions in the chat
   - Verify bot is added to the chat
   - Check bot token is correct

2. **Poll not working**:
   - Ensure poll bot has poll permissions
   - Verify poll chat ID is correct
   - Check poll bot token is set

3. **Wrong bot sending messages**:
   - Verify environment variables are set correctly
   - Check which bot token is being used
   - Ensure proper configuration

### Debugging

1. **Check logs**:
   ```bash
   # Run with verbose logging
   go run cmd/poll-scheduler/main.go
   ```

2. **Test connections**:
   ```bash
   # Test main bot
   curl "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getMe"
   
   # Test poll bot
   curl "https://api.telegram.org/bot$TELEGRAM_POLL_BOT_TOKEN/getMe"
   ```

3. **Verify chat IDs**:
   - Use @userinfobot to get correct chat IDs
   - Ensure bots are added to the correct chats

## Migration from Single Bot

If you're migrating from a single bot setup:

1. **Create the poll bot** using @BotFather
2. **Update environment variables** to include `TELEGRAM_POLL_BOT_TOKEN`
3. **Update GitHub Actions secrets** with the new bot token
4. **Test the configuration** using the provided test scripts
5. **Deploy the changes** and verify both bots work correctly

## Best Practices

1. **Bot Naming**: Use descriptive names for your bots
2. **Permissions**: Grant only necessary permissions to each bot
3. **Monitoring**: Monitor both bots for errors and usage
4. **Backup**: Keep backup copies of bot tokens securely
5. **Documentation**: Document which bot is used for what purpose

## Support

For issues or questions:
1. Check the troubleshooting section
2. Review bot permissions and chat settings
3. Test with the provided scripts
4. Check GitHub Actions logs for errors
