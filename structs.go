package main

import (
	"time"

	"gorm.io/datatypes"
)

type appConfiguration struct {
	MySQLUser                  string  `json:"MySQL_User,omitempty"`
	MySQLPassword              string  `json:"MySQL_Password,omitempty"`
	MySQLHost                  string  `json:"MySQL_Host,omitempty"`
	MySQLPort                  int     `json:"MySQL_Port,omitempty"`
	MySQLDB                    string  `json:"MySQL_DB,omitempty"`
	MySQLCharset               string  `json:"MySQL_Charset,omitempty"`
	MySQLLimit                 int     `json:"MySQL_Limit,omitempty"`
	MySQLQueryHoursInterval    float64 `json:"MySQL_QueryHoursInterval,omitempty"`
	MySQLFilterStartDate       string  `json:"MySQL_FilterStartDate,omitempty"`
	MySQLFilterEndDate         string  `json:"MySQL_FilterEndDate,omitempty"`
	MySQLHASensorQuery         string  `json:"MySQL_HA_Sensor_Query,omitempty"`
	MySQLHASensorQueryEnabled  bool    `json:"MySQL_HA_Sensor_Query_Enabled,omitempty"`
	MySQLHAClimateQuery        string  `json:"MySQL_HA_Climate_Query,omitempty"`
	MySQLHAClimateQueryEnabled bool    `json:"MySQL_HA_Climate_Query_Enabled,omitempty"`
	InfluxHost                 string  `json:"Influx_Host,omitempty"`
	InfluxPort                 int     `json:"Influx_Port,omitempty"`
	InfluxToken                string  `json:"Influx_Token,omitempty"`
	InfluxBucket               string  `json:"Influx_Bucket,omitempty"`
	InfluxOrg                  string  `json:"Influx_Org,omitempty"`
}

type haState struct {
	StateID       uint
	Domain        string
	EntityID      string
	State         string // float64 has been replaced with string for broader compatibility. Transformation would have to be done just before wtiging to InfluxDB
	Attributes    datatypes.JSON
	EventID       uint
	LastChanged   time.Time
	LastUpdated   time.Time
	Created       time.Time
	ContextID     string
	ContextUserID string
	OldStateID    string
}

type entities struct {
	Domain             string
	Enabled            bool
	MySQLSearchPattern string
}

type sensorAttributes struct {
	Voltage           float64 `json:"voltage,omitempty"`
	BatteryLevel      float64 `json:"battery_level,omitempty"`
	UnitOfMeasurement string  `json:"unit_of_measurement,omitempty"`
	FriendlyName      string  `json:"friendly_name,omitempty"`
	DeviceClass       string  `json:"device_class,omitempty"`
	Model             string  `json:"model,omitempty"`
}

type climateAttributes struct {
	HvacModes             []string `json:"hvac_modes,omitempty"`
	MinTemp               float64  `json:"min_temp,omitempty"`
	MaxTemp               float64  `json:"max_temp,omitempty"`
	PresetModes           []string `json:"preset_modes,omitempty"`
	CurrentTemperature    float64  `json:"current_temperature,omitempty"`
	Temperature           float64  `json:"temperature,omitempty"`
	HvacAction            string   `json:"hvac_action,omitempty"`
	PresetMode            string   `json:"preset_mode,omitempty"`
	PercentageDemand      float64  `json:"percentage_demand,omitempty"`
	ControlOutputState    string   `json:"control_output_state,omitempty"`
	HeatingRate           float64  `json:"heating_rate,omitempty"`
	WindowState           string   `json:"window_state,omitempty"`
	WindowDetectionActive bool     `json:"window_detection_active,omitempty"`
	AwayModeSupressed     bool     `json:"away_mode_supressed,omitempty"`
	FriendlyName          string   `json:"friendly_name,omitempty"`
	SupportedFeatures     float64  `json:"supported_features,omitempty"`
	UnitOfMeasurement     string   `json:"unit_of_measurement,omitempty"`
}

// '{"voltage": 2.75, "battery_level": 0.0, "unit_of_measurement": "\u00b0C", "friendly_name": "Bedroom Temperature", "device_class": "temperature"}'
