# SonarCloud MCP Server Usage Examples

This document provides practical examples of how to use the SonarCloud MCP server with your BalancedNewsGo project.

## Basic Project Information

### List All Projects
```
"List all my SonarQube projects"
```

### Get Project Details
```
"Show me details for the BalancedNewsGo project"
"What's the current quality gate status for alexandru-savinov_BalancedNewsGo?"
```

## Code Quality Analysis

### Quality Gates
```
"Check the quality gate status for BalancedNewsGo"
"Has the BalancedNewsGo project passed its quality gate?"
"Show me quality gate conditions for my project"
```

### Code Coverage
```
"What's the code coverage for BalancedNewsGo?"
"Show me coverage metrics for the main branch"
"Compare coverage between main and develop branches"
```

### Code Metrics
```
"Get code metrics for BalancedNewsGo"
"Show me lines of code and complexity metrics"
"What are the maintainability metrics for my project?"
```

## Issue Management

### Finding Issues
```
"Show me all critical issues in BalancedNewsGo"
"List security vulnerabilities in the main branch"
"Find all code smells with high severity"
"Show me issues created in the last 7 days"
"Get all unresolved bugs in the project"
```

### Issue Details
```
"Get details for issue ABC-123"
"Show me the source code for issue XYZ-456"
"What's the rule description for issue DEF-789?"
```

### Issue Management
```
"Mark issue ABC-123 as false positive with comment 'Test code only'"
"Assign issue XYZ-456 to john.doe"
"Add comment to issue DEF-789: 'Fixed in commit abc123'"
"Mark issues ABC-123, XYZ-456 as won't fix"
```

## Security Analysis

### Security Hotspots
```
"Find all security hotspots in BalancedNewsGo"
"Show me security hotspots that need review"
"Get security hotspots in the authentication module"
"List hotspots with high vulnerability probability"
```

### Security Standards
```
"Find issues related to OWASP Top 10"
"Show me CWE-related security issues"
"Get SANS Top 25 security vulnerabilities"
```

## Branch and Pull Request Analysis

### Branch Analysis
```
"Analyze code quality in the feature/new-login branch"
"Compare issues between main and develop branches"
"Show me new issues introduced in the current branch"
```

### Pull Request Analysis
```
"Analyze issues in pull request #42"
"Show me code quality changes in PR #123"
"Get quality gate status for pull request #456"
```

## Advanced Queries

### Multi-Project Analysis
```
"Compare code quality across all my projects"
"Show me projects with failing quality gates"
"List projects with the highest technical debt"
```

### Historical Analysis
```
"Show me code quality trends for the last 3 months"
"How has code coverage changed over time?"
"Get metrics history for BalancedNewsGo"
```

### Component-Specific Analysis
```
"Analyze issues in the src/auth directory"
"Show me code quality for the API module"
"Get metrics for specific Go packages"
```

## Filtering and Search

### By Severity
```
"Show me only blocker and critical issues"
"List all minor code smells"
"Find issues with high impact on maintainability"
```

### By Type
```
"Show me all bugs in the project"
"List security vulnerabilities only"
"Find all code smells related to complexity"
```

### By Status
```
"Show me all open issues"
"List resolved issues from last week"
"Find all confirmed bugs"
```

### By Assignment
```
"Show me issues assigned to me"
"List unassigned critical issues"
"Find issues assigned to specific team members"
```

## Reporting and Dashboards

### Summary Reports
```
"Create a code quality summary for BalancedNewsGo"
"Generate a security report for the main branch"
"Show me a technical debt overview"
```

### Metrics Dashboard
```
"Create a dashboard showing key quality metrics"
"Show me a comparison of before/after metrics"
"Generate a quality trends report"
```

## Integration with Development Workflow

### Pre-Commit Checks
```
"Check if my changes will pass the quality gate"
"Analyze code quality impact of recent changes"
"Show me issues that would block deployment"
```

### Code Review Support
```
"Analyze code quality for files changed in the last commit"
"Show me potential issues in the current branch"
"Get quality feedback for the feature I'm working on"
```

### CI/CD Integration
```
"Check if the build will pass quality gates"
"Show me quality metrics for the deployment pipeline"
"Get quality gate status for the release branch"
```

## Troubleshooting and Debugging

### System Health
```
"Check if SonarQube is accessible"
"Ping the SonarQube system"
"Show me system health status"
```

### Configuration Verification
```
"List available metrics in SonarQube"
"Show me supported programming languages"
"Get quality gate configurations"
```

## Best Practices

1. **Start with Overview**: Always begin with project-level queries to understand overall health
2. **Focus on Critical Issues**: Prioritize blocker and critical issues first
3. **Use Filters**: Leverage filtering to focus on specific areas or types of issues
4. **Track Trends**: Regular monitoring of metrics over time helps identify patterns
5. **Security First**: Regularly check for security hotspots and vulnerabilities
6. **Branch Analysis**: Analyze feature branches before merging to main

## Tips for Effective Usage

- Use specific project keys when working with multiple projects
- Combine multiple queries for comprehensive analysis
- Save frequently used queries as templates
- Use the MCP server for both reactive (fixing issues) and proactive (preventing issues) workflows
- Integrate quality checks into your development routine

## Error Handling

If you encounter issues:

1. **Authentication Errors**: Check your SonarCloud token
2. **Project Not Found**: Verify project key and organization
3. **Permission Errors**: Ensure your token has appropriate permissions
4. **Network Issues**: Check connectivity to SonarCloud

For detailed troubleshooting, refer to `docs/SonarCloud-MCP-Setup.md`.
