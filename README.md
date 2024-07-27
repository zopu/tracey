# Tracey - A CLI X-Ray Viewer

A CLI tool for quickly viewing trace and log information collected in AWS X-Ray.

### Install
```
go install github.com/zopu/tracey/cmd/tracey@latest
```

### Configure

Tracey looks for a JSON config file in .config/tracey/tracey.json

Here's an example:
```
{
  "exclude_paths": ["^/health/?$"],
  "logs": {
    "groups": ["/aws/apprunner/MyApprunnerApp/.*/application""],
    "fields": [
      {
        "title": "Level",
        "query": ".level"
      },
      {
        "title": "Func",
        "query": ".func"
      },
      {
        "title": "Message/URL",
        "query": "if .func == \"http.HandlerFunc.ServeHTTP\" then .url else .msg end"
      }
    ]
  }
}
```
- Log groups are specified as regexps that match log groups that should be scanned e.g. "/aws/apprunner/MyApp/.*/application"
- Fields specify what log data should be displayed. Tracey expects log data in json format, and uses gojq under the hood for its log query language.
