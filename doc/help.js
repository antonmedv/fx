'use strict'

function help() {
    return HELP_TEXT
}

module.exports = help

const PRG_HEADER = 'fx version ' + require('../package.json').version + '\n'

const HELP_TEXT = PRG_HEADER + `
Command-line JSON processing tool

USAGE
    fx [OPTIONS] [FILE]...

DESCRIPTION
    fx can work in two modes: cli and interactive. To start interactive mode
    pipe any JSON into fx:

    $ curl ... | fx

    Or pass a filename as the first parameter:

    $ fx my.json

    If any argument was passed, fx will apply it and prints to stdout.

    Full documentation could be found here: https://github.com/antonmedv/fx/blob/master/DOCS.md

    You may found what you can't just select text in fx. This is due the fact
    that all mouse events redirected to stdin. To be able select again you need
    instruct your terminal not to do it. This can be done by holding special
    keys while selecting:

    Option+Mouse - iTerm2, Hyper
    Fn+Mouse     - Terminal.app
    Shift+Mouse  - Linux

    Note what you can press "p"/"P" to print everything to stdout and select it there.

EXAMPLES
    $ echo '{"key": "value"}' | fx 'x => x.key'
    value

    $ echo '{"key": "value"}' | fx .key
    value

    $ echo '[1,2,3]' | fx 'this.map(x => x * 2)'
    [2, 4, 6]

    $ echo '{"items": ["one", "two"]}' | fx 'this.items' 'this[1]'
    two

    $ echo '{"count": 0}' | fx '{...this, count: 1}'
    {"count": 1}

    $ echo '{"foo": 1, "bar": 2}' | fx ?
    ["foo", "bar"]

OPTIONS
    --version
    --help

ENVIRONMENT
    You may find that sometimes, on really big JSON files, fx prints an error
    message like this:

    FATAL ERROR: JavaScript heap out of memory

    V8 limits memory usage to around 2 GB by default. You can increase the limit
    by putting this line in your *.profile*:

    export NODE_OPTIONS='--max-old-space-size=8192'

MORE INFORMATION
    https://github.com/antonmedv/fx/blob/master/DOCS.md
    https://fx.wtf/

`
