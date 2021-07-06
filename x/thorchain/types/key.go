package types

const (
	// module names
	ModuleName       = "thorchain"
	DefaultCodespace = ModuleName
	ReserveName      = "reserve"
	AsgardName       = "asgard"
	BondName         = "bond"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	RouterKey = ModuleName // this was defined in your key.go file
)
