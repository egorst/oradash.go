package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	ui "github.com/gizak/termui"
	"github.com/jmoiron/sqlx"
	termbox "github.com/nsf/termbox-go"
	_ "gopkg.in/goracle.v2"
)

type instanceSummary struct {
	iname    string
	ctime    string
	sessions string
	execs    float32
	calls    float32
	commits  float32
	sparse   float32
	hparse   float32
	cchits   float32
	lios     float32
	phyrd    float32
	phywr    float32
	readmb   float32
	writemb  float32
	redomb   float32
}

type SessionRecord struct {
	Ashtime          int            `db:"ASHTIME"`
	Sid              int            `db:"SID"`
	Serial           int            `db:"SERIAL#"`
	Username         sql.NullString `db:"USERNAME"`
	Machine          sql.NullString `db:"MACHINE"`
	Program          sql.NullString `db:"PROGRAM"`
	Sql_id           sql.NullString `db:"SQL_ID"`
	Sql_child_number sql.NullInt64  `db:"SQL_CHILD_NUMBER"`
	Blocking_session sql.NullString `db:"BLOCKING_SESSION"`
	Event            sql.NullString `db:"EVENT"`
	Wait_class       sql.NullString `db:"WAIT_CLASS"`
	Wait_time        sql.NullString `db:"WAIT_TIME"`
	Seconds_in_wait  sql.NullString `db:"SECONDS_IN_WAIT"`
}

var stat1, stat2 map[string]int64

var borderLabelFg = c216(0xee, 0xbb, 0x44)

