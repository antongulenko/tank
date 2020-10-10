#!/bin/bash

xboxdrv -d |& stdbuf -i0 -o0 -e0 grep ERROR | ( read xxx; pkill -SIGKILL xboxdrv; )

