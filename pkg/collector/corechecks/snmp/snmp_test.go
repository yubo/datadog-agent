// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/soniah/gosnmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type mockSession struct {
	mock.Mock
}

func (s *mockSession) Configure(config snmpConfig) {
}

func (s *mockSession) Get(oids []string) (result *gosnmp.SnmpPacket, err error) {
	return &gosnmp.SnmpPacket{}, nil
}

func (s *mockSession) Connect() error {
	return nil
}

func (s *mockSession) Close() error {
	return nil
}

func TestBasicSample(t *testing.T) {
	session := &mockSession{}
	check := Check{session: session}
	// language=yaml
	rawInstanceConfig := []byte(`
ip_address: 1.2.3.4
`)

	err := check.Configure(rawInstanceConfig, []byte(``), "test")
	assert.Nil(t, err)

	sender := mocksender.NewMockSender(check.ID()) // required to initiate aggregator
	sender.On("Gauge", "snmp.test.metric", mock.Anything, mock.Anything, mock.Anything).Return()
	sender.On("Commit").Return()

	err = check.Run()
	assert.Nil(t, err)
}
