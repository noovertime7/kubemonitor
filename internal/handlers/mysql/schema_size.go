package mysql

import (
	"database/sql"
	"github.com/noovertime7/kubemonitor/pkg/tagx"
	"github.com/noovertime7/kubemonitor/pkg/types"
	"github.com/sirupsen/logrus"
)

func (ins *Instance) gatherSchemaSize(slist *types.SampleList, db *sql.DB, globalTags map[string]string) {
	if !ins.GatherSchemaSize {
		return
	}

	rows, err := db.Query(SQL_QUERY_SCHEMA_SIZE)
	if err != nil {
		logrus.Error("E! failed to get schema size:", err)
		return
	}

	defer rows.Close()

	labels := tagx.Copy(globalTags)

	for rows.Next() {
		var schema string
		var size int64

		err = rows.Scan(&schema, &size)
		if err != nil {
			logrus.Error("E! failed to scan rows:", err)
			return
		}

		slist.PushFront(types.NewSample(inputName, "schema_size_bytes", size, labels, map[string]string{"schema": schema}))
	}
}