func main() {

	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("Usage:\n$ " + os.Args[0] + " <connect_string>")
		os.Exit(1)
	}
	var constring string
	constring = os.Args[1]
	db, err := sqlx.Open("goracle", constring)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	stat2, err = db_get_stats(db)
	if err != nil {
		fmt.Println("ERR:", err)
		return
	}

	//ashpoll(db)

	err = ui.Init()
	termbox.SetOutputMode(termbox.Output256)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	is := getInstanceSummary(db)

	instSum := ui.NewTable()
	instSum.BorderLabel = " INSTANCE SUMMARY "
	instSum.BorderLabelFg = borderLabelFg
	instSum.Separator = false
	instSum.Height = 5
	instSum.Width = 111
	//instSum.CellWidth = []int{24, 17, 17, 16, 19} // 110
	instSum.X = 0
	instSum.Y = 0
	//instSum.SetSize()
	instSum.Rows = [][]string{
		/*return fmt.Sprintf(" Instance: %-14s  | Execs/s: %8.1f | sParse/s: %7.1f | LIOs/s: %8.1f | Read MB/s: %8.1f\n", is.iname, is.execs, is.sparse, is.lios, is.readmb) +
		fmt.Sprintf(" Cur Time: %-14s | Calls/s: %8.1f | hParse/s: %7.1f | PhyRD/s: %7.1f | Writ MB/s: %8.1f\n", ctime, is.calls, is.hparse, is.phyrd, is.writemb) +
		fmt.Sprintf(" History: %-16s | Commits: %8.1f | ccHits/s: %7.1f | PhyWR/s: %7.1f | Redo MB/s: %8.1f", "", is.commits, is.cchits, is.phywr, is.redomb) */
		[]string{
			fmt.Sprintf("Instance: %14s", is.iname),
			fmt.Sprintf("Execs/s: %8.1f", is.execs),
			fmt.Sprintf("sParse/s: %7.1f", is.sparse),
			fmt.Sprintf("LIOs/s: %8.1f", is.lios),
			fmt.Sprintf("Read MB/s: %8.1f", is.readmb),
		},
		[]string{
			fmt.Sprintf("Cur Time: %14s", is.ctime),
			fmt.Sprintf("Calls/s: %8.1f", is.calls),
			fmt.Sprintf("hParse/s: %7.1f", is.hparse),
			fmt.Sprintf("PhyRD/s: %7.1f", is.phyrd),
			fmt.Sprintf("Writ MB/s: %8.1f", is.writemb),
		},
		[]string{
			fmt.Sprintf("Sessions a/b: %10s", is.sessions),
			fmt.Sprintf("Commits: %8.1f", is.commits),
			fmt.Sprintf("ccHits/s: %7.1f", is.cchits),
			fmt.Sprintf("PhyWR/s: %7.1f", is.phywr),
			fmt.Sprintf("Redo MB/s: %8.1f", is.redomb),
		},
	}
	ui.Render(instSum)

	topSqlId := ui.NewPar("")
	topSqlId.Height = 5
	topSqlId.Width = 31
	topSqlId.X = 0
	topSqlId.Y = 5
	topSqlId.BorderLabel = " TOP SQL_ID (child#) "
	topSqlId.BorderLabelFg = borderLabelFg

	topSess := ui.NewPar("")
	topSess.Height = 5
	topSess.Width = 22
	topSess.X = 30
	topSess.Y = 5
	topSess.BorderLabel = " TOP SESSIONS "
	topSess.BorderLabelFg = borderLabelFg

	topWaits := ui.NewPar("")
	topWaits.Height = 5
	topWaits.Width = 43
	topWaits.X = 52
	topWaits.Y = 5
	topWaits.BorderLabel = " TOP WAITS "
	topWaits.BorderLabelFg = borderLabelFg

	waitClass := ui.NewPar("")
	waitClass.Height = 5
	waitClass.Width = 17
	waitClass.X = 94
	waitClass.Y = 5
	waitClass.BorderLabel = " WAIT CLASS "
	waitClass.BorderLabelFg = borderLabelFg

	sqlId := ui.NewPar("")
	sqlId.Height = 10
	sqlId.Width = 16
	sqlId.X = 0
	sqlId.Y = 10
	sqlId.BorderLabel = " SQL_ID "
	sqlId.BorderLabelFg = borderLabelFg

	planHashValue := ui.NewPar("")
	planHashValue.Height = 10
	planHashValue.Width = 19
	planHashValue.X = 15
	planHashValue.Y = 10
	planHashValue.BorderLabel = " PLAN_HASH_VALUE"
	planHashValue.BorderLabelFg = borderLabelFg

	sqlText := ui.NewPar("")
	sqlText.Height = 10
	sqlText.Width = 78
	sqlText.X = 33
	sqlText.Y = 10
	sqlText.BorderLabel = " SQL_TEXT "
	sqlText.BorderLabelFg = borderLabelFg

	sqlids := ashTopSqlids(db)
	sqlidtext := ""
	for _, s := range sqlids {
		if s.Sql_id.Valid {
			sqlidtext += fmt.Sprintf("\n %3d%% | %10s (%d)", s.Seconds*100/(5*60), s.Sql_id.String, s.Sql_child_number.Int64)
		}
	}
	if len(sqlidtext) > 0 {
		topSqlId.Text = sqlidtext[1:]
	} else {
		topSqlId.Text = ""
	}

	sql_ids, plans, sqltexts := getSqls(db, sqlids)
	sqlId.Text = sql_ids
	planHashValue.Text = plans
	sqlText.Text = sqltexts

	sids := ashTopSids(db)
	topSess.Text = sids

	events, wait_classes := ashTopEvents(db)
	topWaits.Text = events
	waitClass.Text = wait_classes

	ui.Render(instSum, topSqlId, topSess, topWaits, waitClass, sqlId, planHashValue, sqlText)

	ui.Handle("/sys/kbd/q", func(e ui.Event) {
		ui.StopLoop()
	})

	instSum.Handle("/timer/1s", func(e ui.Event) {
		t := e.Data.(ui.EvtTimer)
		if t.Count%5 != 0 {
			return
		}
		is = getInstanceSummary(db)
		instSum.Rows = [][]string{
			/*return fmt.Sprintf(" Instance: %-14s  | Execs/s: %8.1f | sParse/s: %7.1f | LIOs/s: %8.1f | Read MB/s: %8.1f\n", is.iname, is.execs, is.sparse, is.lios, is.readmb) +
			fmt.Sprintf(" Cur Time: %-14s | Calls/s: %8.1f | hParse/s: %7.1f | PhyRD/s: %7.1f | Writ MB/s: %8.1f\n", ctime, is.calls, is.hparse, is.phyrd, is.writemb) +
			fmt.Sprintf(" History: %-16s | Commits: %8.1f | ccHits/s: %7.1f | PhyWR/s: %7.1f | Redo MB/s: %8.1f", "", is.commits, is.cchits, is.phywr, is.redomb) */
			[]string{
				fmt.Sprintf("Instance: %14s", is.iname),
				fmt.Sprintf("Execs/s: %8.1f", is.execs),
				fmt.Sprintf("sParse/s: %7.1f", is.sparse),
				fmt.Sprintf("LIOs/s: %8.1f", is.lios),
				fmt.Sprintf("Read MB/s: %8.1f", is.readmb),
			},
			[]string{
				fmt.Sprintf("Cur Time: %14s", is.ctime),
				fmt.Sprintf("Calls/s: %8.1f", is.calls),
				fmt.Sprintf("hParse/s: %7.1f", is.hparse),
				fmt.Sprintf("PhyRD/s: %7.1f", is.phyrd),
				fmt.Sprintf("Writ MB/s: %8.1f", is.writemb),
			},
			[]string{
				fmt.Sprintf("Sessions a/b: %10s", is.sessions),
				fmt.Sprintf("Commits: %8.1f", is.commits),
				fmt.Sprintf("ccHits/s: %7.1f", is.cchits),
				fmt.Sprintf("PhyWR/s: %7.1f", is.phywr),
				fmt.Sprintf("Redo MB/s: %8.1f", is.redomb),
			},
		}
		sqlids := ashTopSqlids(db)
		sqlidtext := ""
		for _, s := range sqlids {
			sqlidtext += fmt.Sprintf("\n %3d%% | %10s (%d)", s.Seconds*100/(5*60), s.Sql_id.String, s.Sql_child_number.Int64)
		}
		if len(sqlidtext) > 0 {
			topSqlId.Text = sqlidtext[1:]
		} else {
			topSqlId.Text = ""
		}

		sql_ids, plans, sqltexts := getSqls(db, sqlids)
		sqlId.Text = sql_ids
		planHashValue.Text = plans
		sqlText.Text = sqltexts

		sids := ashTopSids(db)
		topSess.Text = sids

		events, wait_classes := ashTopEvents(db)
		topWaits.Text = events
		waitClass.Text = wait_classes
		ui.Render(instSum, topSqlId, topSess, topWaits, waitClass, sqlId, planHashValue, sqlText)
	})

	ui.Loop()
}

