package complete

func Zsh() string {
	return `_fx_complete() {
    if [[ $CURRENT -eq 2 ]]; then
        _files
    else
		_values=("${(@f)$(COMP_ZSH="${LBUFFER}" fx)}")
        compadd -a _values
    fi
}

compdef _fx_complete fx
`
}
