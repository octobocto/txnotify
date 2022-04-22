#!/bin/bash

DEST_FILE=src/api/txnotify.tsx

node_modules/restful-react/dist/bin/restful-react.js import --file ../proto/txnotify.swagger.json --output $DEST_FILE

echo -e "/* eslint-disable */\n$(cat $DEST_FILE)" > $DEST_FILE

chmod a+rw $DEST_FILE
