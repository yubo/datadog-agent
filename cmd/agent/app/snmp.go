// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package app

import (
	"fmt"

	"github.com/DataDog/datadog-agent/cmd/agent/common"
	"github.com/DataDog/datadog-agent/pkg/api/util"
	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/spf13/cobra"
)

var (
	snmpCmd = &cobra.Command{
		Use:   "snmp",
		Short: "",
		Long:  ``,
	}

	snmpListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all devices discovered so far by SNMP autodiscovery.",
		Long:  ``,
		RunE:  doGetSnmpDevices,
	}
)

func init() {
	err := common.SetupConfig(confFilePath)
	if err != nil {
		fmt.Printf("unable to set up global agent configuration: %s", err)
		return
	}
	AgentCmd.AddCommand(snmpCmd)
	snmpCmd.AddCommand(snmpListCmd)
}

var getSnmpDevicesCommand = &cobra.Command{
	Use:   "snmp-devices",
	Short: "Displays all known devices for scanned networks.",
	Long:  ``,
	RunE:  doGetSnmpDevices,
}

func doGetSnmpDevices(cmd *cobra.Command, args []string) error {

	c := util.GetClient(false) // FIX: get certificates right then make this true
	ipcAddress, err := config.GetIPCAddress()
	if err != nil {
		return err
	}
	urlstr := fmt.Sprintf("https://%v:%v/agent/snmp/devices", ipcAddress, config.Datadog.GetInt("cmd_port"))
	err = util.SetAuthToken()
	if err != nil {
		return err
	}
	body, e := util.DoGet(c, urlstr)
	if e != nil {
		fmt.Printf("Error getting SNMP devices: %s\n", e)
		return e
	}
	fmt.Printf("%s\n", body)
	return nil
}
