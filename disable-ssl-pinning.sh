#!/usr/bin/env bash
frida -Uf "$1" -l flutter-disable-sslpinning.js --no-pause
