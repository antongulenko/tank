#!/bin/bash

xboxdrv -d |&\
    stdbuf -i0 -o0 -e0 grep 'USB read failure: 32: LIBUSB_TRANSFER_ERROR' |\
    ( read && { pkill -f tank-control-daemon; pkill -SIGKILL xboxdrv; }; )

