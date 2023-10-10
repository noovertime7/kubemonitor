package mysql

import (
	"database/sql"
	"fmt"
	"github.com/noovertime7/kubemonitor/pkg/input"
	"github.com/noovertime7/kubemonitor/pkg/types"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const inputName = "mysql"

func init() {
	input.Factory.RegisterHandler(&Instance{})
}

type Instance struct {
	Address        string `json:"address"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Parameters     string `json:"parameters"`
	TimeoutSeconds int64  `json:"timeout_seconds"`

	ExtraStatusMetrics              bool `json:"extra_status_metrics"`
	ExtraInnodbMetrics              bool `json:"extra_innodb_metrics"`
	GatherProcessListProcessByState bool `json:"gather_processlist_processes_by_state"`
	GatherProcessListProcessByUser  bool `json:"gather_processlist_processes_by_user"`
	GatherSchemaSize                bool `json:"gather_schema_size"`
	GatherTableSize                 bool `json:"gather_table_size"`
	GatherSystemTableSize           bool `json:"gather_system_table_size"`
	GatherSlaveStatus               bool `json:"gather_slave_status"`

	DisableGlobalStatus      bool `json:"disable_global_status"`
	DisableGlobalVariables   bool `json:"disable_global_variables"`
	DisableInnodbStatus      bool `json:"disable_innodb_status"`
	DisableExtraInnodbStatus bool `json:"disable_extra_innodb_status"`
	DisablebinLogs           bool `json:"disable_binlogs"`

	validMetrics map[string]struct{}
	dsn          string
}

func (ins *Instance) Name() string {
	return inputName
}

func (ins *Instance) Init(config input.ConfigMap) error {
	ins.Address = config["address"]
	ins.Username = config["username"]
	ins.Password = config["password"]
	ins.Parameters = config["parameters"]
	timeout, err := strconv.ParseInt(config["timeout_seconds"], 10, 64)
	if err != nil {
		return err
	}
	ins.TimeoutSeconds = timeout

	ins.ExtraStatusMetrics = config["extra_status_metrics"] == "true"
	ins.ExtraInnodbMetrics = config["extra_innodb_metrics"] == "true"
	ins.GatherProcessListProcessByState = config["gather_processlist_processes_by_state"] == "true"
	ins.GatherProcessListProcessByUser = config["gather_processlist_processes_by_user"] == "true"
	ins.GatherSchemaSize = config["gather_schema_size"] == "true"
	ins.GatherTableSize = config["gather_table_size"] == "true"
	ins.GatherSystemTableSize = config["gather_system_table_size"] == "true"
	ins.GatherSlaveStatus = config["gather_slave_status"] == "true"

	ins.DisableGlobalStatus = config["disable_global_status"] == "true"
	ins.DisableGlobalVariables = config["disable_global_variables"] == "true"
	ins.DisableInnodbStatus = config["disable_innodb_status"] == "true"
	ins.DisableExtraInnodbStatus = config["disable_extra_innodb_status"] == "true"
	ins.DisablebinLogs = config["disable_binlogs"] == "true"

	if ins.Address == "" {
		return types.ErrInstancesEmpty
	}

	net := "tcp"
	if strings.HasSuffix(ins.Address, ".sock") {
		net = "unix"
	}
	ins.dsn = fmt.Sprintf("%s:%s@%s(%s)/?%s", ins.Username, ins.Password, net, ins.Address, ins.Parameters)
	conf, err := mysql.ParseDSN(ins.dsn)
	if err != nil {
		return err
	}
	if conf.Timeout == 0 {
		if ins.TimeoutSeconds == 0 {
			ins.TimeoutSeconds = 3
		}
		conf.Timeout = time.Second * time.Duration(ins.TimeoutSeconds)
	}

	ins.dsn = conf.FormatDSN()

	ins.InitValidMetrics()

	return nil
}

func (ins *Instance) InitValidMetrics() {
	ins.validMetrics = make(map[string]struct{})

	for key := range STATUS_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range VARIABLES_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range INNODB_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range BINLOG_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range GALERA_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range PERFORMANCE_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range SCHEMA_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range TABLE_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range REPLICA_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range GROUP_REPLICATION_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	for key := range SYNTHETIC_VARS {
		ins.validMetrics[key] = struct{}{}
	}

	if ins.ExtraStatusMetrics {
		for key := range OPTIONAL_STATUS_VARS {
			ins.validMetrics[key] = struct{}{}
		}
	}

	if ins.ExtraInnodbMetrics {
		for key := range OPTIONAL_INNODB_VARS {
			ins.validMetrics[key] = struct{}{}
		}
	}
}

func (ins *Instance) Gather(slist *types.SampleList) error {
	tags := map[string]string{"address": ins.Address}

	begun := time.Now()

	// scrape use seconds
	defer func(begun time.Time) {
		use := time.Since(begun).Seconds()
		slist.PushSample(inputName, "scrape_use_seconds", use, tags)
	}(begun)

	db, err := sql.Open("mysql", ins.dsn)
	if err != nil {
		slist.PushSample(inputName, "up", 0, tags)
		log.Println("E! failed to open mysql:", err)
		return err
	}

	defer db.Close()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute)

	if err = db.Ping(); err != nil {
		slist.PushSample(inputName, "up", 0, tags)
		log.Println("E! failed to ping mysql:", err)
		return err
	}

	slist.PushSample(inputName, "up", 1, tags)

	cache := make(map[string]float64)

	ins.gatherGlobalStatus(slist, db, tags, cache)
	ins.gatherGlobalVariables(slist, db, tags, cache)
	ins.gatherEngineInnodbStatus(slist, db, tags, cache)
	ins.gatherEngineInnodbStatusCompute(slist, db, tags, cache)
	ins.gatherBinlog(slist, db, tags)
	ins.gatherProcesslistByState(slist, db, tags)
	ins.gatherProcesslistByUser(slist, db, tags)
	ins.gatherSchemaSize(slist, db, tags)
	ins.gatherTableSize(slist, db, tags, false)
	ins.gatherTableSize(slist, db, tags, true)
	ins.gatherSlaveStatus(slist, db, tags)

	return nil
}
