#compdef fx

_fx() {
    local -a reply
    reply=("${(@f)$(COMP_ZSH="${LBUFFER}" fx)}")
    if (( ${#reply} )); then
        local -a insert_files display_files insert_other display_other
        local line display rest value typ
        
        for line in "${reply[@]}"; do
            display="${line%%$'\t'*}"
            rest="${line#*$'\t'}"
            value="${rest%%$'\t'*}"
            typ="${rest#*$'\t'}"

            if [[ "$typ" == "file" ]]; then
                display_files+=("$display")
                insert_files+=("$value")
            else
                display_other+=("$display")
                insert_other+=("$value")
            fi
        done

        if (( ${#insert_files} )); then
            compadd -f -d display_files -a insert_files
        fi
        if (( ${#insert_other} )); then
            compadd -S '' -d display_other -a insert_other
        fi
    fi
}

if [ "$funcstack[1]" = "_fx" ]; then
    _fx "$@"
else
    compdef _fx fx
fi
