name             "dd-agent-debian-workaround"
maintainer       "Datadog"
description      "Works around debian specific issues"  # dirmgr not installed + hostname resolution
long_description IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version          "0.0.2"
depends          "hostsfile"
