#!/usr/bin/env python3

import re
import sys

if sys.argv[1] == '-i':
    sys.argv[1] =  sys.argv[2]

with sys.stdin if sys.argv[1] == '-' else open(sys.argv[1]) as input:
    result = re.sub(
        r"(?s)(<!--\s*INJECT:\s*(.*?)\s*-->\n).*?(\n<!--\s*/INJECT\s*-->)",
        lambda s:
            s[1]
            +
            re.sub(
                r"\$\{(.*?)}",
                lambda f: open(f[1]).read().rstrip(),
                s[2].replace(r"\n", "\n"),
            )
            +
            s[3],
        input.read(),
    )

with sys.stdout if sys.argv[2] == '-' else open(sys.argv[2], 'w') as output:
    output.write(result)
