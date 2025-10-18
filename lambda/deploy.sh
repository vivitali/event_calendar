#!/bin/bash

# Winnipeg Tech Events Lambda Deployment Script
# This script packages and deploys the Lambda function to AWS

set -e

FUNCTION_NAME="winnipeg-tech-events"
REGION="us-east-1"
RUNTIME="python3.11"
HANDLER="handler.lambda_handler"
TIMEOUT=300
MEMORY_SIZE=512

echo "🚀 Deploying Winnipeg Tech Events Lambda Function"
echo "================================================"

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "❌ AWS CLI is not installed. Please install it first."
    echo "   Visit: https://aws.amazon.com/cli/"
    exit 1
fi

# Check if AWS credentials are configured
if ! aws sts get-caller-identity &> /dev/null; then
    echo "❌ AWS credentials not configured. Please run 'aws configure' first."
    exit 1
fi

echo "✅ AWS CLI configured"

# Create deployment package
echo "📦 Creating deployment package..."
mkdir -p dist
cp handler.py dist/
cp requirements.txt dist/

# Install dependencies
echo "📥 Installing Python dependencies..."
cd dist
pip install -r requirements.txt -t .
cd ..

# Create ZIP file
echo "🗜️ Creating ZIP package..."
cd dist
zip -r ../lambda-deployment.zip .
cd ..

echo "✅ Deployment package created: lambda-deployment.zip"

# Check if function exists
if aws lambda get-function --function-name $FUNCTION_NAME --region $REGION &> /dev/null; then
    echo "🔄 Updating existing Lambda function..."
    aws lambda update-function-code \
        --function-name $FUNCTION_NAME \
        --zip-file fileb://lambda-deployment.zip \
        --region $REGION
    
    echo "✅ Lambda function updated successfully"
else
    echo "🆕 Creating new Lambda function..."
    aws lambda create-function \
        --function-name $FUNCTION_NAME \
        --runtime $RUNTIME \
        --role arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):role/lambda-execution-role \
        --handler $HANDLER \
        --zip-file fileb://lambda-deployment.zip \
        --timeout $TIMEOUT \
        --memory-size $MEMORY_SIZE \
        --region $REGION
    
    echo "✅ Lambda function created successfully"
fi

# Set environment variables (if provided)
if [ ! -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo "🔧 Setting environment variables..."
    aws lambda update-function-configuration \
        --function-name $FUNCTION_NAME \
        --environment Variables="{
            TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN,
            TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID,
            CITY=Winnipeg,
            CATEGORIES=tech,
            TEST_MODE=false
        }" \
        --region $REGION
    
    echo "✅ Environment variables set"
fi

# Create EventBridge rule for scheduling (optional)
echo "⏰ Creating EventBridge rule for scheduling..."
RULE_NAME="winnipeg-tech-events-schedule"

# Check if rule exists
if aws events describe-rule --name $RULE_NAME --region $REGION &> /dev/null; then
    echo "🔄 Updating existing EventBridge rule..."
else
    echo "🆕 Creating EventBridge rule..."
    aws events put-rule \
        --name $RULE_NAME \
        --schedule-expression "cron(0 14 * * 1 *)" \
        --description "Trigger Winnipeg Tech Events Lambda every Monday at 9 AM CST" \
        --region $REGION
    
    echo "✅ EventBridge rule created"
fi

# Add Lambda permission for EventBridge
echo "🔐 Adding Lambda permissions..."
aws lambda add-permission \
    --function-name $FUNCTION_NAME \
    --statement-id "allow-eventbridge" \
    --action "lambda:InvokeFunction" \
    --principal events.amazonaws.com \
    --source-arn "arn:aws:events:$REGION:$(aws sts get-caller-identity --query Account --output text):rule/$RULE_NAME" \
    --region $REGION 2>/dev/null || echo "Permission already exists"

# Add EventBridge target
echo "🎯 Adding EventBridge target..."
aws events put-targets \
    --rule $RULE_NAME \
    --targets "Id"="1","Arn"="arn:aws:lambda:$REGION:$(aws sts get-caller-identity --query Account --output text):function:$FUNCTION_NAME" \
    --region $REGION

echo "✅ EventBridge target configured"

# Test the function
echo "🧪 Testing Lambda function..."
aws lambda invoke \
    --function-name $FUNCTION_NAME \
    --payload '{"test": true}' \
    --region $REGION \
    test-output.json

echo "✅ Lambda function tested"

# Cleanup
echo "🧹 Cleaning up..."
rm -rf dist
rm lambda-deployment.zip
rm test-output.json

echo ""
echo "🎉 Deployment completed successfully!"
echo ""
echo "📋 Function Details:"
echo "  Name: $FUNCTION_NAME"
echo "  Region: $REGION"
echo "  Runtime: $RUNTIME"
echo "  Handler: $HANDLER"
echo "  Schedule: Every Monday at 9 AM CST"
echo ""
echo "🔧 Next Steps:"
echo "1. Set environment variables if not already set:"
echo "   aws lambda update-function-configuration --function-name $FUNCTION_NAME --environment Variables='{TELEGRAM_BOT_TOKEN=your_token,TELEGRAM_CHAT_ID=your_chat_id}'"
echo ""
echo "2. Test the function:"
echo "   aws lambda invoke --function-name $FUNCTION_NAME --payload '{}' response.json"
echo ""
echo "3. View logs:"
echo "   aws logs describe-log-groups --log-group-name-prefix /aws/lambda/$FUNCTION_NAME"
echo ""
echo "✨ Your Winnipeg Tech Events Lambda function is ready!"
