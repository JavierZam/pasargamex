#!/bin/bash

# Manual deployment script for Google Cloud Run
# This script provides an alternative to GitHub Actions for manual deployments

set -e

# Configuration
PROJECT_ID=${GCP_PROJECT_ID:-"pasargamex-458303"}
SERVICE_NAME="pasargamex-api"
REGION="asia-southeast2"  # Jakarta region
IMAGE_NAME="gcr.io/${PROJECT_ID}/${SERVICE_NAME}"
ENVIRONMENT=${ENVIRONMENT:-"production"}

echo "🚀 Deploying ${SERVICE_NAME} to Google Cloud Run"
echo "Project: ${PROJECT_ID}"
echo "Region: ${REGION}"
echo "Environment: ${ENVIRONMENT}"
echo "=================================="

# Check if gcloud is installed and authenticated
if ! command -v gcloud &> /dev/null; then
    echo "❌ Google Cloud CLI is not installed. Please install it first."
    echo "Visit: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check authentication
if ! gcloud auth list --filter="status:ACTIVE" --format="value(account)" | grep -q "@"; then
    echo "❌ Not authenticated with Google Cloud. Please run: gcloud auth login"
    exit 1
fi

# Set the project
echo "📋 Setting project to ${PROJECT_ID}"
gcloud config set project ${PROJECT_ID}

# Enable required APIs
echo "🔧 Enabling required APIs..."
gcloud services enable run.googleapis.com
gcloud services enable containerregistry.googleapis.com

# Build and tag the Docker image
echo "🔨 Building Docker image..."
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
IMAGE_TAG="${IMAGE_NAME}:${TIMESTAMP}"
IMAGE_LATEST="${IMAGE_NAME}:latest"

docker build -t ${IMAGE_TAG} -t ${IMAGE_LATEST} -f dockerfile .

# Configure Docker for gcloud
echo "🐳 Configuring Docker authentication..."
gcloud auth configure-docker

# Push the image to Google Container Registry
echo "📤 Pushing image to Container Registry..."
docker push ${IMAGE_TAG}
docker push ${IMAGE_LATEST}

# Deploy to Cloud Run
echo "🚀 Deploying to Cloud Run..."

# Check if we're deploying to production or staging
if [ "${ENVIRONMENT}" = "production" ]; then
    echo "🔴 Deploying to PRODUCTION environment"
    MIDTRANS_ENV="production"
    MEMORY="1Gi"
    CPU="1"
    MAX_INSTANCES="10"
else
    echo "🟡 Deploying to STAGING environment"
    MIDTRANS_ENV="sandbox"
    MEMORY="512Mi"
    CPU="1"
    MAX_INSTANCES="3"
fi

# Deploy the service
gcloud run deploy ${SERVICE_NAME} \
    --image=${IMAGE_TAG} \
    --platform=managed \
    --region=${REGION} \
    --allow-unauthenticated \
    --port=8080 \
    --memory=${MEMORY} \
    --cpu=${CPU} \
    --min-instances=0 \
    --max-instances=${MAX_INSTANCES} \
    --concurrency=80 \
    --timeout=3600 \
    --set-env-vars="PORT=8080" \
    --set-env-vars="ENVIRONMENT=${ENVIRONMENT}" \
    --set-env-vars="MIDTRANS_ENVIRONMENT=${MIDTRANS_ENV}"

# Get the service URL
SERVICE_URL=$(gcloud run services describe ${SERVICE_NAME} --region=${REGION} --format='value(status.url)')

echo ""
echo "✅ Deployment completed successfully!"
echo "🌐 Service URL: ${SERVICE_URL}"
echo "🔍 Health Check: ${SERVICE_URL}/health"
echo ""
echo "📝 Next Steps:"
echo "1. Test the health endpoint: curl ${SERVICE_URL}/health"
echo "2. Update Midtrans webhook URLs to: ${SERVICE_URL}/v1/payments/webhook"
echo "3. Update any frontend applications to use the new API URL"
echo "4. Monitor logs: gcloud logs tail --follow --resource-type cloud_run_revision --resource-name ${SERVICE_NAME}"
echo ""
echo "🔧 Environment Configuration:"
echo "- Set these secrets in your environment or update the deployment:"
echo "  - FIREBASE_PROJECT_ID"
echo "  - FIREBASE_API_KEY"
echo "  - STORAGE_BUCKET"
echo "  - MIDTRANS_SERVER_KEY"
echo "  - MIDTRANS_CLIENT_KEY"
echo "  - FIREBASE_SERVICE_ACCOUNT_JSON (for production)"
echo ""

# Show recent logs
echo "📋 Recent deployment logs:"
gcloud logs read "resource.type=cloud_run_revision AND resource.labels.service_name=${SERVICE_NAME}" --limit=20 --region=${REGION}