#!/usr/bin/env bash
frida -Uf "$1" -l ios-flutter-disable-sslpinning.js --no-pause
