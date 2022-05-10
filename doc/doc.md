# Documentation

The **fx** can work in two modes: as a reducer or an interactive viewer.

To start the interactive mode pipe a JSON into **fx**:

```sh
$ curl ... | fx
```

Or you can pass a filename as the first parameter:

```sh
$ fx data.json
```

## Reducers

Use [JavaScript](js.md), [Python](python.md), or [Ruby](ruby.md).

## Streaming mode

The **fx** supports line-delimited JSON streaming or concatenated JSON streaming.

```sh
$ echo '
> {"message": "hello"}
> {"message": "world!"}
> ' | fx .message
hello
world!
```

## Interactive mode

Type `?` to see the full list of available shortcuts while in the interactive mode.

### Search

Press `/` and type regexp pattern to search in the current JSON. 
Search is performed on the internal representation of the JSON without newlines.

Type `n` to jump to the next result, and `N` to the previous

### Selecting text

You can't just select text in fx. This is due to the fact that all mouse events are 
redirected to stdin. To be able to select again you need to instruct your terminal 
not to do it. This can be done by holding special keys while selecting:

|       Key        |   Terminal    |
|------------------|---------------|
| `Option`+`Mouse` | iTerm2, Hyper |
| `Fn`+`Mouse`     | Terminal.app  |
| `Shift`+`Mouse`  | Linux         |


## Configs

Next configs available for **fx** via environment variables.

| Name           | Values                                              | Description                                           |
|----------------|-----------------------------------------------------|-------------------------------------------------------|
| `FX_LANG`      | `js` (default), `node`, `python`, `python3`, `ruby` | Reducer type.                                         |
| `FX_THEME`     | `0` (disable colors), `1` (default), `2..9`         | Color theme.                                          |
| `FX_SHOW_SIZE` | `true` or `false` (default)                         | Show size of arrays and object in collapsed previews. |
