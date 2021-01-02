# hamysqldb2influxdb
Tool for HomeAssistant MySQL Historical Sensor Data Migration to InfluxDB
# Purpose
Migration of the sensor state historical data from MYSQLDB to InfluxDB.

# Pre-requisits
Home Assistant: https://www.home-assistant.io

MySQL used as a Home Assistant recorder. I have MySQL 8.

InfluxDB V2 https://docs.influxdata.com/influxdb/v2.0/  I have used official HomeAssistant Guide for the configuration: https://www.home-assistant.io/integrations/influxdb/ 

Golang to compile the application for your system.

Both MySQL and InfluxDB need to be exposed to the host you run the tool on over the network.

# Supported Domains
## Sensor
Currently only supported Sensors with state in numeric format.

The following Attributes values would be copied if not empty or 0:

*   Voltage 
*	BatteryLevel 
*	UnitOfMeasurement 
*	FriendlyName
*	DeviceClass 
*	Model

## Climate

The following Attributes values would be copied if not empty or greater than -50:
   
*    HvacModes            
*	MinTemp              
*	MaxTemp              
*	PresetModes          
*	CurrentTemperature   
*	Temperature          
*	HvacAction           
*	PresetMode           
*	PercentageDemand     
*	ControlOutputState   
*	HeatingRate          
*	WindowState          
*	WindowDetectionActive
*	AwayModeSupressed    
*	FriendlyName         
*	SupportedFeatures    
*	UnitOfMeasurement    
# HowTo
## Build
go get -t ./...

go build

chmod 766 hamysqldb2influxdb

## Execute
Copy config/config_sample.json to config/config.json

Update config.json setting with you values.

Run the tool
