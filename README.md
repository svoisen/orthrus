# Ibeji

Ibeji is a minimalist static site generator for converting
markdown files to both HTML and gemtext, allowing the same content to be
published on both the web and [gemini](https://geminiprotocol.net).

## Usage

```
go build
./ibeji [-c path/to/config.toml] [build|serve]
```

The `--config` or `-c` argument is optional and will default to `./config.toml`
if not supplied.

The `build` and `serve` subcommands are optional, and will default to `build` if
none is provided.

The `build` subcommand will build all HTML and gemtext content, then exit.

The `serve` subcommand will run local servers for web and gemini, watching
changes to any markdown files and rebuilding content as needed.

## HTML templates

Ibeji expects a single entry template file called `base.tmpl` written using
standard Go template syntax. This template file should be placed in the
templates directory defined in the `config.toml` file.

## Configuration file

The TOML configuration file supports the following options:

- `HostName`: The hostname to use when running the local server. Defaults to `localhost`.
- `GeminiPort`: The port to use for gemini when running the local server. Defaults to the standard gemini port 1965.
- `WebPort`: The port to use for HTTP when running the local server. Defaults to
  8080.
- `MarkdownDir`: The path to the directory of markdown content.
- `GeminiCertStore`: The path to use for storing self-signed certificates for
the gemini server.

## Ibeji?

Ibeji is the name of a pair of divine twins in the Yoruba religion of the Yoruba
people.
