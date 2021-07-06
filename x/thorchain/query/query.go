package query

import (
	"fmt"
	"strings"
)

// Query define all the queries
type Query struct {
	Key              string
	EndpointTemplate string
}

// Endpoint return the end point string
func (q Query) Endpoint(args ...string) string {
	count := strings.Count(q.EndpointTemplate, "%s")
	a := args[:count]

	in := make([]interface{}, len(a))
	for i := range in {
		in[i] = a[i]
	}

	return fmt.Sprintf(q.EndpointTemplate, in...)
}

// Path return the path
func (q Query) Path(args ...string) string {
	temp := []string{args[0], q.Key}
	args = append(temp, args[1:]...)
	return fmt.Sprintf("custom/%s", strings.Join(args, "/"))
}

// query endpoints supported by the thorchain Querier
var (
	QueryPool               = Query{Key: "pool", EndpointTemplate: "/%s/pool/{%s}"}
	QueryPools              = Query{Key: "pools", EndpointTemplate: "/%s/pools"}
	QueryStakers            = Query{Key: "stakers", EndpointTemplate: "/%s/pool/{%s}/stakers"}
	QueryTxIn               = Query{Key: "txin", EndpointTemplate: "/%s/tx/{%s}"}
	QueryTxInVoter          = Query{Key: "txinvoter", EndpointTemplate: "/%s/tx/{%s}/voter"}
	QueryKeysignArray       = Query{Key: "keysign", EndpointTemplate: "/%s/keysign/{%s}"}
	QueryKeysignArrayPubkey = Query{Key: "keysignpubkey", EndpointTemplate: "/%s/keysign/{%s}/{%s}"}
	QueryKeygensPubkey      = Query{Key: "keygenspubkey", EndpointTemplate: "/%s/keygen/{%s}/{%s}"}
	QueryQueue              = Query{Key: "outqueue", EndpointTemplate: "/%s/queue"}
	QueryHeights            = Query{Key: "heights", EndpointTemplate: "/%s/lastblock"}
	QueryChainHeights       = Query{Key: "chainheights", EndpointTemplate: "/%s/lastblock/{%s}"}
	QueryObservers          = Query{Key: "observers", EndpointTemplate: "/%s/observers"}
	QueryObserver           = Query{Key: "observer", EndpointTemplate: "/%s/observer/{%s}"}
	QueryNodeAccounts       = Query{Key: "nodeaccounts", EndpointTemplate: "/%s/nodeaccounts"}
	QueryNodeAccount        = Query{Key: "nodeaccount", EndpointTemplate: "/%s/nodeaccount/{%s}"}
	QueryNodeAccountCheck   = Query{Key: "nodeaccountcheck", EndpointTemplate: "/%s/nodeaccount/{%s}/preflight"}
	QueryPoolAddresses      = Query{Key: "pooladdresses", EndpointTemplate: "/%s/pool_addresses"}
	QueryVaultData          = Query{Key: "vaultdata", EndpointTemplate: "/%s/vault"}
	QueryBalanceModule      = Query{Key: "balancemodule", EndpointTemplate: "/%s/balance/module/{%s}"}
	QueryVaultsAsgard       = Query{Key: "vaultsasgard", EndpointTemplate: "/%s/vaults/asgard"}
	QueryVaultsYggdrasil    = Query{Key: "vaultsyggdrasil", EndpointTemplate: "/%s/vaults/yggdrasil"}
	QueryVault              = Query{Key: "vault", EndpointTemplate: "/%s/vault/{%s}/{%s}"}
	QueryVaultPubkeys       = Query{Key: "vaultpubkeys", EndpointTemplate: "/%s/vaults/pubkeys"}
	QueryTSSSigners         = Query{Key: "tsssigner", EndpointTemplate: "/%s/vaults/{%s}/signers"}
	QueryConstantValues     = Query{Key: "constants", EndpointTemplate: "/%s/constants"}
	QueryVersion            = Query{Key: "version", EndpointTemplate: "/%s/version"}
	QueryMimirValues        = Query{Key: "mimirs", EndpointTemplate: "/%s/mimir"}
	QueryBan                = Query{Key: "ban", EndpointTemplate: "/%s/ban/{%s}"}
	QueryRagnarok           = Query{Key: "ragnarok", EndpointTemplate: "/%s/ragnarok"}
	QueryPendingOutbound    = Query{Key: "pendingoutbound", EndpointTemplate: "/%s/queue/outbound"}
)

// Queries all queries
var Queries = []Query{
	QueryPool,
	QueryPools,
	QueryStakers,
	QueryTxInVoter,
	QueryTxIn,
	QueryKeysignArray,
	QueryKeysignArrayPubkey,
	QueryQueue,
	QueryHeights,
	QueryChainHeights,
	QueryObservers,
	QueryObserver,
	QueryNodeAccount,
	QueryNodeAccountCheck,
	QueryNodeAccounts,
	QueryPoolAddresses,
	QueryVaultData,
	QueryBalanceModule,
	QueryVaultsAsgard,
	QueryVaultsYggdrasil,
	QueryVaultPubkeys,
	QueryVault,
	QueryKeygensPubkey,
	QueryTSSSigners,
	QueryConstantValues,
	QueryVersion,
	QueryMimirValues,
	QueryBan,
	QueryRagnarok,
	QueryPendingOutbound,
}
