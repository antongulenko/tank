#!/bin/bash

function kill_xboxdrv() {
    read
    pkill xboxdrv
}

xboxdrv -d |& grep ERROR | kill_xboxdrv

