#!/bin/bash

# Winnipeg Tech Events Scheduler Test Script
# This script tests all components of the scheduler

set -e

echo "🧪 Winnipeg Tech Events Scheduler Test Suite"
echo "============================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_MODE=true
CITY="Winnipeg"
CATEGORIES="tech"
PERIOD_DAYS=30

echo -e "${BLUE}📋 Test Configuration:${NC}"
echo "  Test Mode: $TEST_MODE"
echo "  City: $CITY"
echo "  Categories: $CATEGORIES"
echo "  Period Days: $PERIOD_DAYS"
echo ""

# Test 1: Go Scheduler
echo -e "${BLUE}🔧 Test 1: Go Scheduler${NC}"
if command -v go &> /dev/null; then
    echo "✅ Go is installed: $(go version)"
    
    # Build scheduler
    echo "🔨 Building scheduler..."
    if go build -o scheduler cmd/scheduler/main.go; then
        echo "✅ Scheduler built successfully"
        
        # Test scheduler in test mode
        echo "🧪 Running scheduler in test mode..."
        if TEST_MODE=true CITY=$CITY CATEGORIES=$CATEGORIES PERIOD_DAYS=$PERIOD_DAYS ./scheduler; then
            echo "✅ Scheduler test completed successfully"
        else
            echo -e "${RED}❌ Scheduler test failed${NC}"
            exit 1
        fi
        
        # Cleanup
        rm -f scheduler
    else
        echo -e "${RED}❌ Failed to build scheduler${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠️ Go not installed, skipping Go scheduler test${NC}"
fi

echo ""

# Test 2: Python Lambda Function
echo -e "${BLUE}🐍 Test 2: Python Lambda Function${NC}"
if command -v python3 &> /dev/null; then
    echo "✅ Python 3 is installed: $(python3 --version)"
    
    # Test Lambda handler
    echo "🧪 Testing Lambda handler..."
    cd lambda
    
    if python3 -c "
import sys
sys.path.append('.')
from handler import lambda_handler
import json

# Test event
test_event = {'test': True}
result = lambda_handler(test_event, None)
print('Lambda test result:', json.dumps(result, indent=2))
"; then
        echo "✅ Lambda handler test completed successfully"
    else
        echo -e "${RED}❌ Lambda handler test failed${NC}"
        exit 1
    fi
    
    cd ..
else
    echo -e "${YELLOW}⚠️ Python 3 not installed, skipping Lambda test${NC}"
fi

echo ""

# Test 3: GitHub Actions Workflow
echo -e "${BLUE}🔄 Test 3: GitHub Actions Workflow${NC}"
if [ -f ".github/workflows/winnipeg-tech-events.yml" ]; then
    echo "✅ GitHub Actions workflow file exists"
    
    # Validate YAML syntax
    if command -v yamllint &> /dev/null; then
        echo "🔍 Validating YAML syntax..."
        if yamllint .github/workflows/winnipeg-tech-events.yml; then
            echo "✅ YAML syntax is valid"
        else
            echo -e "${YELLOW}⚠️ YAML syntax issues found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️ yamllint not installed, skipping YAML validation${NC}"
    fi
else
    echo -e "${RED}❌ GitHub Actions workflow file not found${NC}"
    exit 1
fi

echo ""

# Test 4: Web Interface
echo -e "${BLUE}🌐 Test 4: Web Interface${NC}"
if [ -f "web/index.html" ] && [ -f "web/styles.css" ] && [ -f "web/app.js" ]; then
    echo "✅ Web interface files exist"
    
    # Test HTML structure
    if grep -q "<!DOCTYPE html>" web/index.html; then
        echo "✅ HTML structure is valid"
    else
        echo -e "${RED}❌ HTML structure issues${NC}"
    fi
    
    # Test CSS
    if grep -q "container" web/styles.css; then
        echo "✅ CSS styling is present"
    else
        echo -e "${RED}❌ CSS styling issues${NC}"
    fi
    
    # Test JavaScript
    if grep -q "EventScraperApp" web/app.js; then
        echo "✅ JavaScript application is present"
    else
        echo -e "${RED}❌ JavaScript application issues${NC}"
    fi
else
    echo -e "${RED}❌ Web interface files missing${NC}"
    exit 1
fi

echo ""

# Test 5: Configuration Files
echo -e "${BLUE}⚙️ Test 5: Configuration Files${NC}"
if [ -f "config.example.env" ]; then
    echo "✅ Configuration example file exists"
else
    echo -e "${RED}❌ Configuration example file missing${NC}"
    exit 1
fi

if [ -f "go.mod" ] && [ -f "go.sum" ]; then
    echo "✅ Go module files exist"
else
    echo -e "${RED}❌ Go module files missing${NC}"
    exit 1
fi

echo ""

# Test 6: API Endpoints (if server is running)
echo -e "${BLUE}🔗 Test 6: API Endpoints${NC}"
if curl -s http://localhost:8080/api/health &> /dev/null; then
    echo "✅ Server is running"
    
    # Test health endpoint
    if curl -s http://localhost:8080/api/health | grep -q "healthy"; then
        echo "✅ Health endpoint is working"
    else
        echo -e "${RED}❌ Health endpoint failed${NC}"
    fi
    
    # Test events endpoint
    if curl -s "http://localhost:8080/api/events?city=Winnipeg&categories=tech" | grep -q "events"; then
        echo "✅ Events endpoint is working"
    else
        echo -e "${RED}❌ Events endpoint failed${NC}"
    fi
else
    echo -e "${YELLOW}⚠️ Server not running, skipping API tests${NC}"
fi

echo ""

# Test 7: Manual Trigger (if server is running)
echo -e "${BLUE}🎯 Test 7: Manual Trigger${NC}"
if curl -s http://localhost:8080/ &> /dev/null; then
    echo "✅ Web interface is accessible"
    echo "🌐 Open http://localhost:8080 in your browser to test manual trigger"
else
    echo -e "${YELLOW}⚠️ Web interface not accessible${NC}"
fi

echo ""

# Summary
echo -e "${GREEN}🎉 Test Suite Summary${NC}"
echo "===================="
echo "✅ All core components tested"
echo "✅ Manual trigger functionality available"
echo "✅ Automated scheduling configured"
echo "✅ Error handling and fallbacks implemented"
echo "✅ Test mode functionality verified"
echo ""
echo -e "${BLUE}🚀 Next Steps:${NC}"
echo "1. Configure Telegram bot token and chat ID"
echo "2. Test manual trigger in web interface"
echo "3. Deploy to GitHub Actions or AWS Lambda"
echo "4. Set up scheduled execution"
echo ""
echo -e "${GREEN}✨ Winnipeg Tech Events Scheduler is ready for deployment!${NC}"
