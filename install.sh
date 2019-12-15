#!/usr/bin/env bash

set -e
set -u
set -o pipefail

put() {
  echo -e "\e[1;32m$1\e[0m"
}

do_install() {
  cat <<'EOM'

  ______      _____           _        _ _
 |  ____|    |_   _|         | |      | | |
 | |____  __   | |  _ __  ___| |_ __ _| | | ___ _ __
 |  __\ \/ /   | | | '_ \/ __| __/ _` | | |/ _ \ '__|
 | |   >  <   _| |_| | | \__ \ || (_| | | |  __/ |
 |_|  /_/\_\ |_____|_| |_|___/\__\__,_|_|_|\___|_|

EOM

  platform=''
  if [[ "$OSTYPE" == "linux"* ]]; then
    platform='linux'
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    platform='macos'
  elif [[ "$OSTYPE" == "win"* ]]; then
    platform='win.exe'
  fi

  if test "x$platform" = "x"; then
    cat <<EOM
/=====================================\\
|      COULD NOT DETECT PLATFORM      |
\\=====================================/

Uh oh! We couldn't automatically detect your operating system.

Please, download fx from https://github.com/antonmedv/fx/releases manually.
EOM
    exit 2
  else
    put "Detected platform: $platform"
  fi

  put "Downloading latest fx-$platform.zip"
  curl -L "https://github.com/antonmedv/fx/releases/latest/download/fx-$platform.zip" > "fx-$platform.zip"

  put "Extracting fx-$platform.zip"
  unzip "fx-$platform.zip"
  rm "fx-$platform.zip"
  mv "fx-$platform" fx

  name=''
  read -p $'\e[1;35mWhould you like to move fx binary to /usr/local/bin?\e[0m [Y/n] ' -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    mv fx /usr/local/bin/fx
    name='fx'
  else
    name='./fx'
  fi

  cat <<EOM
Now you can start using fx.

    $ $name '.downloadCount + 1'

EOM
}

do_install
