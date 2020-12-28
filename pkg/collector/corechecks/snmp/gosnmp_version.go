package snmp

import (
	"fmt"
	"github.com/soniah/gosnmp"
)

func parseVersion(rawVersion string) (gosnmp.SnmpVersion, error) {
	switch rawVersion {
	case "1":
		return gosnmp.Version1, nil
	case "", "2", "2c":
		return gosnmp.Version2c, nil
	case "3":
		return gosnmp.Version3, nil
	}
	return 0, fmt.Errorf("invalid snmp version `%s`. Valid versions are: 1, 2, 2c, 3", rawVersion)
}
