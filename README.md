# ωBNF super awesome parser engine

[![GitHub Actions Go status](https://github.com/arr-ai/wbnf/workflows/Go/badge.svg)](.)


# Grammar Syntax Guid

[![Example Syntax](examples/wbnf.txt)](.)

## The basics

A ωBNF grammar files consists of an unordered list of rules (called *productions*) or *comments*.

### Comments
A Comment can be either C style `// This is a comment to the end of the line` or C++ style ` /* This is a comment which may span multiple lines */` 

### Rules/Productions

A rule is defined in terms of *terms*, or *terminals* in the form of `NAME -> TERM+ ;`
 -  `a -> b;` indicates a *rule* named `a` which is made up of single *term* (in this case the *rule* `b`)
 -  `a -> "Token";` indicates a *rule* named `a` which is made of a single `terminal` (in this case the string `Token`)
 -  `a -> /{\d+};` indicates a *rule* named `a` which is made of a single `terminal` (in this case the regex `\d+`)

### Terminals

- Strings - Quoted text which must be present in the input text, may be quoted by `"` or `'` or `
- Regular Expressions - in the form `/{RE}` where RE is the expression to match. the entire match will be consumed


### Expressions

*term*s can be grouped in various ways to build up *rules*.

- No grouping

    `a -> left /{[+-*/]} right`
    
    This rule requires a `left` followed by one of the math symbols followed by a `right`
    
- Choices

    `a -> ( "hello" | "goodbye") name`
    
    This rule requires either the string `hello` or `goodbye` followed by a `name`
    
- Simple Multiplicity

    `+`, `?`, and `*` may follow any *term* to indicate the required multiplicity of the term.
    - `+` indicates 1 or more
    - `?` indicates 0 or 1
    - `*` indicates 0 or more
    
    `a -> b* ("x" | "y")+` requires any amount of `b` followed by at least one of either `"x"` or `"y"`
   
- Delimited repetition

    One of the design goals of the grammar is to minimise the amount of repetition in each *term*. 
    If we wanted to create a rule to accept a comma separated list of words the simple version would be 
    ```
    word -> /{\w*};
    csv -> word ("," word)*;
    ```
    The *rule* `word` appears 3 times in that tiny snippet! This can be eliminated with the use of the `:` operator after a term:
    `csv -> /{\w*}:",";` expresses the same *rule*. (More on this operator below)

- Stacks (or operator precedence)

   Languages often require some way to define the order of operations (remember BODMAS from school?).
   
   The simple form of a math parser would include something like:
   ```
   expr -> braces+;
   braces -> ("(" multdiv+ ")") | multdiv;
   multdiv -> addsum (("*" | "/") addsum)*;
   addsum -> number (("+" | "-") number)*;
   number -> /{\d+};
    ```
    
    This again has heaps of repetition both in each *rule* and between *rules* (as each refers to the next in the precedence order).
    
    The `^` operator can help with this (newlines are generally ignored by the parser) ->
    ```
    expr -> "(" expr ")"   
          ^ expr:/{[*/]}
          ^ expr:/{[+-]}
          ^ /{\d+};
    ```
    
    Each line of the stack implicitly refers to the next line down. *terms* further along have a higher precedence
    than earlier *terms*.
    
    This should parse the expression `1 + 2 * 3` as the following nodes:
          ```
          
            1 "+"
                \--  2 "*" 3
          ```
          giving the result 7
    If the rule was defined with `|` instead of `^` the above expression would have been parsed as:
        ```
        
            (1 "+" 2) "*" 3 
        ```
          giving the result 9