func getInstanceSummary(db *sqlx.DB) instanceSummary {
	var is instanceSummary
	var iname string
	var tdiff float32
	var asess, bsess int64
	stat1 = stat2
	stat2, err := db_get_stats(db)
	if err != nil {
		fmt.Println(err)
	}
	err = db.Get(&iname, "select instance_name from gv$instance")
	if err != nil {
		is.iname = "?"
	} else {
		is.iname = iname
	}
	err = db.Get(&asess, "select count(*) from gv$session where wait_class!='Idle' and sid != sys_context('userenv', 'sid')")
	if err != nil {
		asess = -1
	}
	err = db.Get(&bsess, "select count(*) from gv$session where blocking_session is not null")
	if err != nil {
		bsess = -1
	}
	is.sessions = fmt.Sprintf("%d/%d", asess, bsess)

	tdiff = float32((stat2["timer"] - stat1["timer"])) / 100
	is.execs = float32(stat2["execute count"]-stat1["execute count"]) / tdiff
	is.calls = float32(stat2["user calls"]-stat1["user calls"]) / tdiff
	is.commits = float32(stat2["user commits"]-stat1["user commits"]) / tdiff
	is.sparse = float32(stat2["parse count (total)"]-stat1["parse count (total)"]) / tdiff
	is.hparse = float32(stat2["parse count (hard)"]-stat1["parse count (hard)"]) / tdiff
	is.cchits = float32(stat2["session cursor cache hits"]-stat1["session cursor cache hits"]) / tdiff
	is.lios = float32(stat2["session logical reads"]-stat1["session logical reads"]) / tdiff
	is.phyrd = float32(stat2["physical read total IO requests"]-stat1["physical read total IO requests"]) / tdiff
	is.phywr = float32(stat2["physical write total IO requests"]-stat1["physical write total IO requests"]) / tdiff
	is.readmb = float32(stat2["physical read total bytes"]-stat1["physical read total bytes"]) / tdiff / 1024 / 1024
	is.writemb = float32(stat2["physical write total bytes"]-stat1["physical write total bytes"]) / tdiff / 1024 / 1024
	is.redomb = float32(stat2["redo size"]-stat1["redo size"]) / tdiff
	is.ctime = time.Now().Format("01-02 15:04:05")
	return is
}

