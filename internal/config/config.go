package config

const (
	// DefaultPort is the default HTTP server port.
	DefaultPort = "8080"

	// DefaultHorizonURL is the Stellar mainnet Horizon endpoint.
	DefaultHorizonURL = "https://horizon.stellar.org"

	// DefaultDatabaseURL is empty; must be provided via flag or environment.
	DefaultDatabaseURL = ""

	// TokenIssuer is the issuer account for MTLAP and MTLAC tokens.
	TokenIssuer = "GCNVDZIHGX473FEI7IXCUAEXUJ4BGCKEMHF36VYP5EMS7PX2QBLAMTLA"

	// TokenMTLAP is the asset code for Persons.
	TokenMTLAP = "MTLAP"

	// TokenMTLAC is the asset code for Companies.
	TokenMTLAC = "MTLAC"

	// TokenMTLAX is the asset code for Synthetic.
	TokenMTLAX = "MTLAX"

	// DefaultPageLimit is the default number of accounts per page.
	DefaultPageLimit = 20

	// DefaultRateLimit is the default requests per minute per IP address.
	DefaultRateLimit = 100
)
