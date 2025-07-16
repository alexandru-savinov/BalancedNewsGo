# Create baseline sources for E2E tests
# Order is important: BBC News must be ID 2

Write-Host "Creating baseline sources for E2E tests..."

# Source 1: HuffPost (left)
$source1 = @{
    name = "HuffPost"
    channel_type = "rss"
    feed_url = "https://www.huffpost.com/section/front-page/feed"
    category = "left"
    default_weight = 1.0
} | ConvertTo-Json

Write-Host "Creating Source 1: HuffPost..."
try {
    $response1 = Invoke-RestMethod -Uri 'http://localhost:8080/api/sources' -Method POST -Body $source1 -ContentType 'application/json'
    Write-Host "✓ Created HuffPost (ID: $($response1.data.id))" -ForegroundColor Green
} catch {
    Write-Host "✗ Failed to create HuffPost: $($_.Exception.Message)" -ForegroundColor Red
}

# Source 2: BBC News (center) - THIS IS WHAT TESTS EXPECT
$source2 = @{
    name = "BBC News"
    channel_type = "rss"
    feed_url = "https://feeds.bbci.co.uk/news/rss.xml"
    category = "center"
    default_weight = 1.0
} | ConvertTo-Json

Write-Host "Creating Source 2: BBC News..."
try {
    $response2 = Invoke-RestMethod -Uri 'http://localhost:8080/api/sources' -Method POST -Body $source2 -ContentType 'application/json'
    Write-Host "✓ Created BBC News (ID: $($response2.data.id))" -ForegroundColor Green
} catch {
    Write-Host "✗ Failed to create BBC News: $($_.Exception.Message)" -ForegroundColor Red
}

# Source 3: MSNBC (right)
$source3 = @{
    name = "MSNBC"
    channel_type = "rss"
    feed_url = "http://www.msnbc.com/feeds/latest"
    category = "right"
    default_weight = 1.0
} | ConvertTo-Json

Write-Host "Creating Source 3: MSNBC..."
try {
    $response3 = Invoke-RestMethod -Uri 'http://localhost:8080/api/sources' -Method POST -Body $source3 -ContentType 'application/json'
    Write-Host "✓ Created MSNBC (ID: $($response3.data.id))" -ForegroundColor Green
} catch {
    Write-Host "✗ Failed to create MSNBC: $($_.Exception.Message)" -ForegroundColor Red
}

# Verify all sources were created
Write-Host "`nVerifying baseline sources..."
try {
    $allSources = Invoke-RestMethod -Uri 'http://localhost:8080/api/sources' -Method GET
    Write-Host "Total sources created: $($allSources.data.total)"

    foreach ($source in $allSources.data.sources) {
        Write-Host "- ID $($source.id): $($source.name) ($($source.category))"
    }

    # Check if BBC News is ID 2
    $bbcNews = $allSources.data.sources | Where-Object { $_.id -eq 2 }
    if ($bbcNews -and $bbcNews.name -eq "BBC News") {
        Write-Host "✓ BBC News correctly has ID 2 - tests should pass!" -ForegroundColor Green
    } else {
        Write-Host "✗ BBC News does not have ID 2 - tests will fail!" -ForegroundColor Red
    }
} catch {
    Write-Host "Failed to verify sources: $($_.Exception.Message)" -ForegroundColor Red
}
