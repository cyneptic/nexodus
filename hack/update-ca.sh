#!/bin/sh
set -e

if [ -f /.certs/rootCA.pem ]; then
  if [ -x /usr/sbin/update-ca-certificates ]; then
    cp /.certs/rootCA.pem /usr/local/share/ca-certificates/rootCA.crt
    /usr/sbin/update-ca-certificates 2> /dev/null > /dev/null
  elif [ -x /usr/bin/update-ca-trust ]; then
    cp ./.certs/rootCA.pem /etc/pki/ca-trust/source/anchors/rootCA.crt
    /usr/bin/update-ca-trust 2> /dev/null > /dev/null
  else
    echo "error: unable to add root CA certificate"
    exit 1
  fi
fi

if [ -n "$1" ] && [ "$1" != "prod" ]; then
  exec "$@"
fi

if [ "$1" = "prod" ]; then
  cat << EOF > ~/.bash_history
NEXD_LOGLEVEL=debug nexd https://qa.nexodus.io
NEXD_LOGLEVEL=debug nexd https://try.nexodus.io
nexd https://qa.nexodus.io
nexd https://try.nexodus.io
EOF

  cat << EOF > ~/.motd

To connect this container to the nexodus network, try running:

    nexd https://try.nexodus.io

Press the up arrow to get this command from bash history.

EOF

else
  cat << EOF > ~/.bash_history
NEXD_LOGLEVEL=debug nexd https://try.nexodus.io
NEXD_LOGLEVEL=debug nexd https://qa.nexodus.io
NEXD_LOGLEVEL=debug nexd --username admin --password floofykittens https://try.nexodus.127.0.0.1.nip.io
nexd https://try.nexodus.io
nexd https://qa.nexodus.io
nexd --username admin --password floofykittens https://try.nexodus.127.0.0.1.nip.io
EOF

  cat << EOF > ~/.motd

To connect this container to the nexodus network, try running:

    nexd --username admin --password floofykittens https://try.nexodus.127.0.0.1.nip.io

Commands for using a dev service, qa.nexodus.io, or try.nexodus.io are in bash history.

Press the up arrow to find these commands.

EOF
fi

tmux new-session -s nexd-session -d
tmux send-keys "cat ~/.motd" "C-m"
exec tmux attach-session -t nexd-session
