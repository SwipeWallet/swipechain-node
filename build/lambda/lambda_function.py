import json
import requests
import boto3

prefix = 'node_ip_'
s3_resource = boto3.resource('s3')
buckets = ['chaosnet-seed.thorchain.info']


def get_rpc_url(ip_addr, path):
    # change port to 26657 for TESTNET
    return 'http://' + ip_addr + ':27147' + path


def get_api_url(ip_addr, path):
    return 'http://' + ip_addr + ':1317' + path


def get_new_ip_list(ip_addr):
    results = []
    try:
        r = requests.get(
            get_api_url(ip_addr, "/thorchain/nodeaccounts"), timeout=3
        )
        peers = [x['ip_address'] for x in r.json() if x['status'] == "active"]
        peers.append(ip_addr)
        peers = list(set(peers))  # uniqify
    except Exception as e:
        print(e)
        print("Node not responding " + ip_addr)
        return results

    # filter nodes that are not "caught up"
    for peer in peers:
        try:
            resp = requests.get(get_rpc_url(peer, "/status"), timeout=3)
            if not resp.json()['result']['sync_info']['catching_up']:
                results.append(peer)
        except Exception as e:
            print(e)
            print("Node not responding " + peer)
    return results


def lambda_handler(event, context):
    try:
        for bucket in buckets:
            thorchain_bucket = s3_resource.Bucket(bucket)
            for obj in thorchain_bucket.objects.all():
                if not obj.key.startswith(prefix):
                    continue
                body = obj.get()['Body'].read()
                ip_list = json.loads(body)
                new_ips = []
                for ip_addr in ip_list:
                    new_ips += get_new_ip_list(ip_addr)
                # only take duplicates confirmed by multiple nodes
                new_ips = set(ip for ip in new_ips if new_ips.count(ip) > 1)
                if len(new_ips) != 0:
                    new_body = json.dumps(sorted(new_ips))
                    print("New IPs list " + new_body)
                    obj.put(Body=new_body, ContentType='application/json')
        return {'message': 'successfully updated!'}
    except Exception as e:
        print(e)
        return {'message': 'exception occured!'}
