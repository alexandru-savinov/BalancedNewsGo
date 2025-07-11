package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceCRUD(t *testing.T) {
	// Setup test database
	db, err := InitDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test data
	source := &Source{
		Name:          "Test Source",
		ChannelType:   "rss",
		FeedURL:       "https://example.com/feed.xml",
		Category:      "center",
		Enabled:       true,
		DefaultWeight: 1.0,
		ErrorStreak:   0,
	}

	// Test Create
	t.Run("InsertSource", func(t *testing.T) {
		id, err := InsertSource(db, source)
		assert.NoError(t, err)
		assert.Greater(t, id, int64(0))
		source.ID = id
	})

	// Test Read by ID
	t.Run("FetchSourceByID", func(t *testing.T) {
		fetched, err := FetchSourceByID(db, source.ID)
		assert.NoError(t, err)
		assert.Equal(t, source.Name, fetched.Name)
		assert.Equal(t, source.ChannelType, fetched.ChannelType)
		assert.Equal(t, source.FeedURL, fetched.FeedURL)
		assert.Equal(t, source.Category, fetched.Category)
		assert.Equal(t, source.Enabled, fetched.Enabled)
		assert.Equal(t, source.DefaultWeight, fetched.DefaultWeight)
	})

	// Test Read all sources
	t.Run("FetchSources", func(t *testing.T) {
		sources, err := FetchSources(db, nil, "", "", 0, 0)
		assert.NoError(t, err)
		assert.Len(t, sources, 1)
		assert.Equal(t, source.Name, sources[0].Name)
	})

	// Test Read enabled sources
	t.Run("FetchEnabledSources", func(t *testing.T) {
		sources, err := FetchEnabledSources(db)
		assert.NoError(t, err)
		assert.Len(t, sources, 1)
		assert.True(t, sources[0].Enabled)
	})

	// Test Update
	t.Run("UpdateSource", func(t *testing.T) {
		updates := map[string]interface{}{
			"name":           "Updated Test Source",
			"default_weight": 1.5,
		}
		err := UpdateSource(db, source.ID, updates)
		assert.NoError(t, err)

		// Verify update
		fetched, err := FetchSourceByID(db, source.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Test Source", fetched.Name)
		assert.Equal(t, 1.5, fetched.DefaultWeight)
	})

	// Test Soft Delete
	t.Run("SoftDeleteSource", func(t *testing.T) {
		err := SoftDeleteSource(db, source.ID)
		assert.NoError(t, err)

		// Verify source is disabled
		fetched, err := FetchSourceByID(db, source.ID)
		assert.NoError(t, err)
		assert.False(t, fetched.Enabled)

		// Verify it doesn't appear in enabled sources
		enabledSources, err := FetchEnabledSources(db)
		assert.NoError(t, err)
		assert.Len(t, enabledSources, 0)
	})

	// Test duplicate name prevention
	t.Run("DuplicateNamePrevention", func(t *testing.T) {
		duplicate := &Source{
			Name:          "Updated Test Source", // Same name as updated source
			ChannelType:   "rss",
			FeedURL:       "https://example2.com/feed.xml",
			Category:      "left",
			Enabled:       true,
			DefaultWeight: 1.0,
		}

		_, err := InsertSource(db, duplicate)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestSourceFiltering(t *testing.T) {
	// Setup test database
	db, err := InitDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test sources
	sources := []*Source{
		{Name: "RSS Left", ChannelType: "rss", FeedURL: "https://left.com/feed", Category: "left", Enabled: true, DefaultWeight: 1.0},
		{Name: "RSS Right", ChannelType: "rss", FeedURL: "https://right.com/feed", Category: "right", Enabled: true, DefaultWeight: 1.0},
		{Name: "Telegram Center", ChannelType: "telegram", FeedURL: "@centerchannel", Category: "center", Enabled: false, DefaultWeight: 1.0},
	}

	for _, source := range sources {
		_, err := InsertSource(db, source)
		require.NoError(t, err)
	}

	// Test filtering by enabled status
	t.Run("FilterByEnabled", func(t *testing.T) {
		enabled := true
		sources, err := FetchSources(db, &enabled, "", "", 0, 0)
		assert.NoError(t, err)
		assert.Len(t, sources, 2)
		for _, source := range sources {
			assert.True(t, source.Enabled)
		}
	})

	// Test filtering by channel type
	t.Run("FilterByChannelType", func(t *testing.T) {
		sources, err := FetchSources(db, nil, "rss", "", 0, 0)
		assert.NoError(t, err)
		assert.Len(t, sources, 2)
		for _, source := range sources {
			assert.Equal(t, "rss", source.ChannelType)
		}
	})

	// Test filtering by category
	t.Run("FilterByCategory", func(t *testing.T) {
		sources, err := FetchSources(db, nil, "", "left", 0, 0)
		assert.NoError(t, err)
		assert.Len(t, sources, 1)
		assert.Equal(t, "left", sources[0].Category)
	})

	// Test pagination
	t.Run("Pagination", func(t *testing.T) {
		sources, err := FetchSources(db, nil, "", "", 2, 0)
		assert.NoError(t, err)
		assert.Len(t, sources, 2)

		sources, err = FetchSources(db, nil, "", "", 2, 2)
		assert.NoError(t, err)
		assert.Len(t, sources, 1)
	})
}

func TestSourceExistsByName(t *testing.T) {
	// Setup test database
	db, err := InitDB(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Test non-existent source
	exists, err := SourceExistsByName(db, "Non-existent Source")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Create a source
	source := &Source{
		Name:          "Test Source",
		ChannelType:   "rss",
		FeedURL:       "https://example.com/feed.xml",
		Category:      "center",
		Enabled:       true,
		DefaultWeight: 1.0,
	}
	_, err = InsertSource(db, source)
	require.NoError(t, err)

	// Test existing source
	exists, err = SourceExistsByName(db, "Test Source")
	assert.NoError(t, err)
	assert.True(t, exists)
}
