package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api/write"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var hastates []haStateTemp
	var i int64
	var attrib tempAttribute
	var p *write.Point
	var entityID string
	var appConfig appConfiguration

	fmt.Println("Welcome to Home Assistant MYSQL 2 InfluxDB migration Tool")
	configFilePath := "config/config.json"
	configFile, err := os.Open(configFilePath)
	if err != nil {
		panic(err.Error())
	}
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&appConfig)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("Trying to connect to MySQL host ", appConfig.MySQLHost, " and port ", appConfig.MySQLPort)
	mySQLdsn := appConfig.MySQLUser + ":" + appConfig.MySQLPassword + "@tcp(" + appConfig.MySQLHost + ":" + appConfig.MySQLPort + ")/" + appConfig.MySQLDB + "?charset=" + appConfig.MySQLCharset + "&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(mySQLdsn), &gorm.Config{})

	// if there is an error opening the connection, handle it
	if err != nil {
		fmt.Println("Connection to MySQL failed with the following error:", err)
		panic(err.Error())
	}
	fmt.Println("MySQL connection was sucessfull")
	// defer the close till after the main function has finished
	// executing
	sqlDB, err := db.DB()
	defer sqlDB.Close()

	//INFLUX DB connection
	// Create a new client using an InfluxDB server base URL and an authentication token
	fmt.Println("Trying to connect to InfluxDB host ", appConfig.InfluxHost, " and port ", appConfig.InfluxPort)
	influxdb2dsn := "http://" + appConfig.InfluxHost + ":" + appConfig.InfluxPort
	client := influxdb2.NewClient(influxdb2dsn, appConfig.InfluxToken)
	// Ensures background processes finishes

	defer client.Close()

	// Use  write client for writes to desired bucket
	writeAPI := client.WriteAPI(appConfig.InfluxOrg, appConfig.InfluxBucket)
	errorsCh := writeAPI.Errors()
	// Create go proc for reading and logging errors
	go func() {
		for err := range errorsCh {
			fmt.Printf("write error: %s\n", err.Error())
		}
	}()

	defer fmt.Println("InfluxDB Updated Successfully")
	defer writeAPI.Flush()
	defer fmt.Println("Fluushing any of the data to InfluxDB")
	fmt.Println("InfluxDB connection was sucessfull")
	// Migration execution
	fmt.Println("Querrying MySQL Database")

	results := db.Table("states").Where("entity_id LIKE ? AND state REGEXP ?", "%temp%", "[0-9]+([.0-9]+)").Limit(appConfig.MySQLLimit).Find(&hastates)
	if results.Error != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	fmt.Println("There are", results.RowsAffected, " Rows to process")
	for i = 0; i < results.RowsAffected; i++ {
		fmt.Printf("\rProcessign Row %d/%d", i, results.RowsAffected)
		err = json.Unmarshal([]byte(hastates[i].Attributes), &attrib)
		if err != nil {
			fmt.Println("Couldn't unmarshal Attribute")
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

		p = influxdb2.NewPointWithMeasurement(attrib.UnitOfMeasurement).
			AddTag("domain", hastates[i].Domain).
			AddTag("entity_id", entityID).
			AddTag("friendly_name", attrib.FriendlyName).
			AddField("value", hastates[i].State).
			SetTime(hastates[i].LastUpdated)
		writeAPI.WritePoint(p)

	}

}
