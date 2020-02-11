#!/bin/sh

out=wbnfgrammar.go

echo Generating $out

tmpfile=`mktemp`
go run .. gen --grammar ../examples/wbnf.wbnf --rootrule grammar --pkg wbnf > $tmpfile && mv $tmpfile $out &&

cat >> $out <<EOF

var grammarGrammarSrc = unfakeBackquote(\`
$(sed 's/`/‵/g' ../examples/wbnf.wbnf)
\`)
EOF