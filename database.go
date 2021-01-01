package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"gorm.io/gorm"
)

func processRequest(db *gorm.DB, writeAPI api.WriteAPI, FilterStartDate time.Time, FilterEndDate time.Time, appConfig *appConfiguration, wg *sync.WaitGroup) {
	fmt.Println("Processing sequence")
	FilterMySQLLimit := appConfig.MySQLLimit
	hastates, results, err := processMysqlrequest(db, FilterStartDate, FilterEndDate, FilterMySQLLimit)
	if err != nil {
		fmt.Println("Somethign went wrong as part of the MySQL query", err)
		panic(err.Error())
	}
	wg.Add(1)
	go func(writeAPI api.WriteAPI, results *gorm.DB, hastates []haStateTemp) {
		// Decrement the counter when the go routine completes
		defer wg.Done()
		// Call the function check
		processInfluxRequest(writeAPI, results, hastates)
	}(writeAPI, results, hastates)

}

func processMysqlrequest(db *gorm.DB, FilterStartDate time.Time, FilterEndDate time.Time, FilterMySQLLimit int) ([]haStateTemp, *gorm.DB, error) {
	var hastates []haStateTemp

	fmt.Println("Querrying MySQL Database for the Sensor data between ", FilterStartDate, "and ", FilterEndDate)
	results := db.Table("states").Where("entity_id LIKE ? AND state REGEXP ?  AND last_changed > ? AND last_changed < ?", "sensor.%", "^[+-]?([0-9]*[.])?[0-9]+$", FilterStartDate, FilterEndDate).Limit(FilterMySQLLimit).Find(&hastates)
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

func processInfluxRequest(writeAPI api.WriteAPI, results *gorm.DB, hastates []haStateTemp) {
	var i int64
	var attrib tempAttribute
	var p *write.Point
	var entityID string

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
