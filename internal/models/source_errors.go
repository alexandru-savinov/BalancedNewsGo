package models

import "errors"

// Source validation errors
var (
	ErrSourceNameRequired        = errors.New("source name is required")
	ErrSourceChannelTypeRequired = errors.New("source channel type is required")
	ErrSourceInvalidChannelType  = errors.New("invalid channel type")
	ErrSourceFeedURLRequired     = errors.New("source feed URL is required")
	ErrSourceCategoryRequired    = errors.New("source category is required")
	ErrSourceInvalidCategory     = errors.New("invalid category")
	ErrSourceInvalidWeight       = errors.New("default weight must be non-negative")
	ErrSourceNotFound            = errors.New("source not found")
	ErrSourceNameExists          = errors.New("source with this name already exists")
)
