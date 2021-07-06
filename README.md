[![pipeline status](https://gitlab.com/thorchain/thornode/badges/master/pipeline.svg)](https://github.com/SwipeWallet/swipechain-node/commits/main)
[![coverage report](https://gitlab.com/thorchain/thornode/badges/master/coverage.svg)](https://github.com/SwipeWallet/swipechain-node/commits/main)

# SwipeChain



======================================

SwipeChain is a decentralised liquidity network built with [CosmosSDK](cosmos.network). 

### SwipeNodes
The SwipeNode software allows a node to join and service the network, which will run with a minimum of four nodes. The only limitation to the number of nodes that can participate is set by the `minimumBondAmount`, which is the minimum amount of capital required to join. Nodes are not permissioned; any node that can bond the required amount of capital can be scheduled to churn in. 

SwipeChain comes to consensus about events observed on external networks via witness transactions from nodes. Swap and stake logic is then applied to these finalised events. Each event causes a state change in SwipeChain, and some events generate an output transaction which require assets to be moved (outgoing swaps or bond/liquidity withdrawals). These output transactions are then batched, signed by a threshold signature scheme protocol and broadcast back to the respective external network. The final gas fee on the network is then accounted for and the transaction complete. 

This is described as a "1-way state peg", where only state enters the system, derived from external networks. There are no pegged tokens or 2-way pegs, because they are not necessary. On-chain Bitcoin can be swapped with on-chain Ethereum in the time it takes to finalise the confirmed event. 

All funds in the system are fully accounted for and can be audited. All logic is fully transparent. 

### Churn
SwipeChain actively churns its validator set to prevent stagnation and capture, and ensure liveness in signing committees. Churning is also the mechanism by which the SwipeNode software can safely facilitate non-contentious upgrades. 

Every 50000 blocks (3 days) SwipeChain will schedule the oldest and the most unreliable node to leave, and rotate in two new nodes. The next two nodes chosen are simply the nodes with the highest bond. 

During a churn event the following happens:
* The incoming nodes participate in a TSS key-generation event to create new Asgard vault addresses
* When successful, the new vault is tested with a on-chain challenge-response. 
* If successful, the vaults are rolled forward, moving all assets from the old vault to the new vault. 
* The outgoing nodes are refunded their bond and removed from the system. 

### Bifröst
The Bifröst faciliates connections with external networks, such as Binance Chain, Ethereum and Bitcoin. The Bifröst is generally well-abstracted, needing only minor changes between different chains. The Bifröst handles observations of incoming transactions, which are passed into SwipeChain via special witness transactions. The Bifröst also handles multi-party computation to sign outgoing transactions via a Genarro-Goldfeder TSS scheme. Only 2/3rds of nodes are required to be in each signing ceremony on a first-come-first-serve basis, and there is no log of who is present. In this way, each node maintains plausible deniabilty around involvement with every transaction. 

To add a new chain, adapt one of the existing modules to the new chain, and submit a merge request to be tested and validated. Once merged, new nodes can start signalling support for the new chain. Once a super-majority (67%) of nodes support the new chain it will be added to the network. 

To remove a chain, nodes can stop witnessing it. If a super-majority of nodes do not promptly follow suit, the non-witnessing nodes will attract penalties during the time they do not witness it. If a super-majority of nodes stop witnessing a chain it will invoke a chain-specific Ragnörok, where all funds attributed to that chain will be returned and the chain delisted. 

### Transactions 
The SwipeChain facilitates the following transactions, which are made on external networks and replayed into the SwipeChain via witness transactions:
- **STAKE**: Anyone can stake assets in pools. If the asset hasn't been seen before, a new pool is created. 
- **WITHDRAW**: Anyone who is staking can withdraw their claim on the pool.
- **SWAP**: Anyone can send in assets and swap to another, including sending to a destination address, and including optional price protection. 
- **BOND**: Anyone can bond assets and attempt to become a Node. Bonds must be greater than the `minimumBondAmount`, else they will be refunded. 
- **LEAVE**: Nodes can voluntarily leave the system and their bond and rewards will be paid out. Leaving takes 6 hours. 
- **RESERVE**: Anyone can add assets to the Protocol Reserve, which pays out to Nodes and Stakers. 220,447,472 SXP will be funded in this way. 

### Continuous Liquidity Pools
The Staking, Unstaking and Swapping logic is based on the `CLP` Continuous Liquidity Pool algorithm. 

**Swaps**
The algorithm for processing assets swaps is given by:
`y = (x * Y * X) / (x + X)^2`, where `x = input, X = Input Asset, Y = Output Asset, y = output`

The fee paid by the trader is given by:
`fee = ( x^2 *  Y ) / ( x + X )^2 `

The slip-based fee model has the following benefits:
* Resistant to manipulation
* A proxy for demand of liquidity
* Asymptotes to zero over time, ensuring pool prices match reference prices
* Prevents Impermanent Loss to liquidity providers

**Staking**
The stake units awarded to a liquidity provider is given by:
`stakeUnits = ((R + T) * (r * T + R * t))/(4 * R * T)`, where `r = SXP Staked, R = SXP Balance, T = Token Balance, t = Token Staked`

This allows them to stake asymmetrically since it has no opinion on price. 

### Incentives
The system is safest and most capital-efficient when 67% of SXP is bonded and 33% is staked in pools. At this point, nodes will be paid 67% of the System Income, and liquidity providers will be paid 33% of the income. The Sytem Income is the block rewards (`blockReward = totalReserve / 6 / 6311390`) plus the liquidity fees collected in that block. 

An Incentive Pendulum ensures that liquidity providers receive 100% of the income when 0% is staked (inefficent), and 0% of the income when `totalStaked >= totalBonded` (unsafe).
The Total Reserve accumulates the `transactionFee`, which pays for outgoing gas fees and stabilises long-term value accrual. 

### Governance
There is strictly minimal goverance possible through SwipeNode software. Each SwipeNode can only generate valid blocks that is fully-compliant with the binary run by the super-majority. 

The best way to apply changes to the system is to submit a SwipeChain Improvement Proposal (TIP) for testing, validation and discussion among the SwipeChain developer community. If the change is beneficial to the network, it can be merged into the binary. New nodes may opt to run this updated binary, signalling via a `semver` versioning scheme. Once the super-majority are on the same binary, the system will update automatically. Schema and logic changes can be applied via this approach. 

Changes to the Bifröst may not need coordination, as long as the changes don't impact SwipeChain schema or logic, such as adding new chains. 

Emergency changes to the protocol may be difficult to coordinate, since there is no ability to communicate with any of the nodes. The best way to handle an emergency is to invoke Ragnarök, simply by leaving the system. When the system falls below 4 nodes all funds are paid out and the system can be shut-down. 

======================================

## Setup
Ensure you have a recent version of go (ie `1.15`) and enabled go modules
And have `GOBIN` in your `PATH`
```
export GOBIN=$GOPATH/bin
```

### Automated Install Locally
Install via this `make` command.

```bash
make install
```

Once you've installed `swipecli` and `swiped`, check that they are there.

```bash
swipecli help
swiped help
```

### Start Standalone Full Stack
For development and running a full chain locally (your own separate network),
use the following `make` command.
```bash
make -C build/docker reset-mocknet-standalone
```

### Format code
```bash
make format
```

### Build all
```bash
make all
```

### Test
Run tests
```bash
make test
# test with NATIVE SXP
NATIVE=true make test
```

To run test live when you change a file, use...
```
go get -u github.com/mitranim/gow
make test-watch
```

### How to contribute

* Create an issue or find an existing issue on https://github.com/SwipeWallet/swipechain-node/-/issues
* Assign the issue to yourself
* Create a branch using the issue id, for example if the issue you are working on is 600, then create a branch call `600-issue` , this way , github will link your PR with the issue
* Raise a PR , and submit it for the team to review
* Make sure the pipeline is green
* Once PR get approved, you can merge it to master

### the semantic version and release
SwipeNode manage changelog entry the same way like github, refer to (https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/creating-a-pull-request-template-for-your-repository) for more detail. Once a merge request get merged into master branch,
if the merge request upgrade the [version](https://github.com/SwipeWallet/swipechain-node/-/blob/master/version), then a new release will be created automatically, and the repository will be tagged with
the new version by the release tool.

### How to generate a changelog entry
A scripts/changelog  is available to generate the changelog entry file automatically.

Its simplest usage is to provide the value for title:
```
./scripts/changelog "my super amazing change"
```
At this point the script would ask you to select the category of the change (mapped to the type field in the entry):
```bash
>> Please specify the category of your change:
1. New feature
2. Bug fix
3. Feature change
4. New deprecation
5. Feature removal
6. Security fix
7. Performance improvement
8. Other
```
The entry filename is based on the name of the current Git branch. If you run the command above on a branch called feature/hey-dz, it will generate a changelogs/unreleased/feature-hey-dz.yml file.

The command will output the path of the generated file and its contents:
```
create changelogs/unreleased/my-feature.yml
---
title: Hey DZ, I added a feature to GitLab!
merge_request:
author:
type:
```

#### Arguments
|Argument|	Shorthand|	Purpose|
|---|---|---|
|--amend| |	 	Amend the previous commit|
|--force|	-f|	Overwrite an existing entry|
|--merge-request|	-m|	Set merge request ID|
|--dry-run|	-n|	Don’t actually write anything, just print|
|--git-username|	-u|	Use Git user.name configuration as the author|
|--type|	-t|	The category of the change, valid options are: added, fixed, changed, deprecated, removed, security, performance, other|
|--help|	-h|	Print help message|

##### --amend
You can pass the --amend argument to automatically stage the generated file and amend it to the previous commit.

If you use --amend and don’t provide a title, it will automatically use the “subject” of the previous commit, which is the first line of the commit message:
```
$ git show --oneline
ab88683 Added an awesome new feature to GitLab

$ scripts/changelog --amend
create changelogs/unreleased/feature-hey-dz.yml
---
title: Added an awesome new feature to GitLab
merge_request:
author:
type:
```
#### --force or -f
Use --force or -f to overwrite an existing changelog entry if it already exists.

```
$ scripts/changelog 'Hey DZ, I added a feature to GitLab!'
error changelogs/unreleased/feature-hey-dz.yml already exists! Use `--force` to overwrite.

$ scripts/changelog 'Hey DZ, I added a feature to GitLab!' --force
create changelogs/unreleased/feature-hey-dz.yml
---
title: Hey DZ, I added a feature to GitLab!
merge_request: 1983
author:
type:
```

####--merge-request or -m
Use the --merge-request or -m argument to provide the merge_request value:

```
$ scripts/changelog 'Hey DZ, I added a feature to GitLab!' -m 1983
create changelogs/unreleased/feature-hey-dz.yml
---
title: Hey DZ, I added a feature to GitLab!
merge_request: 1983
author:
type:
```

#### --dry-run or -n
Use the --dry-run or -n argument to prevent actually writing or committing anything:

```
$ scripts/changelog --amend --dry-run
create changelogs/unreleased/feature-hey-dz.yml
---
title: Added an awesome new feature to GitLab
merge_request:
author:
type:

$ ls changelogs/unreleased/
```

#### --git-username or -u
Use the --git-username or -u argument to automatically fill in the author value with your configured Git user.name value:

```
$ git config user.name
Jane Doe

$ scripts/changelog -u 'Hey DZ, I added a feature to GitLab!'
create changelogs/unreleased/feature-hey-dz.yml
---
title: Hey DZ, I added a feature to GitLab!
merge_request:
author: Jane Doe
type:
```

#### --type or -t
Use the --type or -t argument to provide the type value:

```
$ bin/changelog 'Hey DZ, I added a feature to GitLab!' -t added
create changelogs/unreleased/feature-hey-dz.yml
---
title: Hey DZ, I added a feature to GitLab!
merge_request:
author:
type: added
```


