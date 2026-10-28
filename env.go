package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// OptionsFromEnv reads the same environment variables as @zenvia/logger:
//
//	APP_NAME                   "application" field; defaults to the executable name
//	NODE_ENV, APP_ENV          "environment" field (the first one set)
//	HOST, HOSTNAME             "host" field (the first one set); defaults to the OS hostname
//	LOGGING_LEVEL              fatal, error, warn, info, debug (default DEBUG)
//	LOGGING_FORMATTER_DISABLED true writes plain text instead of JSON (default false)
//	LOGGING_TRACE_HEADER       header with the request trace id (default X-TraceId)
//
// Use os.LookupEnv as lookup.
func OptionsFromEnv(lookup func(string) (string, bool)) (Options, error) {
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := lookup(k); ok && v != "" {
				return v
			}
		}
		return ""
	}
	or := func(v, d string) string {
		if v != "" {
			return v
		}
		return d
	}
	host, _ := os.Hostname()
	exe := "app"
	if len(os.Args) > 0 {
		exe = filepath.Base(os.Args[0])
	}

	opts := Options{
		Application: or(get("APP_NAME"), exe),
		Environment: get("NODE_ENV", "APP_ENV"),
		Host:        or(get("HOST", "HOSTNAME"), host),
		TraceHeader: or(get("LOGGING_TRACE_HEADER"), "X-TraceId"),
	}
	var err error
	if opts.Level, err = ParseLevel(or(get("LOGGING_LEVEL"), "DEBUG")); err != nil {
		return opts, fmt.Errorf("LOGGING_LEVEL: %w", err)
	}
	if v := get("LOGGING_FORMATTER_DISABLED"); v != "" {
		if opts.Text, err = strconv.ParseBool(v); err != nil {
			return opts, fmt.Errorf("LOGGING_FORMATTER_DISABLED: %q is not a boolean", v)
		}
	}
	return opts, nil
}
