#!/bin/bash

# Test script for the monthly meetup poll functionality
# This script tests the poll scheduler in test mode

echo "🧪 Testing Monthly Meetup Poll Functionality"
echo "============================================="

# Set test environment variables
export TEST_MODE=true
export TELEGRAM_POLL_BOT_TOKEN="test_poll_token"
export TELEGRAM_POLL_CHAT_ID="test_poll_chat_id"

echo "📋 Environment variables set:"
echo "  TEST_MODE=$TEST_MODE"
echo "  TELEGRAM_POLL_BOT_TOKEN=$TELEGRAM_POLL_BOT_TOKEN"
echo "  TELEGRAM_POLL_CHAT_ID=$TELEGRAM_POLL_CHAT_ID"
echo ""

echo "🔧 Building poll scheduler..."
go build -o poll-scheduler ./cmd/poll-scheduler/

if [ $? -ne 0 ]; then
    echo "❌ Failed to build poll scheduler"
    exit 1
fi

echo "✅ Poll scheduler built successfully"
echo ""

echo "🚀 Running poll scheduler in test mode..."
./poll-scheduler

if [ $? -eq 0 ]; then
    echo "✅ Poll scheduler test completed successfully"
else
    echo "❌ Poll scheduler test failed"
    exit 1
fi

echo ""
echo "🧹 Cleaning up..."
rm -f poll-scheduler

echo "✅ Test completed successfully!"
echo ""
echo "📝 To test with real Telegram:"
echo "  1. Set your real TELEGRAM_POLL_BOT_TOKEN and TELEGRAM_POLL_CHAT_ID"
echo "  2. Set TEST_MODE=false"
echo "  3. Run the poll scheduler on the 20th of any month"
echo "  4. Or modify the is20thOfMonth() function temporarily for testing"
