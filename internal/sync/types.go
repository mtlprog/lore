package sync

import "github.com/shopspring/decimal"

// RelationType represents the type of relationship between accounts.
type RelationType string

const (
	RelationMyPart            RelationType = "MyPart"
	RelationPartOf            RelationType = "PartOf"
	RelationRecommendToMTLA   RelationType = "RecommendToMTLA"
	RelationOneFamily         RelationType = "OneFamily"
	RelationSpouse            RelationType = "Spouse"
	RelationGuardian          RelationType = "Guardian"
	RelationWard              RelationType = "Ward"
	RelationSympathy          RelationType = "Sympathy"
	RelationLove              RelationType = "Love"
	RelationDivorce           RelationType = "Divorce"
	RelationA                 RelationType = "A"
	RelationB                 RelationType = "B"
	RelationC                 RelationType = "C"
	RelationD                 RelationType = "D"
	RelationEmployer          RelationType = "Employer"
	RelationEmployee          RelationType = "Employee"
	RelationContractor        RelationType = "Contractor"
	RelationClient            RelationType = "Client"
	RelationPartnership       RelationType = "Partnership"
	RelationCollaboration     RelationType = "Collaboration"
	RelationOwnershipFull     RelationType = "OwnershipFull"
	RelationOwnershipMajority RelationType = "OwnershipMajority"
	RelationOwnershipMinority RelationType = "OwnershipMinority"
	RelationOwner             RelationType = "Owner"
	RelationOwnerMajority     RelationType = "OwnerMajority"
	RelationOwnerMinority     RelationType = "OwnerMinority"
	RelationWelcomeGuest      RelationType = "WelcomeGuest"
	RelationFactionMember     RelationType = "FactionMember"
)

// TagName represents an association tag type.
type TagName string

const (
	TagProgram TagName = "Program"
	TagFaction TagName = "Faction"
)

// Balance represents an account balance with decimal precision.
type Balance struct {
	AssetCode   string
	AssetIssuer string
	Balance     decimal.Decimal
}

// Metadata represents account metadata with string index.
// Index preserves the original suffix (e.g., "002" vs "2") to avoid duplicates.
type Metadata struct {
	Key   string
	Index string
	Value string
}

// Relationship represents a relationship between accounts.
type Relationship struct {
	TargetAccountID string
	RelationType    RelationType
	RelationIndex   string // Keep original suffix (e.g., "002" vs "2") to avoid conflicts
}

// AssociationTag represents an association tag.
type AssociationTag struct {
	TagName         TagName
	TagIndex        int
	TargetAccountID string
}

// SyncStats holds aggregate statistics after sync.
type SyncStats struct {
	TotalAccounts  int
	TotalPersons   int
	TotalCompanies int
	TotalXLMValue  decimal.Decimal
}

// SyncResult holds the result of a sync operation.
type SyncResult struct {
	Stats           *SyncStats
	FailedAccounts  []string
	FailedPrices    []string
	AccountFailRate float64
	PriceFailRate   float64
}

// Asset represents a Stellar asset.
type Asset struct {
	Code   string
	Issuer string
}

// AccountData holds parsed account information from Horizon.
type AccountData struct {
	ID                string
	Name              string // Primary name from ManageData "Name" key
	Balances          []Balance
	Metadata          []Metadata
	Relationships     []Relationship
	DelegateTo        *string // mtla_delegate - general delegation
	CouncilDelegateTo *string // mtla_c_delegate when it's an account ID
	CouncilReady      bool    // mtla_c_delegate == "ready"
}

// DelegationInfo holds delegation data for an account.
type DelegationInfo struct {
	AccountID         string
	DelegateTo        *string // mtla_delegate
	CouncilDelegateTo *string // mtla_c_delegate when it's an account ID
	MTLAPBalance      decimal.Decimal
	CouncilReady      bool // mtla_c_delegate == "ready"
}

// DefaultFailureThreshold is the default maximum failure rate (10%)
// before sync is considered failed.
const DefaultFailureThreshold = 0.1
