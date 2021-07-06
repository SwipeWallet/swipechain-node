import argparse
import json
import logging
import os
import socket
import sys
import time
from contextlib import closing
from urllib.parse import urlparse

import requests
from web3 import Web3, HTTPProvider
from web3.middleware import geth_poa_middleware

logging.basicConfig(level=logging.INFO)


class TestEthereum:
    """
    An client implementation for a localnet/rinkebye/ropston Ethereum server
    """

    default_gas = 65000
    gas_price = 1
    name = "Ethereum"
    gas_per_byte = 68
    chain = "ETH"
    passphrase = "the-passphrase"
    seed = "SEED"
    stake = "STAKE"
    tokens = dict()
    zero_address = "0x0000000000000000000000000000000000000000"

    private_keys = [
        "ef235aacf90d9f4aadd8c92e4b2562e1d9eb97f0df9ba3b508258739cb013db2",
        "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032",
        "e810f1d7d6691b4a7a73476f3543bd87d601f9a53e7faf670eac2c5b517d83bf",
        "a96e62ed3955e65be32703f12d87b6b5cf26039ecfa948dc5107a495418e5330",
        "9294f4d108465fd293f7fe299e6923ef71a77f2cb1eb6d4394839c64ec25d5c0",
    ]

    def __init__(self, base_url):
        self.url = base_url
        self.web3 = Web3(HTTPProvider(base_url))
        self.web3.middleware_onion.inject(geth_poa_middleware, layer=0)
        # import all the private keys
        for key in self.private_keys:
            payload = json.dumps(
                {"method": "personal_importRawKey", "params": [key, self.passphrase]}
            )
            headers = {"content-type": "application/json", "cache-control": "no-cache"}
            try:
                requests.request("POST", base_url, data=payload, headers=headers)
            except requests.exceptions.RequestException as e:
                logging.error(f"{e}")

        self.accounts = self.web3.geth.personal.list_accounts()
        self.web3.eth.defaultAccount = self.accounts[0]
        self.web3.geth.personal.unlock_account(
            self.accounts[0], ""
        )

        logging.info(f"balance: {self.web3.eth.getBalance(self.accounts[0])}")
        for x in range(1, len(self.accounts)):
            self.fund_account(self.accounts[0], self.accounts[1], 92000000000000000000, x == 1)

        self.web3.eth.defaultAccount = self.accounts[1]
        self.web3.geth.personal.unlock_account(
            self.web3.eth.defaultAccount, self.passphrase
        )

    def fund_account(self, from_address, to_address, amount, wait_for_commit):
        tx = {
            "from": Web3.toChecksumAddress(from_address),
            "to": Web3.toChecksumAddress(to_address),
            "value": amount,
            "gas": self.calculate_gas(""),
        }
        if wait_for_commit:
            # wait for the transaction to be mined
            tx_hash = self.web3.geth.personal.sendTransaction(tx, "")
            self.web3.eth.waitForTransactionReceipt(tx_hash)

    def deploy_init_contracts(self):
        self.vault = self.deploy_vault()
        token = self.deploy_token()
        symbol = token.functions.symbol().call()
        self.tokens[symbol] = token

    def calculate_gas(self, msg):
        return self.default_gas + self.gas_per_byte * len(msg)

    def deploy_token(self, abi_file="data_token.json", bytecode_file="data_token.txt"):
        abi = json.load(open(os.path.join(os.path.dirname(__file__), abi_file)))
        bytecode = open(os.path.join(os.path.dirname(__file__), bytecode_file), "r").read()
        token = self.web3.eth.contract(abi=abi, bytecode=bytecode)
        tx_hash = token.constructor().transact()
        receipt = self.web3.eth.waitForTransactionReceipt(tx_hash)
        logging.info(f"Token Contract Address: {receipt.contractAddress}")
        return self.web3.eth.contract(address=receipt.contractAddress, abi=abi)

    def deploy_vault(self):
        abi = json.load(open(os.path.join(os.path.dirname(__file__), "data_vault.json")))
        bytecode = open(os.path.join(os.path.dirname(__file__), "data_vault.txt"), "r").read()
        vault = self.web3.eth.contract(abi=abi, bytecode=bytecode)
        # hard code here , as this only used for mock net test
        tx_hash = vault.constructor(Web3.toChecksumAddress("0x3155ba85d5f96b2d030a4966af206230e46849cb")).transact()
        receipt = self.web3.eth.waitForTransactionReceipt(tx_hash)
        logging.info(f"Vault Contract Address: {receipt.contractAddress}")
        return self.web3.eth.contract(address=receipt.contractAddress, abi=abi)


def check_socket(host, port):
    with closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
        if sock.connect_ex((host, port)) == 0:
            return True
        else:
            return False


def main():
    parser = argparse.ArgumentParser()
    # ethereum daemon address
    parser.add_argument(
        "--ethereum", default="", help="ethereum daemon address",
    )

    subparsers = parser.add_subparsers()

    deploy_parser = subparsers.add_parser('deploy')
    deploy_parser.set_defaults(name='deploy')
    deploy_parser.add_argument("--from_address", help="from address")

    args = parser.parse_args()
    defaultEth = "http://ethereum-localnet:8545"
    if "CI" in os.environ:
        defaultEth = "http://localhost:8545"
    if args.ethereum is None or args.ethereum == "":
        args.ethereum = defaultEth

    # check that the port is open
    t = urlparse(args.ethereum)
    for i in range(1, 30):
        if check_socket(t.hostname, t.port):
            time.sleep(5)
            break
        if i == 30:
            logging.info("Ethereum node does not appear to be running... exiting")
            sys.exit(1)
        time.sleep(1)

    test_ethereum = TestEthereum(args.ethereum)

    if args.name == "deploy":
        logging.info("Deploying contracts...")
        test_ethereum.deploy_init_contracts()

    logging.info("Done.")


if __name__ == "__main__":
    main()
