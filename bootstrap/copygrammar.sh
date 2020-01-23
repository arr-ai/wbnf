#!/bin/bash

cat > wbnfgrammar.go <<EOF
package bootstrap
import "encoding/base64"

//go:generate sh copygrammar.sh

const grammarGrammarBase64 = \`
EOF
base64 ../examples/wbnf.txt >> wbnfgrammar.go

cat  >> wbnfgrammar.go <<EOF
\`

func grammarGrammarSrc() string {
	text, err := base64.StdEncoding.DecodeString(grammarGrammarBase64)
	if err != nil {
		panic(err)
	}
	return string(text)
}


EOF