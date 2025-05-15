package complete

func Zsh() string {
	return `_fx_complete() {
	reply="$(COMP_ZSH="${LBUFFER}" fx)"
	if [[ -n "$reply" ]]; then 
		_values=("${(@f)reply}")
		compadd -f -S '' -a _values
    fi
}

compdef _fx_complete fx
`
}
