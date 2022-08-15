# Logging Guidelines

Service Manager uses [logrus](https://github.com/sirupsen/logrus) library for logging.

The current context should be provided to the logger in order to include context information.
One example of such information is the correlation id, which allows correlating log messages that are associated wih one operation.

Usually you access the logger like this:
```go
import (
  "context"

  "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
)

func doSomething(ctx context.Context) {
  // ...
  log.C(ctx).Infof("Did something with %s", name)
  // ...
}
```

In rare cases, when there is no context, you can access the logger like this:
```go
log.D().Info("No context info here")
```

## Log Level
The default log level is defined by `log.level` configuration.

There are two options to change the log level (and other configuration) without restart:
* By changing _application.yml_ on the file system where Service Manager is running, e.g. inside the container.
Note that if the log level is set via environment variable, it will override the value in the file.
* Via `/v1/config/logging` endpoint. See [api/configuration/controller.go](api/configuration/controller.go) for details.

Note that the log level is not stored in a central place.
So, if Service Manager is running with multiple instances, the log level should be set separately for each one.

### Error
Use _error_ level when something is wrong and the current operation/request cannot complete successfully.
In such situations you usually have an `error` object.

You usually log an error like this:
```go
log.C(ctx).WithError(err).Error("Could not do X")
```

If you have an `error`, **either return it or log it**, but not both as this leads to log duplication.
The central error handler (`util.WriteError`) logs all errors returned from request processing, so usually you don't need to log them again.

### Warning
Use _warning_ level when something is wrong but is not a problem for the current operation/request.
Still it might be a problem for another operation.

### Info
Should provide enough information to tell what Service Manager did and why.

Any actions performed by Service Manager concerning external entities should be logged at _info_ level.
Examples of such actions are changes to brokers / platforms / visibilities.

_Info_ messages should be understandable by people who are familiar with Service Manager concepts
but are not involved with its development. So these messages should not contain code-level details.
Strive to use the same terminology as in Service Manager documentation instead of internal abbreviations and terms.

Whenever possible _info_ messages should be normal english sentences.
For example instead of:
```go
log.C(ctx).Infof("Registered broker - name: %s URL: %s", name, url)
```
use
```go
log.C(ctx).Infof("Registered service broker %s with URL %s", name, url)
```

### Debug
Use _debug_ level to capture data that could be useful for troubleshooting.
Think, if you have to debug the code, which values would be most useful and log them.
These messages are intended for Service Manager developers
Keep in mind that **debugging in production is rarely possible**.

On the other hand do not think "the more the better".
Flooding the log with tons of messages makes it harder to understand.
Also it increases the chance of loosing log messages in case logging rate is exceeded.

Be aware that some situations are not possible to reproduce.
So, setting log level to _debug_ and retrying might not help.
As described above, important messages should be logged on _info_ level.

## Security

:warning: Be careful not to log sensitive data like passwords, tokens, personal data, etc.
Pay special attention if you do some generic logging where you don't know the exact semantic of the values being logged.