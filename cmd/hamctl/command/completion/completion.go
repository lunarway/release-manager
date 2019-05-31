package completion

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"
)

// FlagAnnotation adds a bash completion annotation to command c for flag f.
// completionFunc is the bash function identifier that should be called upon
// completion requests.
func FlagAnnotation(c *cobra.Command, f string, completionFunc string) {
	c.Flag(f).Annotations = map[string][]string{
		cobra.BashCompCustom: []string{completionFunc},
	}
}

// Hamctl contains bash completions for the hamctl command.
const Hamctl = `
__hamctl_get_environments()
{
	local template
	template=$'{{ range $k, $v := . }}{{ $k }} {{ end }}'
	local shuttle_out
	if shuttle_out=$(shuttle --skip-pull get k8s  --template="${template}" 2>/dev/null); then
		# remove "local" from possible environments as it has no use for hamctl
		shuttle_out=${shuttle_out[@]//local}
		COMPREPLY=( $( compgen -W "${shuttle_out[@]}" -- "$cur" ) )
	fi
}

__hamctl_get_namespaces()
{
	local template
	template="{{ range .items  }}{{ .metadata.name }} {{ end }}"
	local kubectl_out
	if kubectl_out=$(kubectl get -o template --template="${template}" namespace 2>/dev/null); then
		COMPREPLY=( $( compgen -W "${kubectl_out[*]}" -- "$cur" ) )
	fi
}

__hamctl_get_branches()
{
	local git_out
	if git_out=$(git branch --remote | grep -v HEAD | sed 's/[ \t*]origin\///' 2>/dev/null); then
		COMPREPLY=( $( compgen -W "${git_out[*]}" -- "$cur" ) )
	fi
}
`

// Zsh writes a zsh completion script that wraps the bash completion script.
//
// Copied from kubectl: https://github.com/kubernetes/kubernetes/blob/9c2df998af9eb565f11d42725dc77e9266483ffc/pkg/kubectl/cmd/completion/completion.go#L145
func Zsh(out io.Writer, hamctl *cobra.Command) error {
	zshHead := "#compdef hamctl\n"

	out.Write([]byte(zshHead))

	zshInitialization := `
__hamctl_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand
	source "$@"
}
__hamctl_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift
		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__hamctl_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}
__hamctl_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?
	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}
__hamctl_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}
__hamctl_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}
__hamctl_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}
__hamctl_filedir() {
	local RET OLD_IFS w qw
	__hamctl_debug "_filedir $@ cur=$cur"
	if [[ "$1" = \~* ]]; then
		# somehow does not work. Maybe, zsh does not call this at all
		eval echo "$1"
		return 0
	fi
	OLD_IFS="$IFS"
	IFS=$'\n'
	if [ "$1" = "-d" ]; then
		shift
		RET=( $(compgen -d) )
	else
		RET=( $(compgen -f) )
	fi
	IFS="$OLD_IFS"
	IFS="," __hamctl_debug "RET=${RET[@]} len=${#RET[@]}"
	for w in ${RET[@]}; do
		if [[ ! "${w}" = "${cur}"* ]]; then
			continue
		fi
		if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
			qw="$(__hamctl_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}
__hamctl_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
	printf %q "$1"
    fi
}
autoload -U +X bashcompinit && bashcompinit
# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --help 2>&1 | grep -q GNU; then
	LWORD='\<'
	RWORD='\>'
fi
__hamctl_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__hamctl_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__hamctl_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__hamctl_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__hamctl_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__hamctl_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/builtin declare/g" \
	-e "s/\\\$(type${RWORD}/\$(__hamctl_type/g" \
	<<'BASH_COMPLETION_EOF'
`
	out.Write([]byte(zshInitialization))

	buf := new(bytes.Buffer)
	hamctl.GenBashCompletion(buf)
	out.Write(buf.Bytes())

	zshTail := `
BASH_COMPLETION_EOF
}
__hamctl_bash_source <(__hamctl_convert_bash_to_zsh)
_complete hamctl 2>/dev/null
`
	out.Write([]byte(zshTail))
	return nil
}
