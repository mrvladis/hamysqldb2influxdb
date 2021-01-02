package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"gorm.io/gorm"
)

func processRequest(db *gorm.DB, writeAPI api.WriteAPI, entity entities, FilterStartDate time.Time, FilterEndDate time.Time, appConfig *appConfiguration, wg *sync.WaitGroup) {
	fmt.Println("Processing sequence  for domain", entity.Domain, "with the MySQL filter", entity.MySQLSearchPattern)
	FilterMySQLLimit := appConfig.MySQLLimit
	hastates, results, err := processMysqlrequest(db, entity, FilterStartDate, FilterEndDate, FilterMySQLLimit)
	if err != nil {
		fmt.Println("Somethign went wrong as part of the MySQL query", err)
		panic(err.Error())
	}
	wg.Add(1)
	go func(writeAPI api.WriteAPI, results *gorm.DB, hastates []haState, entity entities) {
		// Decrement the counter when the go routine completes
		defer wg.Done()
		// Call the function check
		processInfluxRequest(writeAPI, results, hastates, entity)
	}(writeAPI, results, hastates, entity)

}

func processMysqlrequest(db *gorm.DB, entity entities, FilterStartDate time.Time, FilterEndDate time.Time, FilterMySQLLimit int) ([]haState, *gorm.DB, error) {
	var hastates []haState
	var results *gorm.DB

	fmt.Println("Querying MySQL Database for the Sensor data between ", FilterStartDate, "and ", FilterEndDate)
	if entity.Domain == "sensor" {
		results = db.Table("states").Where("entity_id LIKE ? AND state REGEXP ?  AND last_changed > ? AND last_changed < ?", entity.MySQLSearchPattern, "^[+-]?([0-9]*[.])?[0-9]+$", FilterStartDate, FilterEndDate).Limit(FilterMySQLLimit).Find(&hastates)
	} else if entity.Domain == "climate" {
		results = db.Table("states").Where("entity_id LIKE ?  AND last_changed > ? AND last_changed < ?", entity.MySQLSearchPattern, FilterStartDate, FilterEndDate).Limit(FilterMySQLLimit).Find(&hastates)
	}
	/*
		2021/01/01 17:54:38 /Users/vladislavnedosekin/go/src/hamysqldb2influxdb/mysql.go:23 SLOW SQL >= 200ms
		[628776.215ms] [rows:1309307] SELECT * FROM `states` WHERE entity_id LIKE 'sensor.%' AND state REGEXP '^[+-]?([0-9]*[.])?[0-9]+$'  AND last_changed > '2020-11-01 20:00:00' AND last_changed < '2020-12-01 23:59:59'
	*/
	/*	results := db.Table("(?) as tmpt", db.Table("states").Where("last_changed > ? AND last_changed < ?", FilterStartDate, FilterEndDate)).Where("entity_id LIKE ? AND state REGEXP ?", "sensor.%", "^[+-]?([0-9]*[.])?[0-9]+$").Limit(appConfig.MySQLLimit).Find(&hastates)
		2021/01/01 17:33:05 /Users/vladislavnedosekin/go/src/hamysqldb2influxdb/mysql.go:25 SLOW SQL >= 200ms
		[695058.282ms] [rows:1309307] SELECT * FROM (SELECT * FROM `states` WHERE last_changed > '2020-11-01 20:00:00' AND last_changed < '2020-12-01 23:59:59') as tmpt WHERE entity_id LIKE 'sensor.%' AND state REGEXP '^[+-]?([0-9]*[.])?[0-9]+$'
	*/
	if results.Error != nil {
		panic(results.Error) // proper error handling instead of panic in your app
	}
	fmt.Println("There are", results.RowsAffected, " Rows to process")
	return hastates, results, results.Error
}

