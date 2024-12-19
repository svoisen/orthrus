# Orthrus

Orthrus is a minimalist static site generator for converting
markdown files to both HTML and gemtext, allowing the same content to be
published on both the web and [gemini](https://geminiprotocol.net).

## Usage

```
go build
./orthrus [-c path/to/config.toml] [build|serve]
```

The `--config` or `-c` argument is optional and will default to `./config.toml`
if not supplied.

The `build` and `serve` subcommands are optional, and will default to `build` if
none is provided.

The `build` subcommand will build all HTML and gemtext content, then exit.

The `serve` subcommand will run local servers for web and gemini, watching
changes to any markdown files and templates, then rebuilding content as needed. 

Each server will run on the ports defined in the `config.toml`. If no ports are 
defined, the servers default to the following:

- Gemini: port 1965
- Web: port 8080

## HTML templates

Orthrus supports templates using go's standard templating language. For
information on how to write templates, [see the documentation for the
`html/template` package](https://pkg.go.dev/html/template) of the go standard library.

Orthrus expects all templates to use the `.tmpl` file extension.

## Configuration file

See `example/config.toml` for an example configuration file including
comments on the various options.

