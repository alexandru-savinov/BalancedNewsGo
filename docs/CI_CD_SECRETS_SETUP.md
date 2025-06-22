# CI/CD Secrets Setup Guide

This guide explains how to securely configure LLM API keys and other sensitive information for the CI/CD pipeline.

## 🔐 Required Secrets

The following secrets need to be configured in your GitHub repository:

### **LLM API Configuration**
- `LLM_API_KEY` - Primary OpenRouter API key
- `LLM_API_KEY_SECONDARY` - Secondary/backup OpenRouter API key  
- `LLM_BASE_URL` - OpenRouter API endpoint (https://openrouter.ai/api/v1/chat/completions)

### **Optional Deployment Secrets** (if using deployment features)
- `CONTAINER_REGISTRY` - Docker registry URL
- `REGISTRY_USERNAME` - Docker registry username
- `REGISTRY_PASSWORD` - Docker registry password

## 📝 Step-by-Step Setup

### Step 1: Get Your OpenRouter API Keys

1. Go to [OpenRouter.ai](https://openrouter.ai/)
2. Sign up/login to your account
3. Navigate to **API Keys** section
4. Create a new API key (or use existing)
5. Copy the API key value

### Step 2: Add Secrets to GitHub Repository

1. Navigate to your GitHub repository: `https://github.com/alexandru-savinov/BalancedNewsGo`
2. Click the **Settings** tab
3. In the left sidebar, click **Secrets and variables** → **Actions**
4. Click **New repository secret**
5. Add each secret:

```
Name: LLM_API_KEY
Value: [Your OpenRouter API key - starts with sk-or-...]

Name: LLM_API_KEY_SECONDARY
Value: [Your secondary OpenRouter API key - optional, can be same as primary]

Name: LLM_BASE_URL
Value: https://openrouter.ai/api/v1/chat/completions
```

### Step 3: Verify Configuration

After adding the secrets:

1. Go to **Actions** tab in your repository
2. Trigger a new workflow run (push a commit or manually trigger)
3. Check the workflow logs to ensure:
   - No "authentication failed" errors
   - LLM endpoints return proper responses
   - Tests pass successfully

## 🔒 Security Best Practices

### **Secret Management**
- ✅ Never commit API keys to code
- ✅ Use GitHub Secrets for sensitive data
- ✅ Rotate API keys regularly
- ✅ Use different keys for different environments

### **Access Control**
- ✅ Limit repository access to trusted collaborators
- ✅ Use environment-specific secrets when possible
- ✅ Monitor API key usage in OpenRouter dashboard

### **Cost Management**
- ✅ Set usage limits in OpenRouter dashboard
- ✅ Monitor API costs regularly
- ✅ Use `NO_AUTO_ANALYZE=true` in CI to limit API calls

## 🚨 Troubleshooting

### Common Issues

**1. "Context access might be invalid" warnings**
- These are IDE linting warnings
- They disappear once secrets are configured in GitHub
- The workflow will work correctly

**2. "Authentication failed" errors**
- Check that `LLM_API_KEY` secret is set correctly
- Verify the API key is valid in OpenRouter dashboard
- Ensure the key has sufficient credits/permissions

**3. "Secret not found" errors**
- Verify secret names match exactly (case-sensitive)
- Check that secrets are set at repository level, not environment level
- Ensure you have admin access to the repository

### Testing Secrets Locally

For local development, create a `.env` file (never commit this):

```bash
# .env (DO NOT COMMIT)
LLM_API_KEY=your-openrouter-key-here
LLM_API_KEY_SECONDARY=your-secondary-key-here
LLM_BASE_URL=https://openrouter.ai/api/v1/chat/completions
```

## 📊 Monitoring

### API Usage Monitoring
- Check OpenRouter dashboard for usage statistics
- Monitor costs and rate limits
- Set up alerts for unusual usage patterns

### CI/CD Monitoring
- Review workflow logs regularly
- Monitor test success rates
- Check for any authentication issues

## 🔄 Key Rotation

To rotate API keys:

1. Generate new key in OpenRouter dashboard
2. Update the secret in GitHub repository settings
3. Test with a workflow run
4. Revoke old key in OpenRouter dashboard

## 📞 Support

If you encounter issues:
- Check OpenRouter documentation: https://openrouter.ai/docs
- Review GitHub Actions documentation: https://docs.github.com/en/actions
- Check repository workflow logs for detailed error messages
