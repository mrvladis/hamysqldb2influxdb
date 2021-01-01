package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	layoutISO = "2006-01-02 15:04:05"
)

func main() {

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
	mySQLdsn := appConfig.MySQLUser + ":" + appConfig.MySQLPassword + "@tcp(" + appConfig.MySQLHost + ":" + strconv.Itoa(appConfig.MySQLPort) + ")/" + appConfig.MySQLDB + "?charset=" + appConfig.MySQLCharset + "&parseTime=True&loc=Local"
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
	influxdb2dsn := "http://" + appConfig.InfluxHost + ":" + strconv.Itoa(appConfig.InfluxPort)
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
	defer fmt.Println("Flushing any of the data to InfluxDB")
	defer fmt.Println("")
	fmt.Println("InfluxDB connection was sucessfull")
	// Migration execution
	MySQLFilterStartDate, err := time.Parse(layoutISO, appConfig.MySQLFilterStartDate)
	if err != nil {
		fmt.Println("Can't conver ", appConfig.MySQLFilterStartDate, "into Time. Error:", err)
		panic(err.Error())
	}
	MySQLFilterEndDate, err := time.Parse(layoutISO, appConfig.MySQLFilterEndDate)
	if err != nil {
		fmt.Println("Can't conver ", appConfig.MySQLFilterEndDate, "into Time. Error:", err)
		panic(err.Error())
	}
	hoursPerMonth := appConfig.MySQLQuerryHoursInterval
	fmt.Println("Preparing to process MySQL data from the date / time:", MySQLFilterStartDate, "till the date / time:", MySQLFilterEndDate)
	if MySQLFilterEndDate.Sub(MySQLFilterStartDate).Hours()/hoursPerMonth > 2 { // If we have duration more than 2 month
		FilterStartDate := MySQLFilterStartDate
		for FilterEndDate := MySQLFilterStartDate.Add(time.Hour * time.Duration(hoursPerMonth)); MySQLFilterEndDate.Sub(FilterEndDate).Hours() > hoursPerMonth; FilterEndDate = FilterEndDate.Add(time.Hour * time.Duration(hoursPerMonth)) {
			processRequest(db, writeAPI, FilterStartDate, FilterEndDate, &appConfig)
			FilterStartDate = FilterEndDate
		}
		processRequest(db, writeAPI, FilterStartDate, MySQLFilterEndDate, &appConfig)

	} else {
		processRequest(db, writeAPI, MySQLFilterStartDate, MySQLFilterEndDate, &appConfig)
	}

}
