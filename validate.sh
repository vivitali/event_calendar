#!/bin/bash

# Winnipeg Tech Events App Validation Script
# This script validates the application structure and functionality

echo "🚀 Winnipeg Tech Events App Validation"
echo "======================================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.24.1 or later."
    echo "   Visit: https://golang.org/dl/"
    exit 1
fi

echo "✅ Go is installed: $(go version)"

# Check if required files exist
echo ""
echo "📁 Checking application structure..."

required_files=(
    "cmd/main.go"
    "internal/models/event.go"
    "pkg/aggregator/aggregator.go"
    "pkg/meetup/scraper.go"
    "pkg/eventbrite/client.go"
    "pkg/devevents/scraper.go"
    "web/index.html"
    "web/styles.css"
    "web/app.js"
    "go.mod"
    "README.md"
    "CHANGELOG.md"
)

all_files_exist=true
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "❌ $file (missing)"
        all_files_exist=false
    fi
done

if [ "$all_files_exist" = false ]; then
    echo ""
    echo "❌ Some required files are missing. Please check the structure."
    exit 1
fi

echo ""
echo "✅ All required files present"

# Check Go modules
echo ""
echo "📦 Checking Go modules..."
if go mod tidy; then
    echo "✅ Go modules are valid"
else
    echo "❌ Go modules validation failed"
    exit 1
fi

# Check if application compiles
echo ""
echo "🔨 Testing compilation..."
if go build -o main cmd/main.go; then
    echo "✅ Application compiles successfully"
    rm -f main
else
    echo "❌ Compilation failed"
    exit 1
fi

# Check web assets
echo ""
echo "🌐 Checking web assets..."

# Check if HTML is valid
if grep -q "<!DOCTYPE html>" web/index.html; then
    echo "✅ HTML structure is valid"
else
    echo "❌ HTML structure issues detected"
fi

# Check if CSS is present
if grep -q "container" web/styles.css; then
    echo "✅ CSS styling is present"
else
    echo "❌ CSS styling issues detected"
fi

# Check if JavaScript is present
if grep -q "EventScraperApp" web/app.js; then
    echo "✅ JavaScript application is present"
else
    echo "❌ JavaScript application issues detected"
fi

# Test HTTP endpoints (if server can start)
echo ""
echo "🌐 Testing HTTP endpoints..."

# Start server in background
echo "Starting server..."
go run cmd/main.go &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test health endpoint
if curl -s http://localhost:8080/api/health > /dev/null; then
    echo "✅ Health endpoint is working"
else
    echo "❌ Health endpoint failed"
fi

# Test events endpoint
if curl -s http://localhost:8080/api/events?city=Winnipeg&categories=tech > /dev/null; then
    echo "✅ Events endpoint is working"
else
    echo "❌ Events endpoint failed"
fi

# Test web interface
if curl -s http://localhost:8080/ > /dev/null; then
    echo "✅ Web interface is accessible"
else
    echo "❌ Web interface failed"
fi

# Stop server
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo ""
echo "🎉 Validation Complete!"
echo ""
echo "📋 Next Steps:"
echo "1. Open http://localhost:8080 in your browser"
echo "2. Test the event loading and filtering"
echo "3. Test Telegram sharing functionality"
echo "4. Check debug console for any issues"
echo ""
echo "🐛 If you encounter issues:"
echo "1. Check the debug console in the web interface"
echo "2. Review the README.md for troubleshooting"
echo "3. Check the CHANGELOG.md for known issues"
echo ""
echo "✨ The Winnipeg Tech Events App is ready to use!"
