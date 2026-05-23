#!/usr/bin/env bash
#
# deploy/setup-gha.sh — One-time AWS setup for GitHub Actions OIDC deploy.
#
# Idempotent. Safe to re-run.
#
# Requires: AWS CLI v2 already authenticated (aws login / aws configure sso).
# Infers REPO from "origin" if not passed.
#
# Usage:
#   ./deploy/setup-gha.sh                       # infer repo from git remote
#   REPO=owner/name ./deploy/setup-gha.sh       # explicit override
set -euo pipefail

ROLE_NAME="github-deploy-winnipeg-tech-events"
REGION="${AWS_REGION:-us-east-1}"
OIDC_URL="token.actions.githubusercontent.com"
OIDC_THUMBPRINT="6938fd4d98bab03faadb97b34396831e3780aea1"

REPO="${REPO:-}"
if [[ -z "$REPO" ]]; then
    origin=$(git -C "$(dirname "$0")/.." remote get-url origin 2>/dev/null || true)
    # Match git@github.com:owner/repo(.git) or https://github.com/owner/repo(.git)
    if [[ "$origin" =~ github\.com[:/]([^/]+)/([^/.]+)(\.git)?$ ]]; then
        REPO="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
    else
        echo "Could not infer repo from origin: $origin" >&2
        echo "Pass REPO=owner/name explicitly." >&2
        exit 1
    fi
fi

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
OIDC_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/${OIDC_URL}"
ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

echo "== AWS account: ${ACCOUNT_ID}"
echo "== GitHub repo: ${REPO}"
echo "== Role:        ${ROLE_NAME}"
echo

echo "== Ensuring OIDC provider =="
if aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$OIDC_ARN" >/dev/null 2>&1; then
    echo "  already exists"
else
    aws iam create-open-id-connect-provider \
        --url "https://${OIDC_URL}" \
        --client-id-list "sts.amazonaws.com" \
        --thumbprint-list "$OIDC_THUMBPRINT" >/dev/null
    echo "  created"
fi

trust_doc=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "Federated": "${OIDC_ARN}" },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": { "${OIDC_URL}:aud": "sts.amazonaws.com" },
      "StringLike":   { "${OIDC_URL}:sub": "repo:${REPO}:ref:refs/heads/main" }
    }
  }]
}
EOF
)

echo "== Ensuring IAM role =="
if aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
    aws iam update-assume-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-document "$trust_doc" >/dev/null
    echo "  updated trust policy"
else
    aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document "$trust_doc" >/dev/null
    echo "  created"
fi

policy_doc=$(cat <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    { "Effect": "Allow", "Action": ["lambda:*"],          "Resource": "*" },
    { "Effect": "Allow", "Action": ["events:*"],          "Resource": "*" },
    { "Effect": "Allow", "Action": [
        "logs:PutRetentionPolicy","logs:CreateLogGroup","logs:DescribeLogGroups"
    ], "Resource": "*" },
    { "Effect": "Allow", "Action": [
        "iam:GetRole","iam:CreateRole","iam:AttachRolePolicy","iam:PassRole"
    ], "Resource": "*" },
    { "Effect": "Allow", "Action": ["sts:GetCallerIdentity"], "Resource": "*" }
  ]
}
EOF
)

echo "== Attaching inline policy =="
aws iam put-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-name deploy-permissions \
    --policy-document "$policy_doc" >/dev/null
echo "  ok"

echo
echo "== Done =="
echo
echo "GitHub secret you still need to set:"
echo "  AWS_ACCOUNT_ID = ${ACCOUNT_ID}"
echo
echo "Role ARN the workflow assumes:"
echo "  ${ROLE_ARN}"
