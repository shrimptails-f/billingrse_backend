package domain

// SaveStatus indicates whether a fetched email created a new row or matched an existing one.
type SaveStatus string

const (
	SaveStatusCreated  SaveStatus = "created"
	SaveStatusExisting SaveStatus = "existing"
)

// SaveResult captures the persistence outcome for a single fetched email.
type SaveResult struct {
	EmailID           uint
	ExternalMessageID string
	Status            SaveStatus
}
