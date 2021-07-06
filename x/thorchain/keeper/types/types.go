package types

import "errors"

type DbPrefix string

// ErrVaultNotFound an error indicate vault can't be found
var ErrVaultNotFound = errors.New("vault not found")
