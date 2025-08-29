#compdef fx

_fx() {
    local reply
    reply=("${(@f)$(COMP_ZSH="${LBUFFER}" fx)}")
    if (( ${#reply} )); then
        local -a insert display
        for line in "${reply[@]}"; do
            display+=("${line%%$'\t'*}")
            insert+=("${line#*$'\t'}")
        done
        compadd -f -S '' -d display -a insert
    fi
}

if [ "$funcstack[1]" = "_fx" ]; then
    _fx "$@"
else
    compdef _fx fx
fi
