package snmp

type oidConfig struct {
	scalarOids []string
	columnOids []string
}

func (oc *oidConfig) hasOids() bool {
	// TODO: Test me
	return len(oc.columnOids) != 0 || len(oc.scalarOids) != 0
}

func (oc *oidConfig) addScalarOid(oidToAdd string) {
	// TODO: Test me
	oc.scalarOids = appendOidIfMissing(oc.scalarOids, oidToAdd)
}

func (oc *oidConfig) addScalarOids(oidsToAdd []string) {
	// TODO: Test me
	for _, oid := range oidsToAdd {
		oc.addScalarOid(oid)
	}
}

func (oc *oidConfig) addColumnOid(oidToAdd string) {
	// TODO: Test me
	oc.columnOids = appendOidIfMissing(oc.columnOids, oidToAdd)
}

func (oc *oidConfig) addColumnOids(oidsToAdd []string) {
	// TODO: Test me
	for _, oid := range oidsToAdd {
		oc.addColumnOid(oid)
	}
}

func appendOidIfMissing(oids []string, oidToAdd string) []string {
	for _, oid := range oids {
		if oid == oidToAdd {
			return oids
		}
	}
	return append(oids, oidToAdd)
}
