set -e 
set -o pipefail
set -u

if test -f "/root/certs/live/localhost/fullchain.pem"; then
    echo Certificate already exists, quitting
    exit 0
fi

CAROOT=/root/caroot mkcert -install 2>/dev/null
CAROOT=/root/caroot mkcert -key-file=/root/certs/live/localhost/privkey.pem \
    -cert-file=/root/certs/live/localhost/fullchain.pem localhost www.localhost api.localhost