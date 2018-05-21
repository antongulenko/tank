#!/bin/bash

search_root='/home/anton/wifi-credentials'
file_name='wifi-credentials*.txt'
declare -A connect_attempts
max_attempts=3

mount_target="$search_root/mounts"
mount_devices='.*/mmcblk[0-9]+p[0-9]+'
mount_search='/dev'
function try_mounts() {
    for i in `find "$mount_search" -regex "$mount_devices"`; do
        if ! mount | grep "$i" &> /dev/null; then
            target="$mount_target/$(basename "$i")"
            mkdir -p "$target"
            echo "Trying to mount $i to $target..."
            mount "$i" "$target"
        fi
    done
}

function do_connect() {
    ssid="$1"
    password="$2"

    show_output=$(nmcli connection show "$ssid")
    if [ $? = 0 ]; then
        if echo "$show_output" | grep GENERAL.STATE | grep activated &> /dev/null; then
            echo "Connection '$ssid' already exists and is connected!"
            return
        else
            echo "Connection '$ssid' already exists but is not connected, trying to connect..."
            if nmcli connection up "$ssid"; then
                echo "Successfully connected to $ssid"
                return
            else
                fails=${connect_attempts["$ssid"]}
                test -z "$fails" && fails=0
                fails=$((fails+1))
                connect_attempts["$ssid"]=$fails
                echo "Connection to $ssid failed for the $fails. time"
                if [ "$fails" -lt "$max_attempts" ]; then
                    return
                else
                    echo "Connection failed too often, deleting and re-creating with current credentials..."
                    nmcli connection delete "$ssid"
                fi
            fi
        fi
    fi
    echo "Trying to connect to WIFI $ssid with password $password"
    nmcli device wifi rescan ssid "$ssid"
    nmcli device wifi connect "$ssid" password "$password"
}

function check_wifi_credentials() {
    files=$(find "$search_root" -name "$file_name")
    test -z "$files" && { echo "No files named '$file_name' found in $search_root"; return; }
    echo "Reading wifi file(s) with WIFI credentials:" $files
    for i in $files; do
        IFS=$'\r\n' GLOBIGNORE='*' command eval "lines=(\$(cat '$i'))"
        if [ "${#lines[@]}" -ne 2 ]; then
            echo "Ignoring $i: contains ${#lines[@]} line(s), but I need exactly 2"
        else
            ssid="${lines[0]}"
            password="${lines[1]}"
            do_connect "$ssid" "$password"
        fi
    done
}

while true; do
    try_mounts
    check_wifi_credentials
    sleep 3
done