func processInfluxRequest(writeAPI api.WriteAPI, results *gorm.DB, hastates []haState, entity entities) {
	var i int64
	var sensorAttrib sensorAttributes
	var climateAttrib climateAttributes
	var p *write.Point
	var entityID, UnitOfMeasurement string
	var err error

	fmt.Println("Asynchronisly writing data to InfluxDB")
	for i = 0; i < results.RowsAffected; i++ {
		fmt.Printf("\rProcessign Row %d/%d", i, results.RowsAffected)

		switch entity.Domain {
		case "sensor":
			entityID = strings.ReplaceAll(hastates[i].EntityID, "sensor.", "")
			err = json.Unmarshal([]byte(hastates[i].Attributes), &sensorAttrib)
			if err != nil {
				fmt.Println("Couldn't unmarshal Attribute for Entity ", hastates[i].EntityID)
				if ute, ok := err.(*json.UnmarshalTypeError); ok {
					fmt.Printf("UnmarshalTypeError %v - %v - %v\n", ute.Value, ute.Type, ute.Offset)
				} else {
					fmt.Println("Other error:", err)
				}
			}
			value, err := strconv.ParseFloat(hastates[i].State, 64)
			if err != nil {
				fmt.Println("Couldn't convert state [", hastates[i].State, "] into Float64 for Entity [", hastates[i].EntityID, "]")
				panic(err.Error())
			} else {
				// Create point using fluent style
				p = influxdb2.NewPointWithMeasurement(sensorAttrib.UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", sensorAttrib.FriendlyName).
					AddField("value", value).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}
			if sensorAttrib.BatteryLevel > 0 {
				p = influxdb2.NewPointWithMeasurement(sensorAttrib.UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", sensorAttrib.FriendlyName).
					AddField("battery_level", sensorAttrib.BatteryLevel).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}
			if sensorAttrib.Voltage > 0 {
				p = influxdb2.NewPointWithMeasurement(sensorAttrib.UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", sensorAttrib.FriendlyName).
					AddField("voltage", sensorAttrib.Voltage).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}
			if sensorAttrib.DeviceClass != "" {
				p = influxdb2.NewPointWithMeasurement(sensorAttrib.UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", sensorAttrib.FriendlyName).
					AddField("device_class_str", sensorAttrib.DeviceClass).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if sensorAttrib.Model != "" {
				p = influxdb2.NewPointWithMeasurement(sensorAttrib.UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", sensorAttrib.FriendlyName).
					AddField("device_class_str", sensorAttrib.Model).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

		case "climate":
			entityID = strings.ReplaceAll(hastates[i].EntityID, "climate.", "")
			err = json.Unmarshal([]byte(hastates[i].Attributes), &climateAttrib)
			if err != nil {
				fmt.Println("Couldn't unmarshal Attribute for Entity ", hastates[i].EntityID)
				if ute, ok := err.(*json.UnmarshalTypeError); ok {
					fmt.Printf("UnmarshalTypeError %v - %v - %v\n", ute.Value, ute.Type, ute.Offset)
				} else {
					fmt.Println("Other error:", err)
				}
			}
			if climateAttrib.UnitOfMeasurement != "" {
				UnitOfMeasurement = climateAttrib.UnitOfMeasurement
			} else {
				UnitOfMeasurement = "units"
			}

			// Create point using fluent style
			p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
				AddTag("domain", hastates[i].Domain).
				AddTag("entity_id", entityID).
				AddTag("friendly_name", climateAttrib.FriendlyName).
				AddField("value", hastates[i].State).
				SetTime(hastates[i].LastUpdated)
			writeAPI.WritePoint(p)

			if len(climateAttrib.HvacModes) > 0 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("hvac_modes_str", climateAttrib.HvacModes).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.MinTemp > -50 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("min_temp", climateAttrib.MinTemp).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}
			if climateAttrib.MaxTemp > -50 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("max_temp", climateAttrib.MaxTemp).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if len(climateAttrib.PresetModes) > 0 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("preset_modes_str", climateAttrib.PresetModes).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.CurrentTemperature > -50 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("current_temperature", climateAttrib.CurrentTemperature).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.Temperature > -50 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("temperature", climateAttrib.Temperature).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.HvacAction != "" {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("hvac_action_str", climateAttrib.HvacAction).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.PresetMode != "" {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("preset_mode_str", climateAttrib.PresetMode).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.PercentageDemand >= 0 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("percentage_demand", climateAttrib.PercentageDemand).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.ControlOutputState != "" {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("control_output_state_str", climateAttrib.ControlOutputState).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.HeatingRate >= 0 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("heating_rate", climateAttrib.HeatingRate).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			if climateAttrib.WindowState != "" {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("window_state_str", climateAttrib.WindowState).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

			p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
				AddTag("domain", hastates[i].Domain).
				AddTag("entity_id", entityID).
				AddTag("friendly_name", climateAttrib.FriendlyName).
				AddField("away_mode_supressed", climateAttrib.AwayModeSupressed).
				SetTime(hastates[i].LastUpdated)
			writeAPI.WritePoint(p)

			p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
				AddTag("domain", hastates[i].Domain).
				AddTag("entity_id", entityID).
				AddTag("friendly_name", climateAttrib.FriendlyName).
				AddField("window_detection_active", climateAttrib.WindowDetectionActive).
				SetTime(hastates[i].LastUpdated)
			writeAPI.WritePoint(p)

			if climateAttrib.SupportedFeatures >= 0 {
				p = influxdb2.NewPointWithMeasurement(UnitOfMeasurement).
					AddTag("domain", hastates[i].Domain).
					AddTag("entity_id", entityID).
					AddTag("friendly_name", climateAttrib.FriendlyName).
					AddField("supported_features", climateAttrib.SupportedFeatures).
					SetTime(hastates[i].LastUpdated)
				writeAPI.WritePoint(p)
			}

		}

	}
	fmt.Println("")

}
