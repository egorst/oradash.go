package main

import (
	"database/sql"
	"fmt"
	//    "strings"
	//    "os"
	"github.com/gizak/termui"
	_ "github.com/mattn/go-oci8"
)

func main() {
	err := termui.Init()
	if err != nil {
		fmt.Println(err)
	}
	defer termui.Close()
	db, err := sql.Open("oci8", "tools/catch22")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
	if err = testSelect(db); err != nil {
		fmt.Println(err)
		return
	}
	<-termui.EventCh()
}

func testSelect(db *sql.DB) error {
	rows, err := db.Query(`SELECT * FROM (
    SELECT /*+ LEADING(a) USE_HASH(u) */
      COUNT(*) totalseconds
      , ROUND((COUNT(*) / 300), 1) AAS
      , LPAD(ROUND(RATIO_TO_REPORT(COUNT(*)) OVER () * 100)||'%',5,' ') percent
      , session_id
      , TO_CHAR(MIN(sample_time), 'YYYY-MM-DD HH24:MI:SS') first_seen
      , TO_CHAR(MAX(sample_time), 'YYYY-MM-DD HH24:MI:SS') last_seen
    FROM
        (SELECT
             a.*
           , TO_CHAR(CASE WHEN session_state = 'WAITING' THEN p1 ELSE null END, '0XXXXXXXXXXXXXXX') p1hex
           , TO_CHAR(CASE WHEN session_state = 'WAITING' THEN p2 ELSE null END, '0XXXXXXXXXXXXXXX') p2hex
           , TO_CHAR(CASE WHEN session_state = 'WAITING' THEN p3 ELSE null END, '0XXXXXXXXXXXXXXX') p3hex
        FROM gv$active_session_history a) a
      , dba_users u
    WHERE
        a.user_id = u.user_id (+)
    AND sample_time BETWEEN systimestamp-5/1440 AND systimestamp
    GROUP BY session_id
    ORDER BY TotalSeconds DESC, session_id
)
WHERE ROWNUM <= 5
`)
	if err != nil {
		panic(err)
		return err
	}
	defer rows.Close()
	var got string
	for rows.Next() {
		var totalseconds string
		var aas int
		var percent string
		var session_id string
		var first_seen string
		var last_seen string
		if err = rows.Scan(&totalseconds, &aas, &percent, &session_id, &first_seen, &last_seen); err != nil {
			return err
		}
		got += fmt.Sprintf("%10s\t%6d\t%10s\t%10s\t%14s\t%14s\n", totalseconds, aas, percent, session_id, first_seen, last_seen)
	}
	par := termui.NewPar(got)
	par.Height = 4
	par.Border.Label = "Top 5 Sessions"
	termui.Body.AddRows(termui.NewRow(termui.NewCol(12, 0, par)))
	termui.Body.Align()
	termui.Render(termui.Body)
	//fmt.Println("connected to ", got)
	return nil
}
