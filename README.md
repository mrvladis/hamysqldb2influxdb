# hamysqldb2influxdb
Tool for HomeAssistant MySQL Historical Sensor Data Migration to InfluxDB
# Purpose
Migration of the sensor state historical data from MYSQLDB to InfluxDB.
# Supported Domains
## Sensor
Currently only supported Temperature Sensors
# HowTo
## Build
go get -t ./...

go build

chmod 766 hamysqldb2influxdb

## Execute
Copy config/config_sample.json to config/config.json

Update config.json setting with you values.

Run the tool
