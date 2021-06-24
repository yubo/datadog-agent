// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package serverless

const (
	// Invoke event
	Invoke RuntimeEvent = "INVOKE"
	// Shutdown event
	Shutdown RuntimeEvent = "SHUTDOWN"
	// Timeout is one of the possible ShutdownReasons
	Timeout ShutdownReason = "timeout"
)

// ShutdownReason is an AWS Shutdown reason
type ShutdownReason string

// RuntimeEvent is an AWS Runtime event
type RuntimeEvent string

// ErrorEnum are errors reported to the AWS Extension environment.
type ErrorEnum string

// String returns the string value for this ErrorEnum.
func (e ErrorEnum) String() string {
	return string(e)
}

// String returns the string value for this ShutdownReason.
func (s ShutdownReason) String() string {
	return string(s)
}
