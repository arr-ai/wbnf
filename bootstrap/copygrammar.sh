#!/bin/bash

out=wbnfgrammar.go
echo Generating $out
cat > $out <<EOF
package bootstrap

var grammarGrammarSrc = unfakeBackquote(\`
$(sed 's/`/‵/g' ../examples/wbnf.txt)
\`)
EOF
