package snmp

import (
	"github.com/soniah/gosnmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestFetchColumnOids(t *testing.T) {
	session := &mockSession{}

	bulkPacket := gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  "1.1.1.1",
				Type:  gosnmp.TimeTicks,
				Value: 11,
			},
			{
				Name:  "1.1.2.1",
				Type:  gosnmp.TimeTicks,
				Value: 21,
			},
			{
				Name:  "1.1.1.2",
				Type:  gosnmp.TimeTicks,
				Value: 12,
			},
			{
				Name:  "1.1.2.2",
				Type:  gosnmp.TimeTicks,
				Value: 22,
			},
			{
				Name:  "1.1.1.3",
				Type:  gosnmp.TimeTicks,
				Value: 13,
			},
			{
				Name:  "1.1.3.1",
				Type:  gosnmp.TimeTicks,
				Value: 31,
			},
		},
	}
	session.On("GetBulk", mock.Anything).Return(&bulkPacket, nil)

	oids := []string{"1.1.1", "1.1.2"}

	columnValues, err := fetchColumnOids(session, oids)
	assert.Nil(t, err)

	expectedColumnValues := map[string]map[string]interface{}{
		"1.1.1": {
			"1": float64(11),
			"2": float64(12),
			"3": float64(13),
		},
		"1.1.2": {
			"1": float64(21),
			"2": float64(22),
		},
	}
	assert.Equal(t, expectedColumnValues, columnValues)
}
