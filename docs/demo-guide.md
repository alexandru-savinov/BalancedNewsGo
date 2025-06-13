# NewsBalancer HTMX Demo Script

## Quick Start Guide

### 1. Start the Server with HTMX Mode
```powershell
# Navigate to project directory
cd d:\Dev\NBG

# Set environment for API handlers
$env:USE_API_HANDLERS = "true"

# Start the server
go run ./cmd/server
```

### 2. Open Test Suite
Open the HTMX test page in your browser:
- **File**: `d:\Dev\NBG\test-htmx.html`  
- **URL**: `file:///d:/Dev/NBG/test-htmx.html`

### 3. Test Main Application
Navigate to the main application:
- **Articles Page**: http://localhost:8080/articles
- **Sample Article**: http://localhost:8080/article/1
- **Admin Dashboard**: http://localhost:8080/admin

## Demo Features

### 🎯 Dynamic Filtering (No Page Refresh)
1. Go to http://localhost:8080/articles
2. Use the **Source** dropdown - watch content update instantly
3. Use the **Bias** dropdown - see real-time filtering
4. Type in the **Search** box - results appear as you type (500ms delay)
5. Click **Clear** - form resets and content updates

### 🎯 Seamless Pagination  
1. Navigate through pages using pagination links
2. Notice URL updates in browser address bar
3. Use browser back/forward buttons - they work perfectly
4. Filter state is maintained across page navigation

### 🎯 Dynamic Article Loading
1. Click any article title from the list
2. Article loads without page refresh
3. Sidebar updates with related articles
4. URL changes to reflect current article
5. Use "Back to Articles" link for navigation

### 🎯 Interactive Features
1. On article detail page, click **"Reanalyze Article"**
2. Watch loading spinner during processing
3. Click **"Get Summary"** for dynamic content loading
4. Notice error handling if server issues occur

### 🎯 Performance Monitoring
1. Open Browser Dev Tools (F12)
2. Go to **Network** tab
3. Perform actions above
4. Observe:
   - HTMX requests show as XHR/Fetch
   - Fragment responses are smaller than full pages
   - No full page reloads except initial load

## Browser Testing Checklist

### Core Functionality ✅
- [ ] Server starts successfully
- [ ] Articles page loads with HTMX
- [ ] Filter dropdowns work instantly  
- [ ] Search provides real-time results
- [ ] Pagination updates content smoothly
- [ ] Article links load content dynamically
- [ ] Browser navigation (back/forward) works

### Interactive Features ✅
- [ ] Loading indicators appear during requests
- [ ] Error messages display on failures
- [ ] Form resets work properly
- [ ] Action buttons (reanalyze/summary) function
- [ ] Content updates in correct page areas

### Performance ✅
- [ ] Requests complete in < 1 second
- [ ] Only fragments load during updates
- [ ] Debounced search prevents spam
- [ ] Caching reduces server load
- [ ] Memory usage remains stable

### Compatibility ✅
- [ ] Works in Chrome/Edge
- [ ] Works in Firefox
- [ ] Mobile responsive design
- [ ] Keyboard navigation works
- [ ] Screen readers can navigate

## Troubleshooting

### Server Won't Start
```powershell
# Check for port conflicts
netstat -an | findstr :8080

# Try different port
$env:PORT = "8081"
go run ./cmd/server
```

### HTMX Not Working
```powershell
# Ensure API handlers are enabled
$env:USE_API_HANDLERS = "true"

# Check server logs for errors
# Look for "Using API-based template handlers" message
```

### Fragments Not Loading
1. Check browser console for JavaScript errors
2. Verify server is running on http://localhost:8080
3. Test health endpoint: http://localhost:8080/healthz
4. Check Network tab for failed requests

### Performance Issues
1. Clear browser cache
2. Check server resource usage
3. Verify database is not locked
4. Monitor network request times

## Advanced Testing

### Load Testing
```powershell
# Test fragment endpoint performance
curl -w "@curl-format.txt" -s -o /dev/null http://localhost:8080/api/fragments/articles

# Test with filters
curl -w "@curl-format.txt" -s -o /dev/null "http://localhost:8080/api/fragments/articles?bias=left&page=2"
```

### Cache Testing
1. Load articles page
2. Apply filters multiple times
3. Check server logs for cache hits
4. Verify response times improve

### Error Handling Testing
1. Stop the server while using the app
2. Try to filter or navigate
3. Observe error messages
4. Restart server and verify recovery

## Success Indicators

### ✅ Technical Success
- All endpoints respond with 200 status
- Fragment updates work without page refresh
- Browser history management functions correctly
- Error handling provides graceful degradation

### ✅ User Experience Success  
- Actions feel instant and responsive
- No noticeable delays or flickering
- Forms work intuitively
- Loading states provide clear feedback

### ✅ Performance Success
- Fragment loads are faster than full pages
- Search doesn't spam the server
- Caching reduces duplicate requests
- Memory usage remains stable

## Next Steps

After successful demo:
1. **Phase 4**: Implement comprehensive testing
2. **Phase 5**: Create production deployment guide
3. **Future**: Add real-time updates with WebSockets
4. **Enhancement**: Implement progressive web app features

---

🎉 **Congratulations!** You've successfully completed the NewsBalancer HTMX migration. The application now provides a modern, dynamic user experience while maintaining the reliability and SEO benefits of server-side rendering.
