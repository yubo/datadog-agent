// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build jmx

package jmxfetch

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/status"
	"github.com/DataDog/datadog-agent/pkg/status/health"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/n9e/n9e-agentd/cmd/agentd/common"
	auth "github.com/n9e/n9e-agentd/pkg/authentication"
	"github.com/n9e/n9e-agentd/pkg/config"
	"gopkg.in/yaml.v2"
)

const (
	jmxJarName                        = "jmxfetch.jar"
	jmxMainClass                      = "org.datadog.jmxfetch.App"
	defaultJmxCommand                 = "collect"
	defaultJvmMaxMemoryAllocation     = " -Xmx200m"
	defaultJvmInitialMemoryAllocation = " -Xms50m"
	jvmCgroupMemoryAwareness          = " -XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap"
	jvmContainerSupport               = " -XX:+UseContainerSupport"
	defaultJavaBinPath                = "java"
	defaultLogLevel                   = "info"
)

var (
	jmxLogLevelMap = map[string]string{
		"trace":    "TRACE",
		"debug":    "DEBUG",
		"info":     "INFO",
		"warn":     "WARN",
		"warning":  "WARN",
		"error":    "ERROR",
		"err":      "ERROR",
		"critical": "FATAL",
		"off":      "OFF",
	}
	jvmCgroupMemoryIncompatOptions = []string{
		"Xmx",
		"XX:MaxHeapSize",
		"Xms",
		"XX:InitialHeapSize",
	}
)

// JMXFetch represent a jmxfetch instance.
type JMXFetch struct {
	JavaBinPath        string
	JavaOptions        string
	JavaToolsJarPath   string
	JavaCustomJarPaths []string
	LogLevel           string
	Command            string
	Reporter           JMXReporter
	Checks             []string
	IPCPort            int
	IPCHost            string
	Output             func(...interface{})
	cmd                *exec.Cmd
	managed            bool
	shutdown           chan struct{}
	stopped            chan struct{}
}

// JMXReporter supports different way of reporting the data it has fetched.
type JMXReporter string

var (
	// ReporterStatsd reports the output to statsd.
	ReporterStatsd JMXReporter = "statsd" // default one
	// ReporterConsole reports the output into the console as plain text
	ReporterConsole JMXReporter = "console"
	// ReporterJSON reports the output into the console as json
	ReporterJSON JMXReporter = "json"
)

// checkInstanceCfg lists the config options on the instance against which we make some sanity checks
// on how they're configured. All the other options should be checked on JMXFetch's side.
type checkInstanceCfg struct {
	JavaBinPath      string `yaml:"java_bin_path,omitempty"`
	JavaOptions      string `yaml:"java_options,omitempty"`
	ToolsJarPath     string `yaml:"tools_jar_path,omitempty"`
	ProcessNameRegex string `yaml:"process_name_regex,omitempty"`
}

type checkInitCfg struct {
	CustomJarPaths []string `yaml:"custom_jar_paths,omitempty"`
	ToolsJarPath   string   `yaml:"tools_jar_path,omitempty"`
	JavaBinPath    string   `yaml:"java_bin_path,omitempty"`
	JavaOptions    string   `yaml:"java_options,omitempty"`
}

func (j *JMXFetch) Monitor() {
	limiter := newRestartLimiter(config.C.Jmx.MaxRestarts, config.C.Jmx.RestartInterval.Seconds())
	ticker := time.NewTicker(500 * time.Millisecond)

	defer ticker.Stop()
	defer close(j.stopped)

	go j.heartbeat(ticker)

	for {
		err := j.Wait()
		if err == nil {
			log.Infof("JMXFetch stopped and exited sanely.")
			break
		}

		if !limiter.canRestart(time.Now()) {
			msg := fmt.Sprintf("Too many JMXFetch restarts (%v) in time interval (%vs) - giving up", limiter.maxRestarts, limiter.interval)
			log.Errorf(msg)
			s := status.JMXStartupError{LastError: msg, Timestamp: time.Now().Unix()}
			status.SetJMXStartupError(s)
			return
		}

		select {
		case <-j.shutdown:
			return
		default:
			// restart
			log.Warnf("JMXFetch process had to be restarted.")
			j.Start(false) //nolint:errcheck
		}
	}

	<-j.shutdown
}

func (j *JMXFetch) setDefaults() {
	if j.JavaBinPath == "" {
		j.JavaBinPath = defaultJavaBinPath
	}
	if j.JavaCustomJarPaths == nil {
		j.JavaCustomJarPaths = []string{}
	}
	if j.LogLevel == "" {
		j.LogLevel = defaultLogLevel
	}
	if j.Command == "" {
		j.Command = defaultJmxCommand
	}
	if j.Checks == nil {
		j.Checks = []string{}
	}
	if j.Output == nil {
		j.Output = log.JMXInfo
	}
}

