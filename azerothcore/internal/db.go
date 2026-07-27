package internal

import (
	"fmt"
	"strings"
)

type dbInfo struct {
	host   string
	port   string
	user   string
	pass   string
	dbname string
}

func parseDBInfo(s string) (*dbInfo, error) {
	parts := strings.SplitN(s, ";", 5)
	if len(parts) != 5 {
		return nil, fmt.Errorf("expected host;port;user;pass;dbname, got %q", s)
	}
	return &dbInfo{
		host:   parts[0],
		port:   parts[1],
		user:   parts[2],
		pass:   parts[3],
		dbname: parts[4],
	}, nil
}

func (d *dbInfo) dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", d.user, d.pass, d.host, d.port, d.dbname)
}

func (d *dbInfo) adminDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/", d.user, d.pass, d.host, d.port)
}
