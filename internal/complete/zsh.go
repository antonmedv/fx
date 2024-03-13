package complete

func Zsh() string {
	return `_fx_complete() {
    if [[ $CURRENT -eq 2 ]]; then
        _files
    else
        compadd $(COMP_ZSH="${LBUFFER}" fx)
    fi
}

compdef _fx_complete fx
`
}
