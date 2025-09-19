package config

import (
	"os"
	"strconv"
)

const DATABASE_TYPE = "GFLOW_DATABASE_TYPE"
const DATABASE_URL = "GFLOW_DATABASE_URL"
const DATABASE_SQLLITE_FILE_NAME = "GFLOW_DATABASE_SQLLITE_FILE_NAME"
const ENGINE_SERVER_WEB_PORT = "GFLOW_ENGINE_SERVER_WEB_PORT"
const ENGINE_CHECK_DB_INTERVAL = "GFLOW_ENGINE_CHECK_DB_INTERVAL"
const ENGINE_STUCK_WORKFLOWS_INTERVAL = "GFLOW_ENGINE_STUCK_WORKFLOWS_INTERVAL"
const ENGINE_STUCK_WORKFLOWS_REPAIR_AFTER_MINUTES = "GFLOW_ENGINE_STUCK_WORKFLOWS_REPAIR_AFTER_MINUTES"
const ENGINE_BATCH_SIZE = "GFLOW_ENGINE_BATCH_SIZE"         //number of workflows to pull from the database at a time
const ENGINE_EXECUTOR_GROUP = "GFLOW_ENGINE_EXECUTOR_GROUP" //the group id of the exexutor that it will process jobs from
const ENGINE_EXECUTOR_SIZE = "GFLOW_ENGINE_EXECUTOR_SIZE"   //number of workers to run ie the parallel nature of the jobs
const WEB_SESSION_EXPIRY_HOURS = "GFLOW_WEB_SESSION_EXPIRY_HOURS"

const DATABASE_TYPE_POSTGRES = "POSTGRES"
const DATABASE_TYPE_MYSQL = "MYSQL"
const DATABASE_TYPE_SQLLITE = "SQLLITE"

func GetSystemSettingInteger(settingKey string) int {
	val := GetSystemSettingString(settingKey)
	if val != "" {
		intValue, _ := strconv.Atoi(val)
		return intValue
	}

	//throw an exception
	return 0
}

func GetSystemSettingString(settingKey string) string {
	val := os.Getenv(settingKey)
	if val != "" {
		return val
	}
	if settingKey == ENGINE_CHECK_DB_INTERVAL {
		return "3s" // default to 5 seconds
	}
	if settingKey == ENGINE_STUCK_WORKFLOWS_INTERVAL {
		return "60s" // default to 60 seconds
	}
	if settingKey == ENGINE_BATCH_SIZE {
		return "5" // default to 5 seconds
	}
	if settingKey == ENGINE_STUCK_WORKFLOWS_REPAIR_AFTER_MINUTES {
		return "5" // default to 5 minutes
	}
	if settingKey == ENGINE_EXECUTOR_SIZE {
		return "5" // default to 5
	}
	if settingKey == ENGINE_EXECUTOR_GROUP {
		return "default"
	}
	if settingKey == ENGINE_SERVER_WEB_PORT {
		return "8080"
	}
	if settingKey == WEB_SESSION_EXPIRY_HOURS {
		return "1"
	}
	if settingKey == DATABASE_SQLLITE_FILE_NAME {
		return "./gflow.db"
	}
	return ""
}
