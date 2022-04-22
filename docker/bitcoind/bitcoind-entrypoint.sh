#!/bin/sh
# based on https://github.com/ruimarinho/docker-bitcoin-core/blob/master/0.20/docker-entrypoint.sh
set -e

NETWORK_ARG=""
case "$BITCOIN_NETWORK" in
  mainnet)
    ;;

  testnet)
    NETWORK_ARG="-testnet"
    ;;

  regtest)
    NETWORK_ARG="-regtest"
    ;;

  *)
    echo Unexpected value for environment variable BITCOIN_NETWORK: "${BITCOIN_NETWORK:-empty}" >&2
    echo Must be one of: mainnet, testnet, regtest >&2
    exit 1
    ;;
esac

if [ $(echo "$1" | cut -c1) = "-" ]; then
  echo "$0: assuming arguments for bitcoind"

  set -- bitcoind $NETWORK_ARG "$@"
fi


if [ $(echo "$1" | cut -c1) = "-" ] || [ "$1" = "bitcoind" ]; then
  mkdir -p "$BITCOIN_DATA"
  chmod 700 "$BITCOIN_DATA"
  chown -R bitcoin "$BITCOIN_DATA"

  echo "$0: setting data directory to $BITCOIN_DATA"

  set -- "$@" -datadir="$BITCOIN_DATA"
fi

if [ "$1" = "bitcoind" ] || [ "$1" = "bitcoin-cli" ] || [ "$1" = "bitcoin-tx" ]; then
  echo
  exec gosu bitcoin "$@"
fi

echo
exec "$@"