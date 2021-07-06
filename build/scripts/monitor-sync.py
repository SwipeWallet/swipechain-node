#!/usr/bin/python

import logging
import os
import sys
import time

from tqdm import tqdm
import requests
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

# Init logging
logging.basicConfig(
    format="%(asctime)s | %(levelname).4s | %(message)s",
    level=os.environ.get("LOGLEVEL", "INFO"),
)

SLEEP = 5
RETRIES = 6

def retry(func, args=None, kwargs=None, max_tries=10):
    pass_on_args = args if args else []
    pass_on_kwargs = kwargs if kwargs else {}
    for i in range(max_tries):
        try:
            return func(*pass_on_args, **pass_on_kwargs)
            break
        except Exception as e:
            logging.error(f"retry failure ({i}): {e}")
            time.sleep(i)
            continue

def requests_retry_session(
    retries=6, backoff_factor=1, status_forcelist=(500, 502, 504), session=None,
):
    """
    Creates a request session that has auto retry
    """
    session = session or requests.Session()
    retry = Retry(
        total=retries,
        read=retries,
        connect=retries,
        backoff_factor=backoff_factor,
        status_forcelist=status_forcelist,
    )
    adapter = HTTPAdapter(max_retries=retry)
    session.mount("http://", adapter)
    session.mount("https://", adapter)
    return session


class HttpClient:
    """
    An generic http client
    """

    def __init__(self, base_url):
        self.base_url = base_url

    def get_url(self, path):
        """
        Get fully qualified url with given path
        """
        return self.base_url + path

    def fetch(self, path, args={}):
        """
        Make a get request
        """
        url = self.get_url(path)
        resp = requests_retry_session().get(url, params=args, timeout=5)
        resp.raise_for_status()
        return resp.json()


class Tendermint(HttpClient):
    """
    A local simple implementation of tendermint of THORCHAIN
    """

    def __init__(self, base):
        self.base_url = base


    def get_height(self):
        resp = self.fetch("/block")
        return int(resp['result']['block']['header']['height'])


def main():

    netNode = Tendermint("https://rpc.thorchain.info")
    myaddr = "http://localhost:26657"
    if len(sys.argv) > 1:
        myaddr = sys.argv[1]
    myNode = Tendermint(myaddr)

    pbar = tqdm(total=netNode.get_height())
    last_height = 0
    failures = 0

    while True:
        try:
            net_current_height = netNode.get_height()
            my_current_height = myNode.get_height()
            
            # update progress bar
            pbar.total = net_current_height
            pbar.update(my_current_height-last_height)

            # check that we've reached the tip
            if net_current_height == my_current_height:
                exit(0)

            # check if we're stuck
            if my_current_height <= last_height:
                failures += 1
                if failures >= 6:
                    logging.error('same block height, chain may be broken')
                    exit(1)

            last_height = my_current_height
            failures = 0

            time.sleep(SLEEP)

        except Exception:
            logging.error("Fatal error", exc_info=True)
            time.sleep(SLEEP)


if __name__ == "__main__":
    main()
