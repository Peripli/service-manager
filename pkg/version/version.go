package version

import "github.com/sirupsen/logrus"

// GitCommit is the commit id, injected by the build
var GitCommit string

// Version is the SM version, injected by the build
var Version string

// Log writes the Service Manager version info in the log
func Log() {
	logrus.Infof("Service Manager Version: %s (%s)", Version, GitCommit)
}
