How to Become a SwipeNode
==============================

## System setup
To setup your server to run full SwipeNode

### Docker Setup
TODO: Docker documentation

### Linux Setup
This documentation explains how to setup a Linux server as a SwipeNode
manually.

Before SwipeNode do anything, lets get the source code and build our binaries.
```bash
git clone https://github.com/SwipeWallet/swipechain-node.git
make install
```

Next, SwipeNode will need to setup the Binance full node....
#### Binance
Binance themselves provide documentation. Please follow their
[documentation](https://docs.binance.org/fullnode.html).

Alternatively, you can use a pre made [docker
image](https://github.com/varnav/binance-node-docker) to simplfy it. 

Wait until your Binance full node is caught up before continuing onto the next
sections.

#### Observer
TODO

#### Signer
TODO

#### Swiped
To setup `swiped`, we'll need to run the following commands.

```bash
swiped init local --chain-id swipechain

swipecli keys add operator
swipecli keys add observer

swipecli config chain-id swipechain
swipecli config output json
swipecli config indent true
swipecli config trust-node true
```

Next, SwipeNode need to get the genesis file of SwipeChain.
For testnet, run...
```bash
curl
https://github.com/SwipeWallet/swipechain-node/raw/master/genesis/testnet.json -o ~/.swiped/config/genesis.json
```

For mainnet, run...
```bash
Coming soon
```

Validate your genesis file is valid.
```bash
swiped validate-genesis
```

You can now start your `swiped` process

```bash
swiped start --rpc.laddr tcp://0.0.0.0:26657
```

#### Rest Server
To start the rest API of your `swiped` daemon, run the following...

```bash
swipecli rest-server --laddr tcp://0.0.0.0:1317
```


## Bonding
In order to become a validator, you must bond the minimum amount of sxp to a
`swipe` address. 

To do so, send your sxp to the SwipeChain with a specific memo. You will need
your swipe address to do so. You can retrieve that via...
```bash
swipecli keys show operator --address
```

Once you have your address, include in your memo to Swipechain
```
BOND:<address>
```

Once you have done that, you can then use the `swipecli` to
register your other addresses.

```bash
echo password | swipecli tx swipechain set-node-keys $(swipecli keys show swipechain --pubkey) $(swipecli keys show swipechain --pubkey) $(swiped tendermint show-validator) --from swipechain --yes
```

Once you have done this, your node is ready to be rotated into the active
group of validators.
