package snmp

func fetchScalarOids(session sessionAPI, oids []string) (map[string]interface{}, error) {
	// Get results
	// TODO: make batches
	results, err := session.Get(oids)
	if err != nil {
		return nil, err
	}
	return resultToScalarValues(results), nil
}

func fetchColumnOids(session sessionAPI, oids []string) (map[string]map[string]interface{}, error) {
	// GetBulk results
	// TODO:
	//   - make batches
	//   - GetBulk loop to get all rows
	columnResults, err := session.GetBulk(oids)
	if err != nil {
		return nil, err
	}
	return resultToColumnValues(oids, columnResults), nil
}
