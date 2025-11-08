# Setup local DynamoDB table for development

aws dynamodb create-table `
    --table-name healthsense-telemetry-dev `
    --attribute-definitions `
        AttributeName=PK,AttributeType=S `
        AttributeName=SK,AttributeType=S `
    --key-schema `
        AttributeName=PK,KeyType=HASH `
        AttributeName=SK,KeyType=RANGE `
    --billing-mode PAY_PER_REQUEST `
    --endpoint-url http://localhost:8000 `
    --region us-east-1

Write-Host "DynamoDB table created" -ForegroundColor Green