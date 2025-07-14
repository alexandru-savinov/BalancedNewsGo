# Create the original 3 sources from feed_sources.json

$sources = @(
    @{
        name = "HuffPost"
        channel_type = "rss"
        feed_url = "https://www.huffpost.com/section/front-page/feed"
        category = "left"
        default_weight = 1.0
    },
    @{
        name = "The Guardian"
        channel_type = "rss"
        feed_url = "https://www.theguardian.com/world/rss"
        category = "center"
        default_weight = 1.0
    },
    @{
        name = "MSNBC"
        channel_type = "rss"
        feed_url = "http://www.msnbc.com/feeds/latest"
        category = "right"
        default_weight = 1.0
    }
)

foreach ($source in $sources) {
    $body = $source | ConvertTo-Json
    Write-Host "Creating source: $($source.name)"
    
    try {
        $response = Invoke-RestMethod -Uri 'http://localhost:8080/api/sources' -Method POST -Body $body -ContentType 'application/json'
        Write-Host "✓ Created $($source.name)" -ForegroundColor Green
    } catch {
        Write-Host "✗ Failed to create $($source.name): $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Verify sources were created
Write-Host "`nVerifying sources..."
try {
    $response = Invoke-RestMethod -Uri 'http://localhost:8080/api/sources' -Method GET
    Write-Host "Total sources: $($response.data.total)"
    foreach ($source in $response.data.sources) {
        Write-Host "- $($source.name) ($($source.category))"
    }
} catch {
    Write-Host "Failed to verify sources: $($_.Exception.Message)" -ForegroundColor Red
}
