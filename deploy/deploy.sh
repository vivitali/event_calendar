#!/usr/bin/env bash
#
# deploy/deploy.sh — Build and deploy the Winnipeg Tech Events Lambda.
#
# Requirements:
#   - AWS CLI v2 configured (e.g. `aws login` or `aws configure sso`)
#   - Go 1.24+ installed
#
# Environment (read by Lambda; pass via shell or `deploy/.env`):
#   TELEGRAM_BOT_TOKEN
#   TELEGRAM_CHAT_ID
#   TELEGRAM_POLL_BOT_TOKEN   (optional; falls back to TELEGRAM_BOT_TOKEN)
#   TELEGRAM_POLL_CHAT_ID     (optional; falls back to TELEGRAM_CHAT_ID)
#   CITY=Winnipeg (default)
#   CATEGORIES=tech (default)
#   PERIOD_DAYS=30 (default)
#   TEST_MODE=false (default)
#   ANNUAL_SSM_PARAM=/winnipeg-tech-events/annual-events (default; set by this script)
set -euo pipefail

FUNCTION_NAME="winnipeg-tech-events"
REGION="${AWS_REGION:-us-east-1}"
ROLE_NAME="winnipeg-tech-events-lambda-role"
RULE_EVENTS="winnipeg-events-weekly"
RULE_POLL="winnipeg-poll-monthly"
RULE_ANNUAL="winnipeg-annual-scrape-monthly"
SSM_PARAM="/winnipeg-tech-events/annual-events"
RUNTIME="provided.al2023"
HANDLER="bootstrap"
TIMEOUT=300
MEMORY=512
LOG_RETENTION_DAYS=14

cd "$(dirname "$0")/.."

# Load deploy/.env if present (gitignored)
if [[ -f deploy/.env ]]; then
    set -a
    # shellcheck disable=SC1091
    source deploy/.env
    set +a
fi

echo "== Building Go Lambda =="
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -tags lambda.norpc -o bootstrap ./cmd/lambda
zip -j deploy/lambda.zip bootstrap >/dev/null
rm bootstrap
echo "  -> deploy/lambda.zip"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

echo "== Ensuring IAM role ${ROLE_NAME} =="
if ! aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
    aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document '{
            "Version": "2012-10-17",
            "Statement": [{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]
        }' >/dev/null
    aws iam attach-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
    echo "  -> role created; waiting 10s for IAM propagation"
    sleep 10
fi

echo "== Attaching SSM read/write policy to ${ROLE_NAME} =="
SSM_PARAM_ARN="arn:aws:ssm:${REGION}:${ACCOUNT_ID}:parameter${SSM_PARAM}"
aws iam put-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-name "ssm-annual-events" \
    --policy-document "{
        \"Version\": \"2012-10-17\",
        \"Statement\": [{
            \"Effect\": \"Allow\",
            \"Action\": [\"ssm:GetParameter\", \"ssm:PutParameter\"],
            \"Resource\": \"${SSM_PARAM_ARN}\"
        }]
    }" >/dev/null

echo "== Ensuring SSM parameter ${SSM_PARAM} =="
if ! aws ssm get-parameter --name "$SSM_PARAM" --region "$REGION" >/dev/null 2>&1; then
    aws ssm put-parameter \
        --name "$SSM_PARAM" \
        --type String \
        --value '[]' \
        --region "$REGION" >/dev/null
    echo "  -> created (empty list)"
else
    echo "  -> exists; leaving value untouched"
fi

echo "== Deploying Lambda function =="
if aws lambda get-function --function-name "$FUNCTION_NAME" --region "$REGION" >/dev/null 2>&1; then
    aws lambda update-function-code \
        --function-name "$FUNCTION_NAME" \
        --zip-file fileb://deploy/lambda.zip \
        --region "$REGION" >/dev/null
    aws lambda wait function-updated --function-name "$FUNCTION_NAME" --region "$REGION"
    aws lambda update-function-configuration \
        --function-name "$FUNCTION_NAME" \
        --runtime "$RUNTIME" \
        --handler "$HANDLER" \
        --timeout "$TIMEOUT" \
        --memory-size "$MEMORY" \
        --region "$REGION" >/dev/null
    echo "  -> updated"
