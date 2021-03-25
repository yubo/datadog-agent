package snmp

type oidConfig struct {
	scalarOids []string
	columnOids []string
}

func (oc *oidConfig) hasOids() bool {
	return len(oc.columnOids) != 0 || len(oc.scalarOids) != 0
}

func (oc *oidConfig) addScalarOid(oidToAdd string) {
	oc.scalarOids = appendOidIfMissing(oc.scalarOids, oidToAdd)
}

func (oc *oidConfig) addScalarOids(oidsToAdd []string) {
	for _, oid := range oidsToAdd {
		oc.addScalarOid(oid)
	}
}

func (oc *oidConfig) addColumnOid(oidToAdd string) {
	oc.columnOids = appendOidIfMissing(oc.columnOids, oidToAdd)
}

func (oc *oidConfig) addColumnOids(oidsToAdd []string) {
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
