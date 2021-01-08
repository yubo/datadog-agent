package snmp

import (
	"fmt"
)

// columnResultValuesType is used to store results fetched for column oids
// Structure: map[<COLUMN OIDS AS STRING>]map[<ROW INDEX>]snmpValue
// - the first map key is the table column oid
// - the second map key is the index part of oid (not prefixed with column oid)
type columnResultValuesType map[string]map[string]snmpValue

// scalarResultValuesType is used to store results fetched for scalar oids
// Structure: map[<INSTANCE OID VALUE>]snmpValue
// - the instance oid value (suffixed with `.0`)
type scalarResultValuesType map[string]snmpValue

func fetchValues(session sessionAPI, config snmpConfig) (*snmpValues, error) {
	// fetch scalar values
	scalarResults, err := fetchScalarOidsWithBatching(session, config.OidConfig.scalarOids, config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("failed to fetch scalar oids with batching: %v", err)
	}

	// fetch column values
	oids := make(map[string]string)
	for _, value := range config.OidConfig.columnOids {
		oids[value] = value
	}
	columnResults, err := fetchColumnOidsWithBatching(session, oids, config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("failed to fetch oids with batching: %v", err)
	}

	return &snmpValues{scalarResults, columnResults}, nil
}
