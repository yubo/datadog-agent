
```
python3 -m invoke test --skip-linters --targets ./pkg/collector/corechecks/snmp

python3 -m invoke agent.build

bin/agent/agent check snmp

```
