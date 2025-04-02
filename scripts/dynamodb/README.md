# Local DynamoDB Setup

This directory contains scripts and instructions for setting up a local DynamoDB instance for development.

## Prerequisites

- Docker installed on your machine
- AWS CLI installed (for local DynamoDB operations)

## Setup Instructions

1. Start the local DynamoDB container:
```bash
docker run -p 8000:8000 amazon/dynamodb-local
```

2. Set up environment variables:
```bash
# Setting dummy values so the cli will work
export AWS_ACCESS_KEY_ID=local
export AWS_SECRET_ACCESS_KEY=local
export AWS_DEFAULT_REGION=us-east-1
export DYNAMODB_ENDPOINT=http://localhost:8000
```

3. Create the table:
```bash
aws dynamodb create-table \
  --table-name cello \
  --attribute-definitions \
    AttributeName=pk,AttributeType=S \
    AttributeName=sk,AttributeType=S \
  --key-schema \
    AttributeName=pk,KeyType=HASH \
    AttributeName=sk,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url $DYNAMODB_ENDPOINT
```

## Useful Commands

### List Tables
```bash
aws dynamodb list-tables --endpoint-url $DYNAMODB_ENDPOINT
```

### Scan Table
```bash
aws dynamodb scan --table-name cello --endpoint-url $DYNAMODB_ENDPOINT
```

### Delete Table
```bash
aws dynamodb delete-table --table-name cello --endpoint-url $DYNAMODB_ENDPOINT
```

### Reset Table
To completely reset the table (delete and recreate):
```bash
./reset-table.sh
```

## Troubleshooting

1. If you can't connect to DynamoDB:
   - Verify the Docker container is running: `docker ps`
   - Check the container logs: `docker logs <container_id>`
   - Ensure port 8000 is not in use

2. If table operations fail:
   - Verify the table exists: `aws dynamodb list-tables --endpoint-url $DYNAMODB_ENDPOINT`
   - Check table status: `aws dynamodb describe-table --table-name cello --endpoint-url $DYNAMODB_ENDPOINT`

3. If the service can't connect:
   - Verify all environment variables are set correctly
   - Check the service logs for connection errors
