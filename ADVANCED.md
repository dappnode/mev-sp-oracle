## Deployments

This repo contains not only the oracle software but also a docker-compose with all required components such as consensus, execution clients and prometheus/grafana to monitor the setup. Everything is provisioned, so you just need to do the following.

## Goerli Example

Running in `updater` mode:

```
./mev-sp-oracle \
--consensus-endpoint="http://127.0.0.1:5051" \
--execution-endpoint="http://127.0.0.1:8545" \
--pool-address=0xF21fbbA423f3a893A2402d68240B219308AbCA46 \
--updater-keystore-path=keystore-path \
--updater-keystore-pass=password-of-keystore
```

Running in `verifier` mode:

```
./mev-sp-oracle \
--consensus-endpoint="http://127.0.0.1:5051" \
--execution-endpoint="http://127.0.0.1:8545" \
--pool-address=0xF21fbbA423f3a893A2402d68240B219308AbCA46 \
--dry-run
```

### Goerli Testnet

```console
$ sudo openssl rand -hex 32 > jwtsecret
$ chmod 777 teku
```

Store in `.env` so that it's picked up by `docker-compose`. Notice that REGISTERED_RELAYS should be a comma separated list of relays that 
operate in the same network that the oracle is running on. The following is an example for goerli:

```
export NETWORK=goerli
export POOL_ADDRESS=0xF21fbbA423f3a893A2402d68240B219308AbCA46
export UPDATER_KEYSTORE_PASS=xxx
export REGISTERED_RELAYS=https://builder-relay-goerli.flashbots.net,https://bloxroute.max-profit.builder.goerli.blxrbdn.com,https://relay-goerli.edennetwork.io,https://goerli-relay.securerpc.com
```
And manually modify the following placeholders in `deployments/monitoring/prometheus-config.yml`:
* Change `OPERATOR_PLACEHOLDER` for your identifier eg `monitor: "oracle_someentity"`.
* Replace the `URL_PLACEHOLDER` (or whole url) to the one you want to push metrics to.
* Replace `USERNAME_PLACEHOLDER`.
* And `PASSWORD_PLACEHOLDER`.


```
docker-compose up -d
```

Use to check that all env variables were correctly replaced
```console
docker compose convert
```
