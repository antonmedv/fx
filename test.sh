#!/usr/bin/env bash
set -euxo pipefail
alias fx='node index.js'
echo '[{"greeting": "hello world"}]' | fx
echo '{"key": "value"}' | fx "x => x.key"
echo '[1, 2, 3, 4, 5]' | fx "this.map(x => x * this.length)"
echo '[1, 2, 3, 4, 5]' | fx "for (let i of this) if (i % 2 == 0) yield i"
echo '{"items": ["foo", "bar"]}' | fx "this.items" "yield* this; yield 'baz'"
