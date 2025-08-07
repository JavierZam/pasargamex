# Deployment Guide - Pasargamex API

This guide covers deploying the Pasargamex API to Google Cloud Run with CI/CD pipeline.

## Prerequisites

### 1. Google Cloud Setup
- Google Cloud project with billing enabled
- Project ID: `pasargamex-458303` (or your project ID)
- Required APIs enabled:
  - Cloud Run API
  - Container Registry API
  - Firestore API
  - Cloud Storage API

### 2. Local Development Setup
- Docker installed
- Google Cloud CLI installed and authenticated
- Go 1.22.4 or later

### 3. Required Secrets and Environment Variables

#### Production Environment Variables
```bash
# Core Application
PORT=8080
ENVIRONMENT=production

# Firebase Configuration
FIREBASE_PROJECT_ID=pasargamex-458303
FIREBASE_API_KEY=your_firebase_api_key
STORAGE_BUCKET=your_storage_bucket_name

# Midtrans Payment Gateway
MIDTRANS_ENVIRONMENT=production
MIDTRANS_SERVER_KEY=your_production_server_key
MIDTRANS_CLIENT_KEY=your_production_client_key

# Firebase Service Account (for production deployment)
FIREBASE_SERVICE_ACCOUNT_JSON={"type":"service_account"...}
```

#### GitHub Actions Secrets
Set these secrets in your GitHub repository (Settings > Secrets and variables > Actions):

```
GCP_PROJECT_ID=pasargamex-458303
GCP_SA_KEY={"type":"service_account"...}  # Service account JSON key
FIREBASE_PROJECT_ID=pasargamex-458303
FIREBASE_API_KEY=your_firebase_api_key
GCS_BUCKET_NAME=your_storage_bucket_name
MIDTRANS_SERVER_KEY=your_production_server_key
MIDTRANS_CLIENT_KEY=your_production_client_key
MIDTRANS_SANDBOX_SERVER_KEY=your_sandbox_server_key
MIDTRANS_SANDBOX_CLIENT_KEY=your_sandbox_client_key
FIREBASE_SERVICE_ACCOUNT_JSON={"type":"service_account"...}
```

## Deployment Methods

### Method 1: Automated CI/CD (Recommended)

The GitHub Actions workflow automatically:
- Runs tests on every PR and push
- Creates preview deployments for PRs
- Deploys to production on main branch pushes
- Cleans up preview deployments when PRs are closed

**Workflow Files:**
- `.github/workflows/deploy-cloud-run.yml` - Main deployment workflow
- `.github/workflows/cleanup-preview.yml` - Preview cleanup workflow

**Process:**
1. Push changes to a feature branch
2. Create PR → Preview deployment is created automatically
3. Merge PR → Production deployment is triggered
4. PR closed → Preview environment is cleaned up

### Method 2: Manual Deployment

Use the deployment script for manual deployments:

```bash
# Set environment variables
export GCP_PROJECT_ID="pasargamex-458303"
export ENVIRONMENT="production"  # or "staging"

# Run deployment script
./deploy-cloud-run.sh
```

### Method 3: Direct gcloud Commands

```bash
# Build and push image
docker build -t gcr.io/pasargamex-458303/pasargamex-api:latest -f dockerfile .
gcloud auth configure-docker
docker push gcr.io/pasargamex-458303/pasargamex-api:latest

# Deploy to Cloud Run
gcloud run deploy pasargamex-api \
  --image=gcr.io/pasargamex-458303/pasargamex-api:latest \
  --platform=managed \
  --region=asia-southeast2 \
  --allow-unauthenticated \
  --port=8080 \
  --memory=1Gi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=10
```

## Post-Deployment Configuration

### 1. Update Midtrans Webhook URLs
After deployment, update your Midtrans merchant portal with the new webhook URL:

**Production URL:**
```
https://pasargamex-api-[hash].a.run.app/v1/payments/webhook
```

### 2. Update Frontend Configuration
Update your frontend applications to point to the new API endpoint.

### 3. DNS Configuration (Optional)
Configure custom domain if needed:
```bash
# Map custom domain to Cloud Run service
gcloud run domain-mappings create --service=pasargamex-api --domain=api.yourdomain.com
```

### 4. SSL/TLS Certificates
Cloud Run automatically provides SSL certificates for `*.run.app` domains.
For custom domains, certificates are automatically provisioned.

## Monitoring and Maintenance

### View Logs
```bash
# Tail logs in real-time
gcloud logs tail --follow --resource-type cloud_run_revision --resource-name pasargamex-api

# View recent logs
gcloud logs read "resource.type=cloud_run_revision" --limit=50
```

### Health Check
The application provides a health endpoint:
```bash
curl https://your-service-url/health
```

### Scaling Configuration
Current configuration:
- **Production:** 0-10 instances, 1 CPU, 1Gi memory
- **Staging/Preview:** 0-3 instances, 1 CPU, 512Mi memory
- **Concurrency:** 80 requests per instance
- **Timeout:** 1 hour (for long-running operations)

### Performance Monitoring
- Cloud Run automatically provides metrics
- Monitor via Cloud Console: Cloud Run > pasargamex-api > Metrics
- Set up alerting for error rates, latency, and instance counts

## Troubleshooting

### Common Issues

1. **Service Account Permissions**
   - Ensure the service account has Cloud Run Developer role
   - Verify Firestore and Storage permissions

2. **Environment Variables Missing**
   - Check all required environment variables are set
   - Verify secrets are properly configured in GitHub

3. **Build Failures**
   - Check Go version compatibility
   - Verify all dependencies are available
   - Review Docker build context

4. **Webhook Not Working**
   - Ensure webhook URL is accessible from internet
   - Check Midtrans signature verification
   - Verify environment (sandbox vs production) matches

### Debug Commands
```bash
# Check service status
gcloud run services describe pasargamex-api --region=asia-southeast2

# View environment variables
gcloud run services describe pasargamex-api --region=asia-southeast2 --format="export"

# Test connectivity
curl -v https://your-service-url/health
```

## Security Considerations

### Container Security
- Application runs as non-root user
- Minimal Alpine Linux base image
- No unnecessary packages installed

### Network Security
- Cloud Run provides automatic DDoS protection
- HTTPS enforced by default
- Private Google network for internal communications

### Secrets Management
- Sensitive data passed via environment variables
- Service account keys stored as GitHub secrets
- Firebase service account JSON handled securely

### Application Security
- JWT token validation for authenticated endpoints
- Midtrans webhook signature verification
- Input validation and sanitization
- Rate limiting for WebSocket connections

## Backup and Disaster Recovery

### Database Backup
- Firestore provides automatic backups
- Manual exports can be scheduled via Cloud Functions

### Container Images
- Images stored in Google Container Registry
- Automatic retention policies in place
- Tagged with timestamps for rollback capability

### Rollback Procedure
```bash
# List previous deployments
gcloud run revisions list --service=pasargamex-api

# Rollback to previous version
gcloud run services update-traffic pasargamex-api --to-revisions=REVISION_NAME=100
```

## Cost Optimization

### Current Configuration
- **CPU allocation:** Only during request processing
- **Memory:** Optimized per environment (512Mi staging, 1Gi production)
- **Min instances:** 0 (scales to zero when idle)
- **Concurrency:** 80 requests per instance

### Cost Monitoring
- Monitor via Cloud Billing console
- Set up budget alerts
- Review Cloud Run pricing regularly

This deployment configuration provides a robust, scalable, and cost-effective hosting solution for the Pasargamex API with proper security, monitoring, and maintenance capabilities.