func ashpoll(db *sqlx.DB) {
	sessRecs := []SessionRecord{}
	selectVSession := `select 
  dbms_utility.get_time() ashtime,
  sid,
  serial#,
  username,
  machine,
  program,
  sql_id,
  sql_child_number,
  blocking_session,
  case when state = 'WAITING' then event else 'ON CPU' end event,
  case when state = 'WAITING' then wait_class else 'ON CPU' end wait_class,
  wait_time,
  seconds_in_wait
from gv$session
where 
  status = 'ACTIVE'
  and (wait_class != 'Idle' or state != 'WAITING')
  --and sid != sys_context('userenv', 'sid')`
	err := db.Select(&sessRecs, selectVSession)
	if err != nil {
		log.Println("ERR:", err)
	}
}

type SqlidRow struct {
	Sql_id           sql.NullString `db:"SQL_ID"`
	Sql_child_number sql.NullInt64  `db:"SQL_CHILD_NUMBER"`
	Seconds          int            `db:"SECONDS"`
}

func ashTopSqlids(db *sqlx.DB) []SqlidRow {
	var sqlidRows []SqlidRow
	var r SqlidRow
	rows, err := db.Queryx(`select * from 
	(select sql_id, sql_child_number, count(*) seconds 
	 from gv$active_session_history 
	 where sample_time >= sysdate-5/1440 group by sql_id,sql_child_number order by 3 desc
	)
	where rownum < 4`)
	if err != nil {
		log.Println(err)
	}
	for rows.Next() {
		rows.StructScan(&r)
		if r.Sql_id.Valid && r.Sql_id.String != "" {
			sqlidRows = append(sqlidRows, r)
		}
	}
	return sqlidRows
}

type SessionRow struct {
	Sid     sql.NullString `db:"SESSION_ID"`
	Serial  sql.NullString `db:"SESSION_SERIAL#"`
	Seconds int            `db:"SECONDS"`
}

func ashTopSids(db *sqlx.DB) string {
	var r SessionRow
	res := ""
	rows, err := db.Queryx(`select * from 
	(select session_id, session_serial#, count(*) seconds 
	 from gv$active_session_history 
	 where sample_time >= sysdate-5/1440 group by session_id,session_serial# order by 3 desc
	)
	where rownum < 4`)
	if err != nil {
		log.Println(err)
	}
	for rows.Next() {
		rows.StructScan(&r)
		res += fmt.Sprintf("\n %3d%% | %s.%s", r.Seconds*100/(5*60), r.Sid.String, r.Serial.String)
	}
	if len(res) > 0 {
		res = res[1:]
	}
	return res
}

type EventRow struct {
	Event      sql.NullString `db:"EVENT"`
	Wait_class sql.NullString `db:"WAIT_CLASS"`
	Seconds    int            `db:"SECONDS"`
}

