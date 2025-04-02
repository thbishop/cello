#!/bin/bash

set -e

if [ -z "$DYNAMODB_ENDPOINT" ]; then
    echo "Error: DYNAMODB_ENDPOINT environment variable is not set"
    exit 1
fi

# Delete the table if it exists
aws dynamodb delete-table \
  --table-name cello \
  --endpoint-url $DYNAMODB_ENDPOINT \
  || true

# Wait for table deletion to complete
sleep 2

# Create the table
aws dynamodb create-table \
  --table-name cello \
  --attribute-definitions \
    AttributeName=PK,AttributeType=S \
    AttributeName=SK,AttributeType=S \
  --key-schema \
    AttributeName=PK,KeyType=HASH \
    AttributeName=SK,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url $DYNAMODB_ENDPOINT

# Wait for table creation to complete
sleep 2

echo "Table 'cello' has been reset successfully" 