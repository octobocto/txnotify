set -e
set -o pipefail

if test -e .env; then
    # load .env values into current environment
    set -o allexport
    source .env
    set +o allexport
fi

exe() { echo "\$" "$@" ; "$@" ; echo ;  }
error() { echo "error:" "$@"; exit 1;  }

COMPOSE="docker-compose -f docker-compose.yml -f docker-compose.prod.yml"

required_envs=(SERVER_NAME BITCOIN_NETWORK EMAIL_PASSWORD)

for env in "${required_envs[@]}"; do
      if [ -z  "${!env}"  ]; then
          error environment variable "env" is not set
      fi
done

# Now that we've checked all environment variables, it should be an
# error to reference one that hasn't been passed
set -u

# start bitcoind early, to let it sync
exe $COMPOSE up --detach --remove-orphans bitcoind

exe $COMPOSE build --parallel

if [[ $SERVER_NAME != localhost  ]] ; then
      exe env DEPLOY=1 RENEW_CERTBOT=0 ./scripts/init-letsencrypt.sh
fi

exe $COMPOSE up --detach

check_can_reach_api() {
      OUT=$(mktemp)

    for i in $(seq 1 30); do
        INSECURE_ARG=""
        if [[ "$SERVER_NAME" == localhost  ]]; then
            INSECURE_ARG="--insecure"
        fi

        # shellcheck disable=SC2065
        if curl $INSECURE_ARG --fail "https://api.${SERVER_NAME}/notifications" > /dev/null 2> "$OUT"; then
            echo Reached server at "https://api.${SERVER_NAME}/notifications"
            return 0
        fi
        sleep 0.5
    done

    error could not reach server at "https://api.${SERVER_NAME}/notifications": "$(cat "$OUT")"
}

exe sleep 10

exe $COMPOSE restart nginx

stopped_services=""
for i in $(seq 1 5); do
    stopped_services=$($COMPOSE ps --services --filter status=stopped)
    if [ "$stopped_services" == ""  ]; then
        echo All services up and running 👍

        check_can_reach_api

        exit 0
    fi

    echo Trying to restart stopped services "$stopped_services"
    exe $COMPOSE up -d $stopped_services
done

error "the following services did not start: $(tr '\n' ' ' <<< "$stopped_services")"