// Start starts the JMXFetch process
func (j *JMXFetch) Start(manage bool) error {
	j.setDefaults()

	cf := config.C.Jmx

	classpath := filepath.Join(config.C.WorkDir, "dist", "jmx", jmxJarName)
	if j.JavaToolsJarPath != "" {
		classpath = fmt.Sprintf("%s%s%s", j.JavaToolsJarPath, string(os.PathListSeparator), classpath)
	}

	globalCustomJars := cf.CustomJars
	if len(globalCustomJars) > 0 {
		classpath = fmt.Sprintf("%s%s%s", strings.Join(globalCustomJars, string(os.PathListSeparator)), string(os.PathListSeparator), classpath)
	}

	if len(j.JavaCustomJarPaths) > 0 {
		classpath = fmt.Sprintf("%s%s%s", strings.Join(j.JavaCustomJarPaths, string(os.PathListSeparator)), string(os.PathListSeparator), classpath)
	}

	var reporter string
	switch j.Reporter {
	case ReporterConsole:
		reporter = "console"
	case ReporterJSON:
		reporter = "json"
	default:
		if common.DSD != nil && common.DSD.UdsListenerRunning {
			reporter = fmt.Sprintf("statsd:unix://%s", config.C.Statsd.Socket)
		} else {
			bindHost := config.C.GetBindHost()
			if bindHost == "" || bindHost == "0.0.0.0" {
				bindHost = "localhost"
			}
			reporter = fmt.Sprintf("statsd:%s:%s", bindHost, config.C.Statsd.Port)
		}
	}

	//TODO : support auto discovery

	subprocessArgs := []string{}

	// Specify a maximum memory allocation pool for the JVM
	javaOptions := j.JavaOptions

	useContainerSupport := cf.UseContainerSupport
	useCgroupMemoryLimit := cf.UseCgroupMemoryLimit

	if useContainerSupport && useCgroupMemoryLimit {
		return fmt.Errorf("incompatible options %q and %q", jvmContainerSupport, jvmCgroupMemoryAwareness)
	} else if useContainerSupport {
		javaOptions += jvmContainerSupport
	} else if useCgroupMemoryLimit {
		passOption := true
		// This option is incompatible with the Xmx and Xms options, log a warning if there are found in the javaOptions
		for _, option := range jvmCgroupMemoryIncompatOptions {
			if strings.Contains(javaOptions, option) {
				log.Warnf("Java option %q is incompatible with cgroup_memory_limit, disabling cgroup mode", option)
				passOption = false
			}
		}
		if passOption {
			javaOptions += jvmCgroupMemoryAwareness
		}
	} else {
		// Specify a maximum memory allocation pool for the JVM
		if !strings.Contains(javaOptions, "Xmx") && !strings.Contains(javaOptions, "XX:MaxHeapSize") {
			javaOptions += defaultJvmMaxMemoryAllocation
		}
		// Specify the initial memory allocation pool for the JVM
		if !strings.Contains(javaOptions, "Xms") && !strings.Contains(javaOptions, "XX:InitialHeapSize") {
			javaOptions += defaultJvmInitialMemoryAllocation
		}
	}

	subprocessArgs = append(subprocessArgs, strings.Fields(javaOptions)...)

	jmxLogLevel, ok := jmxLogLevelMap[strings.ToLower(j.LogLevel)]
	if !ok {
		jmxLogLevel = "INFO"
	}

	ipcHost := config.C.BindHost
	ipcPort := config.C.BindPort
	if j.IPCHost != "" {
		ipcHost = j.IPCHost
	}
	if j.IPCPort != 0 {
		ipcPort = j.IPCPort
	}

	// checks are now enabled via IPC on JMXFetch
	subprocessArgs = append(subprocessArgs,
		"-classpath", classpath,
		jmxMainClass,
		"--ipc_host", ipcHost,
		"--ipc_port", fmt.Sprintf("%v", ipcPort),
		"--check_period", fmt.Sprintf("%v", cf.CheckPeriod), // Period of the main loop of jmxfetch in ms
		"--thread_pool_size", fmt.Sprintf("%v", cf.ThreadPoolSize), // Size for the JMXFetch thread pool
		"--collection_timeout", fmt.Sprintf("%v", cf.CollectionTimeout.Seconds()), // Timeout for metric collection in seconds
		"--reconnection_timeout", fmt.Sprintf("%v", cf.ReconnectionTimeout.Seconds()), // Timeout for instance reconnection in seconds
		"--reconnection_thread_pool_size", fmt.Sprintf("%v", cf.ReconnectionThreadPoolSize), // Size for the JMXFetch reconnection thread pool
		"--log_level", jmxLogLevel,
		"--reporter", reporter, // Reporter to use
	)

	if config.C.LogFormatRfc3339 {
		subprocessArgs = append(subprocessArgs, "--log_format_rfc3339")
	}

	subprocessArgs = append(subprocessArgs, j.Command)

	j.cmd = exec.Command(j.JavaBinPath, subprocessArgs...)

	// set environment + token
	j.cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("SESSION_TOKEN=%s", auth.GetAuthToken()),
	)

	// forward the standard output to the Agent logger
	stdout, err := j.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	go func() {
	scan:
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			j.Output(in.Text())
		}
		if in.Err() == bufio.ErrTooLong {
			goto scan
		}
	}()

	// forward the standard error to the Agent logger
	stderr, err := j.cmd.StderrPipe()
	if err != nil {
		return err
	}
	go func() {
	scan:
		in := bufio.NewScanner(stderr)
		for in.Scan() {
			log.JMXError(in.Text())
		}
		if in.Err() == bufio.ErrTooLong {
			goto scan
		}
	}()

	log.Debugf("Args: %v", subprocessArgs)

	err = j.cmd.Start()

	// start synchronization channels
	if err == nil && manage {
		j.managed = true
		j.shutdown = make(chan struct{})
		j.stopped = make(chan struct{})

		go j.Monitor()
	}

	return err
}

