# `github.com/paketo-buildpacks/libpak`
`libpak` is a Go library with useful functionality for building Paketo-style buildpacks.

## Usage
```
go get github.com/paketo-buildpacks/libpak
```

## Log Level Configuration

`libpak` supports configuring log output via the environment variable `BP_LOG_LEVEL`:

- `none`: Disables all log output (useful for runtime/exec.d scenarios)
- `error`: Only error logs are shown (suppresses info/debug)
- `info`: Default, shows info and error logs
- `debug`: Shows all logs, including debug

Alternatively, setting `BP_DEBUG` to any value is equivalent to `BP_LOG_LEVEL=debug` (for backward compatibility).

**Use Cases:**
- Use `BP_LOG_LEVEL=none` to suppress all logs in containers where clean stdout is required (e.g., MCP servers)
- Use `BP_LOG_LEVEL=error` to only see errors
- Default (`info`) is suitable for most build scenarios

**Notes:**
- `BP_DEBUG` takes precedence over `BP_LOG_LEVEL` if both are set
- Unknown or invalid values for `BP_LOG_LEVEL` default to `info`
- Setting `BP_LOG_LEVEL=none` during build disables all buildpack output (use with care)

## License
This library is released under version 2.0 of the [Apache License][a].

[a]: https://www.apache.org/licenses/LICENSE-2.0

