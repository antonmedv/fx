# Ruby Reducers

If any additional arguments was passed, **fx** converts it to a function which
takes the JSON as an argument named `x`.

```sh
export FX_LANG=ruby
```

Example:

```sh
fx data.json 'x.to_a.map {|x| x[1]}'
```

## Dot

Fx supports simple syntax for accessing data, which can be used with any `FX_LANG`.

```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx .foo[0].bar
value
```
