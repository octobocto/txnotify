#!/bin/bash
# This script is cribbed from https://medium.com/@pentacent/nginx-and-lets-encrypt-with-docker-in-less-than-5-minutes-b4b8a60d3a71
# and slightly changed to fit our needs

set -e
set -o pipefail
set -u

COMPOSE="docker-compose -f docker-compose.yml -f docker-compose.prod.yml"

if ! [ -x "$(command -v docker-compose)" ]; then
  echo 'Error: docker-compose is not installed.'
  exit 1
fi

if [ -z "${SERVER_NAME:-}" ]; then
  echo SERVER_NAME environment variable is empty!
  exit 1
fi

subdomains=(api frontend docs)
rsa_key_size=4096
data_path="./docker/certbot"
email="bo@jalborg.com"

# by default this script does a dry run
# set the environment variable DEPLOY to 1 to override
deploy=${DEPLOY:-0}

if [ "$RENEW_CERTBOT" != "1" ]; then
  exit
fi

OLD_PATH="/etc/letsencrypt/old"
echo "### Moving old certificates"
$COMPOSE run --rm --entrypoint "\
  mkdir -p $OLD_PATH && \
  mv /etc/letsencrypt/live/$SERVER_NAME $OLD_PATH/live/$SERVER_NAME && \
  mv /etc/letsencrypt/archive/$SERVER_NAME $OLD_PATH/archive/$SERVER_NAME && \
  mv /etc/letsencrypt/renewal/$SERVER_NAME.conf $OLD_PATH/renewal/$SERVER_NAME.conf" \
  certbot
echo

server_with_subs="$SERVER_NAME (subdomains: ${subdomains[*]})"

echo "### Creating dummy certificate for $server_with_subs..."
path="/etc/letsencrypt/live/$SERVER_NAME"
$COMPOSE run --rm --entrypoint "\
  openssl req -x509 -nodes -newkey rsa:1024 -days 1\
    -keyout '$path/privkey.pem' \
    -out '$path/fullchain.pem' \
    -subj '/CN=localhost'" certbot
echo


echo "### Starting nginx ..."
$COMPOSE up --force-recreate -d nginx
echo

echo "### Deleting dummy certificate for $server_with_subs..."
$COMPOSE run --rm --entrypoint "\
  rm -Rf /etc/letsencrypt/live/$SERVER_NAME && \
  rm -Rf /etc/letsencrypt/archive/$SERVER_NAME && \
  rm -Rf /etc/letsencrypt/renewal/$SERVER_NAME.conf" certbot
echo


echo "### Requesting Let's Encrypt certificate for $server_with_subs ..."
domain_args="-d $SERVER_NAME"
for domain in "${subdomains[@]}"; do
  domain_args="$domain_args -d $domain.$SERVER_NAME"
done

# Select appropriate email arg
case "$email" in
  "") email_arg="--register-unsafely-without-email" ;;
  *) email_arg="--email $email" ;;
esac

# Enable staging mode if needed
staging_arg=""
if [ "$deploy" != "1" ]; then
  echo "#### Running certbot in staging mode"
  staging_arg="--staging"
else
  echo "#### Running certbot in deployment mode"
fi

$COMPOSE run --rm --entrypoint "\
  certbot certonly --webroot -w /var/www/certbot \
    $staging_arg \
    $email_arg \
    $domain_args \
    --rsa-key-size $rsa_key_size \
    --agree-tos \
    --force-renewal" certbot
echo

echo "### Reloading nginx ..."
$COMPOSE exec nginx nginx -s reload
