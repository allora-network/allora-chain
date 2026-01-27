#!/bin/sh
rm -r ~/.relayer || true

export_env_vars() {
    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ ! "$line" =~ ^# && "$line" =~ = ]]; then
            var_name=$(echo "$line" | cut -d '=' -f 1)
            var_value=$(echo "$line" | cut -d '=' -f 2-)
            export "$var_name"="${var_value//\"/}"
        fi
    done < .env
}

export_env_vars

echo "$ALLORA_RELAYER_MNEMONIC"

rly config init
rly chains add-dir ./configs
rly paths add-dir ./paths

rly keys restore allora relayer "$ALLORA_RELAYER_MNEMONIC"
rly keys restore osmosis relayer "$OSMOSIS_RELAYER_MNEMONIC"

echo "Tx Link"
rly tx link demo -d -t 3s

echo "Start Demo"
rly start demo