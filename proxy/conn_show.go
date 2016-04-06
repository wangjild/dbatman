package proxy

import (
	//	. "github.com/bytedance/dbatman/database/mysql"
	//	"github.com/bytedance/dbatman/hack"
	"github.com/bytedance/dbatman/parser"
)

func (c *Session) handleShow(strsql string, stmt parser.IShow) error {
	var err error

	switch stmt.(type) {
	case *parser.ShowDatabases:
		// TODO
		//		err = c.handleShowDatabases()
	default:
		err = c.handleQuery(stmt, strsql)
	}

	return err

}

/*
func (session *Session) handleShowDatabases() error {
	dbs := make([]interface{}, 0, 1)
	dbs[0] = session.user.DBName

	if r, err := session.buildSimpleShowResultset(dbs, "Database"); err != nil {
		return err
	} else {
		return session.WriteRows(session.status, r)
	}
}

func (session *Session) buildSimpleShowResultset(values []interface{}, name string) (*MySQLResult, error) {

	r := new(MySQLResult)

	field := &MySQLField{
		Name:      hack.Slice(name),
		Charset:   uint16(session.collation),
		FieldType: FieldTypeVarString,
	}
	fields := []*MySQLField{field}

	var row []byte
	var err error

	for _, value := range values {
		row, err = formatValue(value)
		if err != nil {
			return nil, err
		}
		r.RowDatas = append(r.RowDatas, PutLengthEncodedString(row))
	}

	return r, nil
}
*/
