package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

func processMysqlrequest(db *gorm.DB, writeAPI api.WriteAPI, FilterStartDate time.Time, FilterEndDate time.Time, appConfig *appConfiguration) {
	var hastates []haStateTemp
	var attrib tempAttribute
	var p *write.Point
	var entityID string
	var i int64
	fmt.Println("Querrying MySQL Database for the Sensor data between ", FilterStartDate, "and ", FilterEndDate)
	results := db.Table("states").Where("entity_id LIKE ? AND state REGEXP ?  AND last_changed > ? AND last_changed < ?", "sensor.%", "^[+-]?([0-9]*[.])?[0-9]+$", FilterStartDate, FilterEndDate).Limit(appConfig.MySQLLimit).Find(&hastates)
	if results.Error != nil {
		panic(results.Error) // proper error handling instead of panic in your app
	}
	fmt.Println("There are", results.RowsAffected, " Rows to process")
	fmt.Println("Asynchronisly writing data to InfluxDB")
	for i = 0; i < results.RowsAffected; i++ {
		fmt.Printf("\rProcessign Row %d/%d", i, results.RowsAffected)
		err := json.Unmarshal([]byte(hastates[i].Attributes), &attrib)
		if err != nil {
			fmt.Println("Couldn't unmarshal Attribute for Entity ", hastates[i].EntityID)
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				fmt.Printf("UnmarshalTypeError %v - %v - %v\n", ute.Value, ute.Type, ute.Offset)
			} else {
				fmt.Println("Other error:", err)
			}
		}
		entityID = strings.ReplaceAll(hastates[i].EntityID, "sensor.", "")
		// Create point using fluent style
		if attrib.BatteryLevel > 0 {
			p = influxdb2.NewPointWithMeasurement(attrib.UnitOfMeasurement).
				AddTag("domain", hastates[i].Domain).
				AddTag("entity_id", entityID).
				AddTag("friendly_name", attrib.FriendlyName).
				AddField("battery_level", attrib.BatteryLevel).
				SetTime(hastates[i].LastUpdated)
			writeAPI.WritePoint(p)
		}
		if attrib.Voltage > 0 {
			p = influxdb2.NewPointWithMeasurement(attrib.UnitOfMeasurement).
				AddTag("domain", hastates[i].Domain).
				AddTag("entity_id", entityID).
				AddTag("friendly_name", attrib.FriendlyName).
				AddField("voltage", attrib.Voltage).
				SetTime(hastates[i].LastUpdated)
			writeAPI.WritePoint(p)
		}
		if attrib.DeviceClass != "" {
			p = influxdb2.NewPointWithMeasurement(attrib.UnitOfMeasurement).
				AddTag("domain", hastates[i].Domain).
				AddTag("entity_id", entityID).
				AddTag("friendly_name", attrib.FriendlyName).
				AddField("device_class_str", attrib.DeviceClass).
				SetTime(hastates[i].LastUpdated)
			writeAPI.WritePoint(p)
		}

		if attrib.Model != "" {
			p = influxdb2.NewPointWithMeasurement(attrib.UnitOfMeasurement).
				AddTag("domain", hastates[i].Domain).
				AddTag("entity_id", entityID).
				AddTag("friendly_name", attrib.FriendlyName).
				AddField("device_class_str", attrib.Model).
				SetTime(hastates[i].LastUpdated)
			writeAPI.WritePoint(p)
		}

		p = influxdb2.NewPointWithMeasurement(attrib.UnitOfMeasurement).
			AddTag("domain", hastates[i].Domain).
			AddTag("entity_id", entityID).
			AddTag("friendly_name", attrib.FriendlyName).
			AddField("value", hastates[i].State).
			SetTime(hastates[i].LastUpdated)
		writeAPI.WritePoint(p)
	}
	fmt.Println("")
}
