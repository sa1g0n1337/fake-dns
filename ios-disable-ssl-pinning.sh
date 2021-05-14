#!/usr/bin/env bash
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"

pid="$(frida-ps -Ua | grep "$1" | cut -d ' ' -f 1)"

if [ "$pid" == "" ]; then
  echo 'Please open the app first.'
  exit 1
fi

frida -U "$pid" -l "$DIR/ios-flutter-disable-sslpinning.js" --no-pause
