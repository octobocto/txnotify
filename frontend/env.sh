#!/bin/sh
# cribbed from https://github.com/kunokdev/cra-runtime-environment-variables
echo "window.runtimeEnv = {" 
grep -v '^#' .env.runtime |\
    grep -v -e '^[[:space:]]*$' |\
    awk -F '=' '{ print "  " $1 ": \"" (ENVIRON[$1] ? ENVIRON[$1] : $2) "\"," }' 
    
echo "};"