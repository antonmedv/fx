package complete

func Zsh() string {
	return `_fx_complete() {
    if [[ $CURRENT -eq 2 ]]; then
        _files
    else
        reply="$(COMP_ZSH="${LBUFFER}" fx)"
        if [[ -n "$reply" ]]; then 
            _values=("${(@f)reply}")
            compadd -a _values
        fi
    fi
}

compdef _fx_complete fx
`
}
