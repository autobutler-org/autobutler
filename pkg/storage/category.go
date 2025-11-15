package storage

// Category represents storage category types
type Category string

const (
	CategorySystem    Category = "system"
	CategoryDocuments Category = "documents"
	CategoryMedia     Category = "media"
	CategoryBackups   Category = "backups"
	CategoryOther     Category = "other"
	CategoryFree      Category = "free"
)
