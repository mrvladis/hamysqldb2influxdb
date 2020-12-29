package main

import (
	"time"

	"gorm.io/datatypes"
)

type appConfiguration struct {
	MySQLUser     string `json:"MySQL_User,omitempty"`
	MySQLPassword string `json:"MySQL_Password,omitempty"`
	MySQLHost     string `json:"MySQL_Host,omitempty"`
	MySQLPort     string `json:"MySQL_Port,omitempty"`
	MySQLDB       string `json:"MySQL_DB,omitempty"`
	MySQLCharset  string `json:"MySQL_Charset,omitempty"`
	MySQLLimit    int    `json:"MySQL_Limit,omitempty"`
	InfluxHost    string `json:"Influx_Host,omitempty"`
	InfluxPort    string `json:"Influx_Port,omitempty"`
	InfluxToken   string `json:"Influx_Token,omitempty"`
	InfluxBucket  string `json:"Influx_Bucket,omitempty"`
	InfluxOrg     string `json:"Influx_Org,omitempty"`
}

type haStateTemp struct {
	StateID       uint
	Domain        string
	EntityID      string
	State         float64
	Attributes    datatypes.JSON
	EventID       uint
	LastChanged   time.Time
	LastUpdated   time.Time
	Created       time.Time
	ContextID     string
	ContextUserID string
	OldStateID    string
}

type tempAttribute struct {
	Voltage           float64 `json:"voltage,omitempty"`
	BatteryLevel      float64 `json:"battery_level,omitempty"`
	UnitOfMeasurement string  `json:"unit_of_measurement,omitempty"`
	FriendlyName      string  `json:"friendly_name,omitempty"`
	DeviceClass       string  `json:"device_class,omitempty"`
	Model             string  `json:"model,omitempty"`
}

// '{"voltage": 2.75, "battery_level": 0.0, "unit_of_measurement": "\u00b0C", "friendly_name": "Bedroom Temperature", "device_class": "temperature"}'
