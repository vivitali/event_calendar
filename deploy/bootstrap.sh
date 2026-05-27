#!/usr/bin/env bash
#
# deploy/bootstrap.sh — one-time AWS setup for the annual-events scrape.
#
# Run this ONCE by an operator whose credentials may modify IAM (the CI deploy
# role intentionally cannot). It is idempotent: safe to re-run.
#
# It does two things deploy/deploy.sh cannot:
#   1. Attaches an inline SSM read/write policy to the Lambda execution role.
#   2. Creates the SSM parameter (empty list) if it does not yet exist.
#
# Requirements:
#   - AWS CLI v2 configured with IAM + SSM permissions
#
# Usage:
#   ./deploy/bootstrap.sh
set -euo pipefail

REGION="${AWS_REGION:-us-east-1}"
ROLE_NAME="winnipeg-tech-events-lambda-role"
SSM_PARAM="/winnipeg-tech-events/annual-events"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
SSM_PARAM_ARN="arn:aws:ssm:${REGION}:${ACCOUNT_ID}:parameter${SSM_PARAM}"

echo "== Attaching SSM read/write policy to ${ROLE_NAME} =="
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
echo "  -> policy ssm-annual-events attached"

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

echo
echo "== Bootstrap done =="
echo "Now push to deploy the Lambda code + EventBridge rule, then trigger the first scrape:"
echo "  aws lambda invoke --function-name winnipeg-tech-events \\"
echo "      --payload '{\"action\":\"scrape_annual\"}' \\"
echo "      --cli-binary-format raw-in-base64-out /tmp/out.json && cat /tmp/out.json"
