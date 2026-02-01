package model

// InitFormType represents the type of init form (participant vs corporate).
type InitFormType string

const (
	InitFormParticipant InitFormType = "participant"
	InitFormCorporate   InitFormType = "corporate"
)

// AvailableTags lists all tags available for selection in init forms.
var AvailableTags = []string{
	"Accelerationist", "Anarchist", "Ancap",
	"Belgrade", "Blockchain", "Blogger", "Business",
	"Catholic", "Charity", "Christian", "Crypto",
	"Defi", "Designer", "Developer",
	"Entrepreneur",
	"Investor",
	"Libertarian",
	"Management", "Marketing", "Monarchist", "Montenegro",
	"Neoreactionist", "Nft",
	"Orthodox",
	"Panarchist",
	"Sales", "Startup",
	"Traditionalist",
}

// NumberedField represents a field with a preserved index string.
// Index is stored as string to preserve leading zeros (e.g., "002" vs "2").
type NumberedField struct {
	Index string `json:"i"` // "0", "001", "002" - preserved as string
	Value string `json:"v"` // Account ID or URL
}

// ParticipantFormData holds all fields for participant init form.
type ParticipantFormData struct {
	AccountID string
	Name      string
	About     string
	Website   string
	PartOf    []NumberedField // PartOf001, PartOf002, etc.
	Tags      []string        // TagBelgrade, TagDeveloper, etc.
}

// CorporateFormData holds all fields for corporate init form.
type CorporateFormData struct {
	AccountID string
	Name      string
	About     string
	Website   string
	MyPart    []NumberedField // MyPart001, MyPart002, etc.
	Tags      []string        // TagBelgrade, TagInvestor, etc.
}

// InitLandingData holds data for the init landing page.
type InitLandingData struct {
	Page  string // "landing"
	Error string
}

// InitFormData holds data for rendering init forms.
type InitFormData struct {
	Page          string      // "participant" or "corporate"
	AccountID     string      // User's Stellar account ID
	FormData      interface{} // ParticipantFormData or CorporateFormData
	OriginalJSON  string      // Base64-encoded JSON of original data
	AvailableTags []string    // List of available tags
	Error         string      // Error message to display
	FormAction    string      // Form action URL
	PreviewAction string      // Preview action URL
}

// InitPreviewData holds data for the XDR preview page.
type InitPreviewData struct {
	Page       string          // "preview"
	AccountID  string          // User's Stellar account ID
	XDR        string          // Base64-encoded unsigned XDR
	Operations []InitOpSummary // Human-readable operation list
	LabLink    string          // Stellar Laboratory link
	Error      string          // Error message
}

// InitOpSummary describes a ManageData operation for display.
type InitOpSummary struct {
	Action string // "Set" or "Delete"
	Key    string // ManageData key
	Value  string // ManageData value (empty for delete)
}
