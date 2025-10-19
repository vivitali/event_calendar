# GitHub Actions Setup for Monthly Poll

This document explains how to set up GitHub Actions to automatically send the monthly meetup poll on the 20th of every month.

## Overview

The GitHub Actions workflow will:
- Run automatically on the 20th of every month at 9:00 AM CST (2:00 PM UTC)
- Send a poll to the specified Telegram chat asking users to select their preferred meeting days
- Provide notifications about success/failure
- Support manual triggering for testing

## Setup Instructions

### 1. Repository Secrets

Add the following secrets to your GitHub repository:

1. Go to your repository on GitHub
2. Click on **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret** and add:

   - **Name**: `TELEGRAM_BOT_TOKEN`
   - **Value**: Your main Telegram bot token (for event notifications)

   - **Name**: `TELEGRAM_CHAT_ID`
   - **Value**: The main chat ID for event notifications

   - **Name**: `TELEGRAM_POLL_BOT_TOKEN`
   - **Value**: Your separate Telegram bot token (for polls)

   - **Name**: `TELEGRAM_POLL_CHAT_ID`
   - **Value**: The chat ID where polls should be sent

### 2. Workflow Files

The following workflow files are already created:

- `.github/workflows/monthly-poll.yml` - Dedicated monthly poll workflow
- `.github/workflows/winnipeg-tech-events.yml` - Updated main workflow with poll support

### 3. Workflow Configuration

#### Monthly Poll Workflow (`.github/workflows/monthly-poll.yml`)

**Schedule**: Runs on the 20th of every month at 9:00 AM CST
```yaml
schedule:
  - cron: '0 14 20 * *'
```

**Features**:
- Automatic execution on the 20th
- Manual triggering with options
- Test mode support
- Force send option for testing
- Success/failure notifications

#### Main Workflow (`.github/workflows/winnipeg-tech-events.yml`)

**Schedule**: Runs every Monday at 9:00 AM CST
```yaml
schedule:
  - cron: '0 14 * * 1'
```

**Features**:
- Posts tech events
- Includes poll functionality when it's the 20th
- Manual triggering support

## Usage

### Automatic Execution

The workflow will automatically run on the 20th of every month. No action required.

### Manual Triggering

1. Go to your repository on GitHub
2. Click on **Actions** tab
3. Select **Monthly Meetup Poll Scheduler**
4. Click **Run workflow**
5. Choose options:
   - **Test mode**: Run without actually sending the poll
   - **Force send**: Send poll even if not the 20th (for testing)

### Testing

#### Local Testing

Run the test script to verify everything works:

```bash
./test_github_actions.sh
```

#### GitHub Actions Testing

1. Trigger the workflow manually with **Test mode** enabled
2. Check the logs to ensure everything runs correctly
3. Verify the poll would be sent (but not actually sent in test mode)

## Workflow Details

### Monthly Poll Workflow Steps

1. **Checkout code** - Gets the latest code
2. **Set up Go** - Installs Go 1.25
3. **Cache Go modules** - Speeds up builds
4. **Install dependencies** - Downloads Go modules
5. **Build poll scheduler** - Compiles the poll scheduler
6. **Test connection** - Verifies Telegram connection (if not test mode)
7. **Run monthly poll scheduler** - Executes the poll logic
8. **Upload logs** - Saves logs as artifacts
9. **Send notifications** - Notifies about success/failure

### Environment Variables

The workflow uses these environment variables:

- `TELEGRAM_BOT_TOKEN` - Main bot token from repository secrets
- `TELEGRAM_CHAT_ID` - Main chat ID from repository secrets
- `TELEGRAM_POLL_BOT_TOKEN` - Poll bot token from repository secrets
- `TELEGRAM_POLL_CHAT_ID` - Poll chat ID from repository secrets
- `TEST_MODE` - Set based on manual trigger options
- `GO_VERSION` - Set to '1.25'

### Notifications

#### Success Notification
Sent to the main chat when poll is sent successfully:
```
✅ Monthly Meetup Poll

Successfully sent monthly meetup poll!

Run ID: 123456789
Time: 2025-01-20T14:00:00Z
```

#### Failure Alert
Sent to the main chat when the workflow fails:
```
🚨 Monthly Poll Alert

Poll scheduler run failed!

Run ID: 123456789
Workflow: Monthly Meetup Poll Scheduler
Time: 2025-01-20T14:00:00Z

Check GitHub Actions logs for details.
```

#### Poll Confirmation
Sent to the poll chat when poll is sent:
```
📊 Monthly Meetup Poll

Poll has been sent successfully!

Date: 2025-01-20T14:00:00Z

Please vote for your preferred meeting days! 🗳️
```

## Troubleshooting

### Common Issues

1. **Workflow not running on 20th**:
   - Check the cron schedule: `0 14 20 * *`
   - Verify the date is correct (20th of the month)
   - Check GitHub Actions logs

2. **Poll not sending**:
   - Verify `TELEGRAM_POLL_CHAT_ID` secret is set
   - Check bot permissions in the poll chat
   - Ensure bot is added to the poll chat

3. **Test mode not working**:
   - Verify `TEST_MODE` environment variable
   - Check workflow logs for test mode confirmation

4. **Build failures**:
   - Check Go version compatibility
   - Verify all dependencies are available
   - Check for syntax errors in code

### Debugging

1. **Check workflow logs**:
   - Go to Actions tab
   - Click on the failed workflow run
   - Review step logs for errors

2. **Test locally**:
   ```bash
   ./test_github_actions.sh
   ```

3. **Verify secrets**:
   - Ensure all required secrets are set
   - Check secret names match exactly

4. **Check bot permissions**:
   - Verify bot can send messages to poll chat
   - Ensure bot is not blocked

## Security Considerations

1. **Secrets Management**:
   - Never commit bot tokens to code
   - Use GitHub repository secrets
   - Rotate tokens regularly

2. **Access Control**:
   - Limit who can trigger workflows
   - Use branch protection rules
   - Monitor workflow runs

3. **Rate Limiting**:
   - Be aware of Telegram API rate limits
   - Implement proper error handling
   - Use exponential backoff for retries

## Monitoring

### Workflow Status

Monitor workflow runs in the GitHub Actions tab:
- Green checkmark: Success
- Red X: Failure
- Yellow circle: In progress

### Logs

Logs are automatically saved as artifacts for 30 days:
- Download logs from the Actions tab
- Review for debugging information
- Check for error patterns

### Notifications

Set up notifications for:
- Workflow failures
- Successful poll sends
- Manual triggers

## Customization

### Changing Schedule

To change when the poll is sent, modify the cron expression:

```yaml
schedule:
  # Run on 15th of every month at 10:00 AM CST
  - cron: '0 15 15 * *'
  
  # Run on 1st and 15th of every month
  - cron: '0 14 1,15 * *'
```

### Different Poll Content

To change the poll question or options, modify the `SendMonthlyMeetupPoll` function in `pkg/telegram/service.go`.

### Additional Notifications

Add more notification steps to the workflow as needed.

## Support

For issues or questions:
1. Check the troubleshooting section
2. Review GitHub Actions logs
3. Test locally with the provided scripts
4. Check Telegram bot permissions and chat settings