// Wait waits for the end of the JMXFetch process and returns the error code
func (j *JMXFetch) Wait() error {
	return j.cmd.Wait()
}

func (j *JMXFetch) heartbeat(beat *time.Ticker) {
	health := health.RegisterLiveness("jmxfetch")
	defer health.Deregister() //nolint:errcheck

	for range beat.C {
		select {
		case <-health.C:
		case <-j.shutdown:
			return
		}
	}
}

// Up returns if JMXFetch is up - used by healthcheck
func (j *JMXFetch) Up() (bool, error) {
	// TODO: write windows implementation
	process, err := os.FindProcess(j.cmd.Process.Pid)
	if err != nil {
		return false, fmt.Errorf("failed to find process: %s", err)
	}

	// from man kill(2):
	// if sig is 0, then no signal is sent, but error checking is still performed
	err = process.Signal(syscall.Signal(0))
	return err == nil, err
}

// ConfigureFromInitConfig configures various options from the init_config
// section of the configuration
func (j *JMXFetch) ConfigureFromInitConfig(initConfig integration.Data) error {
	var initConf checkInitCfg

	// unmarshall init config
	if err := yaml.Unmarshal(initConfig, &initConf); err != nil {
		return err
	}

	if j.JavaBinPath == "" {
		if initConf.JavaBinPath != "" {
			j.JavaBinPath = initConf.JavaBinPath
		}
	}

	if j.JavaOptions == "" {
		if initConf.JavaOptions != "" {
			j.JavaOptions = initConf.JavaOptions
		}
	}

	if j.JavaToolsJarPath == "" {
		if initConf.ToolsJarPath != "" {
			j.JavaToolsJarPath = initConf.ToolsJarPath
		}
	}
	if j.JavaCustomJarPaths == nil {
		if initConf.CustomJarPaths != nil {
			j.JavaCustomJarPaths = initConf.CustomJarPaths
		}
	}

	return nil
}

// ConfigureFromInstance configures various options from the instance
// section of the configuration
func (j *JMXFetch) ConfigureFromInstance(instance integration.Data) error {

	var instanceConf checkInstanceCfg

	// unmarshall instance info
	if err := yaml.Unmarshal(instance, &instanceConf); err != nil {
		return err
	}

	if j.JavaBinPath == "" {
		if instanceConf.JavaBinPath != "" {
			j.JavaBinPath = instanceConf.JavaBinPath
		}
	}
	if j.JavaOptions == "" {
		if instanceConf.JavaOptions != "" {
			j.JavaOptions = instanceConf.JavaOptions
		}
	}
	if j.JavaToolsJarPath == "" {
		if instanceConf.ToolsJarPath != "" {
			j.JavaToolsJarPath = instanceConf.ToolsJarPath
		}
	}

	if instanceConf.ProcessNameRegex != "" {
		if j.JavaToolsJarPath == "" {
			return fmt.Errorf("You must specify the path to tools.jar. See http://docs.datadoghq.com/integrations/java/ for more information")
		}
	}

	return nil
}