else
    aws lambda create-function \
        --function-name "$FUNCTION_NAME" \
        --runtime "$RUNTIME" \
        --role "$ROLE_ARN" \
        --handler "$HANDLER" \
        --zip-file fileb://deploy/lambda.zip \
        --timeout "$TIMEOUT" \
        --memory-size "$MEMORY" \
        --region "$REGION" >/dev/null
    echo "  -> created"
fi

echo "== Setting Lambda environment variables =="
aws lambda wait function-updated --function-name "$FUNCTION_NAME" --region "$REGION"
aws lambda update-function-configuration \
    --function-name "$FUNCTION_NAME" \
    --region "$REGION" \
    --environment "Variables={
        TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN:-},
        TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID:-},
        TELEGRAM_POLL_BOT_TOKEN=${TELEGRAM_POLL_BOT_TOKEN:-},
        TELEGRAM_POLL_CHAT_ID=${TELEGRAM_POLL_CHAT_ID:-},
        CITY=${CITY:-Winnipeg},
        CATEGORIES=${CATEGORIES:-tech},
        PERIOD_DAYS=${PERIOD_DAYS:-30},
        TEST_MODE=${TEST_MODE:-false},
        ANNUAL_SSM_PARAM=${SSM_PARAM}
    }" >/dev/null

echo "== Setting CloudWatch log retention to ${LOG_RETENTION_DAYS} days =="
aws logs put-retention-policy \
    --log-group-name "/aws/lambda/${FUNCTION_NAME}" \
    --retention-in-days "$LOG_RETENTION_DAYS" \
    --region "$REGION" 2>/dev/null || true

create_or_update_rule() {
    local rule_name="$1"
    local schedule="$2"
    local input="$3"
    local description="$4"

    echo "== Ensuring EventBridge rule ${rule_name} =="
    aws events put-rule \
        --name "$rule_name" \
        --schedule-expression "$schedule" \
        --description "$description" \
        --region "$REGION" >/dev/null

    aws lambda add-permission \
        --function-name "$FUNCTION_NAME" \
        --statement-id "allow-${rule_name}" \
        --action lambda:InvokeFunction \
        --principal events.amazonaws.com \
        --source-arn "arn:aws:events:${REGION}:${ACCOUNT_ID}:rule/${rule_name}" \
        --region "$REGION" 2>/dev/null || true

    aws events put-targets \
        --rule "$rule_name" \
        --targets "Id=1,Arn=arn:aws:lambda:${REGION}:${ACCOUNT_ID}:function:${FUNCTION_NAME},Input='${input}'" \
        --region "$REGION" >/dev/null
}

# Mondays 14:00 UTC (9 AM CST)
create_or_update_rule "$RULE_EVENTS" \
    "cron(0 14 ? * MON *)" \
    '{"action":"events"}' \
    "Weekly Winnipeg tech events digest"

# 20th of each month, 14:00 UTC
create_or_update_rule "$RULE_POLL" \
    "cron(0 14 20 * ? *)" \
    '{"action":"poll"}' \
    "Monthly meetup day-of-week poll"

# 1st of each month, 13:00 UTC — refreshes annual events ahead of the
# weekly digest (Mondays 14:00 UTC).
create_or_update_rule "$RULE_ANNUAL" \
    "cron(0 13 1 * ? *)" \
    '{"action":"scrape_annual"}' \
    "Monthly refresh of curated annual events"

echo
echo "== Done =="
echo
echo "Manual invoke:"
echo "  aws lambda invoke --function-name ${FUNCTION_NAME} \\"
echo "      --payload '{\"action\":\"events\"}' --cli-binary-format raw-in-base64-out out.json && cat out.json"
echo
echo "Logs:"
echo "  aws logs tail /aws/lambda/${FUNCTION_NAME} --follow --region ${REGION}"
