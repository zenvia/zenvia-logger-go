# Zenvia Logger for Go

Structured logging for Go services that writes the same [Logstash](https://www.elastic.co/logstash) JSON as
[`@zenvia/logger`](https://github.com/zenvia/zenvia-logger-node), so Go and Node services look the same in Datadog
(same fields, same levels, same environment variables). It is a thin layer over the standard `log/slog`, with no
dependencies.

## Install

```bash
go get github.com/zenvia/zenvia-logger-go
```

Requires Go 1.22 or later.

## Usage

```go
import (
	"log/slog"
	"os"

	logger "github.com/zenvia/zenvia-logger-go"
)

func main() {
	opts, err := logger.OptionsFromEnv(os.LookupEnv)
	if err != nil {
		panic(err)
	}
	slog.SetDefault(logger.New(os.Stdout, opts))

	slog.Info("event applied", "org", "org-a", "panicFlag", true)
}
```

Output:

```json
{"@timestamp":"2026-10-28T21:31:27.133Z","level":"INFO","message":"event applied","@version":1,"application":"api","environment":"stg","host":"pod-1","org":"org-a","panicFlag":true}
```

Extra key/values are flat top-level fields, as in the Node logger. Errors go in as an attribute, for example
`slog.Error("failed", "err", err)`.

## Trace id

Put the trace id of a request in the context and every record logged with that context carries it as `traceID`:

```go
ctx = logger.WithTraceID(ctx, id)
slog.InfoContext(ctx, "denied", "org", org) // {"...","traceID":"<id>"}
```

Use `slog.XxxContext` (not `slog.Xxx`) so the context reaches the logger. `logger.NewTraceID()` generates a UUID v4
for requests that arrive without one, as `@zenvia/logger` does. Reading the header is up to the server;
`Options.TraceHeader` holds its configured name (`X-TraceId` by default).

## Environment variables

`OptionsFromEnv` reads the variables of `@zenvia/logger`:

| Variable | Effect | Default |
|---|---|---|
| `APP_NAME` | `application` field | executable name |
| `NODE_ENV`, `APP_ENV` | `environment` field (first one set) | empty |
| `HOST`, `HOSTNAME` | `host` field (first one set) | OS hostname |
| `LOGGING_LEVEL` | `fatal`, `error`, `warn`, `info`, `debug` | `DEBUG` |
| `LOGGING_FORMATTER_DISABLED` | `true` writes plain text instead of JSON | `false` |
| `LOGGING_TRACE_HEADER` | header with the request trace id | `X-TraceId` |

`slog` has no fatal level: `fatal` is accepted and logs as `ERROR`; `verbose` and `silly` behave as `debug`.

## Never log secrets

Do not log tokens, the `Authorization` header or credentials. Logs go to a system with broader read access than the
service itself.

## License

[MIT](LICENSE.md)