func ashTopEvents(db *sqlx.DB) (string, string) {
	var r EventRow
	events := ""
	wait_classes := ""
	rows, err := db.Queryx(`select * from 
	(select decode(session_state,'ON CPU',session_state,event) event, wait_class , count(*) seconds
	 from gv$active_session_history
	 where sample_time >= sysdate-5/1440
	 group by decode(session_state,'ON CPU',session_state,event), wait_class order by 3 desc
	)
where rownum < 4`)
	if err != nil {
		log.Println(err)
	}
	for rows.Next() {
		rows.StructScan(&r)
		if r.Event.Valid {
			events += fmt.Sprintf("\n %3d%% | %s", r.Seconds*100/(5*60), r.Event.String)
			wait_classes += "\n" + r.Wait_class.String
		}
	}
	if len(events) > 0 {
		events = events[1:]
	}
	if len(wait_classes) > 0 {
		wait_classes = wait_classes[1:]
	}
	return events, wait_classes
}

type SqltextRow struct {
	Sql_id  string        `db:"SQL_ID"`
	Plan    sql.NullInt64 `db:"PLAN_HASH_VALUE"`
	Sqltext string        `db:"SQL_TEXT"`
}

func getSqls(db *sqlx.DB, sql_ids []SqlidRow) (string, string, string) {
	var r SqltextRow
	sqlids := ""
	plans := ""
	sqltexts := ""
	for _, sqlid := range sql_ids {
		if sqlid.Sql_id.Valid {
			row := db.QueryRowx("select distinct sql_id, plan_hash_value, sql_text\nfrom gv$sql\nwhere sql_id = :1 and child_number= :2", sqlid.Sql_id.String, sqlid.Sql_child_number.Int64)
			row.StructScan(&r)
			sqlids += fmt.Sprintf("\n%-6s\n", r.Sql_id)
			plans += fmt.Sprintf("\n%-8d\n", r.Plan.Int64)
			r.Sqltext = trimsql(r.Sqltext)
			if len(r.Sqltext) > 153 {
				r.Sqltext = r.Sqltext[:152]
			}
			sqltexts += fmt.Sprintf("\n%-152s", strings.TrimSpace(r.Sqltext))
		}
	}
	if len(sqlids) > 0 {
		return sqlids[1:], plans[1:], sqltexts[1:]
	} else {
		return "", "", ""
	}
}

func db_get_stats(db *sqlx.DB) (map[string]int64, error) {
	var res = make(map[string]int64)
	rows, err := db.Query(`
select sn.name, ss.value
from   v$statname sn, v$sysstat  ss
where  sn.statistic# = ss.statistic#
and ss.name in ('execute count', 'parse count (hard)', 'parse count (total)',
                'physical read total IO requests', 'physical read total bytes',
				'physical write total IO requests', 'physical write total bytes',
				'redo size', 'redo writes', 'session cursor cache hits',
				'session logical reads', 'user calls', 'user commits')
union all
select 'timer', hsecs from v$timer
`)
	if err != nil {
		log.Println("got an error in Query")
		return nil, err
	}
	for rows.Next() {
		var (
			nam string
			val int64
		)
		err = rows.Scan(&nam, &val)
		if err != nil {
			return nil, err
		}
		res[nam] = val
	}
	return res, nil
}

func trimsql(s string) string {
	res := ""
	instr := false
	prevspace := false
	for _, c := range strings.Split(s, "") {
		if c == "'" {
			instr = !instr
		}
		if !instr {
			if c == " " {
				if prevspace {
					continue
				}
				prevspace = true
			} else {
				prevspace = false
			}
		}
		res += c
	}
	return res
}

func conv216(i int) int {
	cnt := 0
	for {
		if i <= 0 {
			return cnt
		} else if i <= 95 {
			i -= 95
			cnt += 1
		} else {
			i -= 40
			cnt += 1
		}
	}
	return cnt
}

func c216(r, g, b int) ui.Attribute {
	return (ui.Attribute)(16 + conv216(r)*36 + conv216(g)*6 + conv216(b) + 1) // +1 is a temporary dirty hack
}
