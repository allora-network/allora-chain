#!/bin/bash

# This script will print a command to be used on a validator to create 
# the JSON file with the upgrade proposal details, which can be submitted
# to the network to trigger the upgrade process.

# Update the variables below to match current conditions and desired upgrade
current_verison=v0.5.0
new_version=v0.6.0
expedited=true
deposit=50000000uallo
num_minutes_between_proposal_and_upgrade=6
avg_block_time=5 # seconds
current_height=10000

# Calculating height
height=$((60*num_minutes_between_proposal_and_upgrade/avg_block_time+current_height))

# To get the authority address, run this command on the validator:
# allorad q upgrade authority | grep address | awk '{print $2}'
#
authority=allo1 # replace with the actual address

json='{"messages":[{"@type":"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade","authority":"'$authority'","plan":{"name":"'$new_version'","time":"0001-01-01T00:00:00Z","height":"'$height'","info":"","upgraded_client_state":null}}],"metadata":"ipfs://CID","deposit":"'$deposit'","title":"'$new_version'","summary":"Upgrade from '$current_verison' to '$new_version'","expedited":'$expedited'}'

echo "json='$json' && echo \"\$json\" | jq . > upgrade.json"
