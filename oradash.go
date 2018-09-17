package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
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

type instanceMetrics struct {
	iname string
	mtime string
	//
	cpuutil  float32 // Host CPU Utilization (%)
	cpuratio float32 // Database CPU Time Ratio
	aas      float32 // Average Active Sessions
	execs    float32 // Executions Per Sec
	calls    float32 // User Calls Per Sec
	tnxs     float32 // User Transaction Per Sec
	lios     float32 // Logical Reads Per Sec
	phyrd    float32 // Physical Reads Per Sec
	phywr    float32 // Physical Writes Per Sec
	blkgets  float32 // DB Block Gets Per Sec
	blkchng  float32 // DB Block Changes Per Sec
	redomb   float32 // Redo Generated Per Sec
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

type F struct {
	x int
	y int
	w int
	h int
}

var stat1, stat2 map[string]int64
var Cls = "\x1b[2J"

func xy(x int, y int) string {
	return fmt.Sprintf("\x1b[%d;%dH", y, x)
}

func fg(c int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", c)
}

func bg(c int) string {
	return fmt.Sprintf("\x1b[48;5;%dm", c)
}

func puts(s string, x int, y int, f int, b int) {
	fmt.Print(xy(x, y))
	fmt.Print(fg(f), bg(b))
	fmt.Print(s)
}

//var borderLabelFg = c216(0xee, 0xbb, 0x44)
//var sysStatementFg = c216(8, 8, 8)

func printTemplate(S map[string]F) {

	//	tmplt := "┌ \x1b[38;5;230mINSTANCE SUMMARY\x1b[38;5;252m ───────────────────────────────────────────────────────────────────────────────────────────┐\n" +
	//		"│  Instance:                │ Execs/s:          │ sParse/s:         │ LIOs/s:          │ Read MB/s:           │\n" +
	//		"│  Cur Time:                │ Calls/s:          │ hParse/s:         │ PhyRD/s:         │ Writ MB/s:           │\n" +
	//		"│  Sessions a/b:            │ Commits:          │ ccHits/s:         │ PhyWR/s:         │ Redo MB/s:           │\n" +
	//		"└─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘\n" +
	tmplt := "┌ \x1b[38;5;230mINSTANCE METRICS\x1b[38;5;252m ────────────────────────────────────────────────────────────────────────┐\n" +
		"│ CPU Util:                 │ Execs/s:          │ LRDs/s:          │ Blk Gets/s:           │\n" +
		"│ DB CPU Time Ratio:        │ Calls/s:          │ PhyRD/s:         │ Blk Chng/s:           │\n" +
		"│ AvgActive Sessions:       │ Tnxs/s:           │ PhyWR/s:         │ Redo MB/s:            │\n" +
		"└──────────────────────────────────────────────────────────────────────────────────────────┘\n" +
		"┌ \x1b[38;5;230mTOP SQL_ID (child#)\x1b[38;5;252m ────────┬ \x1b[38;5;230mTOP SESSIONS\x1b[38;5;252m ──────┐┌ \x1b[38;5;230mTOP WAITS\x1b[38;5;252m ──────────────────────────────┬ \x1b[38;5;230mWAIT CLASS\x1b[38;5;252m ───┐\n" +
		"│                             │                    ││                                         │               │\n" +
		"│                             │                    ││                                         │               │\n" +
		"│                             │                    ││                                         │               │\n" +
		"│                             │                    ││                                         │               │\n" +
		"│                             │                    ││                                         │               │\n" +
		"└─────────────────────────────┴────────────────────┘└─────────────────────────────────────────┴───────────────┘\n" +
		"┌ \x1b[38;5;230mSQL_ID\x1b[38;5;252m ───────┬ \x1b[38;5;230mPLAN_HV\x1b[38;5;252m ────┬ \x1b[38;5;230mSQL_TEXT\x1b[38;5;252m ─────────────────────────────────────────────────────────────────────┐\n" +
		"│               │             │                                                                               │\n" +
		"│               │             │                                                                               │\n" +
		"│               │             │                                                                               │\n" +
		"│               │             │                                                                               │\n" +
		"│               │             │                                                                               │\n" +
		"│               │             │                                                                               │\n" +
		"│               │             │                                                                               │\n" +
		"│               │             │                                                                               │\n" +
		"└───────────────┴─────────────┴───────────────────────────────────────────────────────────────────────────────┘\n"
	fmt.Print(Cls, xy(0, 0), fg(252), bg(235)) // c216(0xff, 0xff, 0xaf)), bg(234))
	for _, l := range strings.Split(tmplt, "\n") {
		fmt.Println(l)
	}
	fmt.Print(xy(1, 23))
}

func printF(S map[string]F, fn string, v string) {
	if f, ok := S[fn]; ok {
		fmt.Print(xy(f.x+f.w-len(v), f.y), v)
	}
}

/*
func curset(i byte) error {
	if C.curs_set(C.int(i)) == C.ERR {
		return errors.New("Failed to set")
	}
	return nil
}
*/

func main() {
	S := make(map[string]F)
	S["cpuutil"] = F{13, 2, 15, 1}
	S["cpuratio"] = F{13, 3, 15, 1}
	S["aas"] = F{23, 4, 5, 1}
	S["execs"] = F{39, 2, 9, 1}
	S["calls"] = F{39, 3, 9, 1}
	S["tnxs"] = F{39, 4, 9, 1}
	S["lios"] = F{59, 2, 8, 1}
	S["phyrd"] = F{60, 3, 7, 1}
	S["phywr"] = F{60, 4, 7, 1}
	//S["lios"] = F{78, 2, 8, 1}
	//S["phyrd"] = F{78, 3, 7, 1}
	//S["phywr"] = F{78, 4, 7, 1}
	S["blkgets"] = F{79, 2, 12, 1}
	S["blkchng"] = F{79, 3, 12, 1}
	S["redomb"] = F{78, 4, 13, 1}
	S["topsqlids"] = F{4, 7, 24, 5}
	S["topsids"] = F{33, 7, 18, 5}
	S["events"] = F{54, 7, 38, 5}
	S["waitclasses"] = F{97, 7, 14, 5}
	S["sqlid"] = F{3, 14, 13, 5}
	S["phv"] = F{19, 14, 11, 5}
	S["sqltext"] = F{33, 14, 78, 5}

	flag.Parse()

	/*
		lf, err := os.OpenFile("oradash.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer lf.Close()
		logger := log.New(lf, "", log.Ltime)
		logger.Printf("starting oradash\n")
	*/

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

	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// do not display entered characters on the screen
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	// restore the echoing state when exiting
	defer func() {
		exec.Command("stty", "-F", "/dev/tty", "-cbreak").Run()
		exec.Command("stty", "-F", "/dev/tty", "echo").Run()
		fmt.Print("\x1b[?25h") // show cursor
	}()

	// first run
	im, err := getMetrics(db)
	if err != nil {
		panic(err)
	}

	printTemplate(S)
	printMetrics(im, S)

	sqlidrows, err := ashTopSqlids(db)
	if err != nil {
		panic(err)
	}
	printTopSqlids(sqlidrows, S)

	sids, err := ashTopSids(db)
	if err != nil {
		panic(err)
	}
	printTopSids(sids, S)

	events, err := ashTopEvents(db)
	if err != nil {
		panic(err)
	}
	printTopEvents(events, S)

	sqls, err := getSqls(db, sqlidrows)
	if err != nil {
		panic(err)
	}
	printSqls(sqls, S)

	var b []byte = make([]byte, 1)

	quit := make(chan struct{})
	go func() {
		for {
			os.Stdin.Read(b)
			if b[0] == 0x1b {
				close(quit)
			}
			//fmt.Println("I got the byte", b, "("+string(b)+")")
		}
	}()

	//var cnt = 0

	fmt.Print(xy(0, 23))
	fmt.Print("\x1b[?25l") // turn off cursor

loop:
	for {
		select {
		case <-quit:
			break loop
		case <-time.After(time.Millisecond * 10000):
			im, err := getMetrics(db)
			if err != nil {
				panic(err)
			}
			printMetrics(im, S)

			sqlidrows, err := ashTopSqlids(db)
			if err != nil {
				panic(err)
			}
			printTopSqlids(sqlidrows, S)

			sids, err := ashTopSids(db)
			if err != nil {
				panic(err)
			}
			printTopSids(sids, S)

			events, err := ashTopEvents(db)
			if err != nil {
				panic(err)
			}
			printTopEvents(events, S)

			sqls, err := getSqls(db, sqlidrows)
			if err != nil {
				panic(err)
			}
			printSqls(sqls, S)

			fmt.Print(xy(0, 23))
			fmt.Print("\x1b[?25l") // turn off cursor
		}
	}

	//ashpoll(db)

	//is := getInstanceSummary(db)

	//sids := ashTopSids(db)

	//events, wait_classes := ashTopEvents(db)

}

func printTopSqlids(sqlidrows []SqlidRow, S map[string]F) {
	sF, _ := S["topsqlids"]
	for i, sqlid := range sqlidrows {
		val := fmt.Sprintf("%3d%% | %18s", sqlid.Seconds*100/300, fmt.Sprintf("%s (%d)", sqlid.Sql_id.String, sqlid.Sql_child_number.Int64))
		fmt.Print(xy(sF.x+sF.w-len(val), sF.y+i), val)
	}
	for i := len(sqlidrows); i < 5; i++ {
		fmt.Print(xy(sF.x, sF.y+i), "                        ")
	}
}

func printTopSids(sids []SessionRow, S map[string]F) {
	sF, _ := S["topsids"]
	for i, sid := range sids {
		val := fmt.Sprintf("%3d%% | %11s", sid.Seconds*100/300, fmt.Sprintf("%s,%s", sid.Sid.String, sid.Serial.String))
		fmt.Print(xy(sF.x+sF.w-len(val), sF.y+i), val)
	}
	for i := len(sids); i < 5; i++ {
		fmt.Print(xy(sF.x, sF.y+i), "                  ")
	}
}

func printTopEvents(events []EventRow, S map[string]F) {
	F1, _ := S["events"]
	F2, _ := S["waitclasses"]
	for i, ev := range events {
		val1 := fmt.Sprintf("%3d%% | %-30s", ev.Seconds*100/300, ev.Event.String)
		val2 := fmt.Sprintf("%-14s", ev.Wait_class.String)
		fmt.Print(xy(F1.x+F1.w-len(val1), F1.y+i), val1)
		fmt.Print(xy(F2.x+F2.w-len(val2), F2.y+i), val2)
	}
	for i := len(events); i < 5; i++ {
		fmt.Print(xy(F1.x, F1.y+i), "                                      ")
		fmt.Print(xy(F2.x, F2.y+i), "              ")
	}

}

func printSqls(sqls []SqltextRow, S map[string]F) {
	sF1, _ := S["sqlid"]
	sF2, _ := S["phv"]
	sF3, _ := S["sqltext"]
	for i, sql := range sqls {
		val1 := fmt.Sprintf("%-12s", sql.Sql_id)
		val2 := fmt.Sprintf("%11d", sql.Plan.Int64)
		val3 := fmt.Sprintf("%-78s", sql.Sqltext)
		fmt.Print(xy(sF1.x+sF1.w-len(val1), sF1.y+i), val1)
		fmt.Print(xy(sF2.x+sF2.w-len(val2), sF2.y+i), val2)
		fmt.Print(xy(sF3.x+sF3.w-len(val3), sF3.y+i), val3)
	}
	for i := len(sqls); i < 5; i++ {
		fmt.Print(xy(sF1.x, sF1.y+i), fmt.Sprintf("%13s", " ")) // "!********0*!")
		fmt.Print(xy(sF2.x, sF2.y+i), fmt.Sprintf("%11s", " ")) //"!********0**!")
		fmt.Print(xy(sF3.x, sF3.y+i), fmt.Sprintf("%78s", " ")) // "!********1*********2*********3*********4*********5*********6*********7*!")
	}
}

func printMetrics(im instanceMetrics, S map[string]F) {
	fmt.Print(xy(20, 1), fg(230), "[ ", im.iname, " ", im.mtime, " ] ", fg(252)) // c216(0xff, 0xff, 0xaf)), bg(234))
	printF(S, "cpuutil", fmt.Sprintf("%3.0f%%", im.cpuutil))
	printF(S, "cpuratio", fmt.Sprintf("%3.0f%%", im.cpuratio))
	printF(S, "aas", fmt.Sprintf("%5.1f", im.aas))
	printF(S, "execs", fmt.Sprintf("%9.0f", im.execs))
	printF(S, "calls", fmt.Sprintf("%9.0f", im.calls))
	printF(S, "tnxs", fmt.Sprintf("%9.0f", im.tnxs))
	printF(S, "lios", fmt.Sprintf("%7.0f", im.lios))
	printF(S, "phyrd", fmt.Sprintf("%7.0f", im.phyrd))
	printF(S, "phywr", fmt.Sprintf("%7.0f", im.phywr))
	printF(S, "blkgets", fmt.Sprintf("%7.0f", im.blkgets))
	printF(S, "blkchng", fmt.Sprintf("%7.0f", im.blkchng))
	printF(S, "redomb", fmt.Sprintf("%7.0f", im.redomb/1024/1024))
}

func logerr(e string) {
	f, err := os.OpenFile("odash.log", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(e + "\n"); err != nil {
		panic(err)
	}
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

func ashTopSqlids(db *sqlx.DB) ([]SqlidRow, error) {
	var sqlidRows []SqlidRow
	var r SqlidRow
	rows, err := db.Queryx(`select * from 
	(select sql_id, sql_child_number, count(*) seconds 
	 from v$active_session_history 
	 where sample_time >= sysdate-5/1440 group by sql_id,sql_child_number order by 3 desc
	)
	where rownum < 6`)
	if err != nil && err != sql.ErrNoRows {
		return sqlidRows, err
	}
	for rows.Next() {
		rows.StructScan(&r)
		if r.Sql_id.Valid && r.Sql_id.String != "" {
			sqlidRows = append(sqlidRows, r)
		}
	}
	return sqlidRows, nil
}

type SessionRow struct {
	Sid     sql.NullString `db:"SESSION_ID"`
	Serial  sql.NullString `db:"SESSION_SERIAL#"`
	Seconds int            `db:"SECONDS"`
}

func ashTopSids(db *sqlx.DB) ([]SessionRow, error) {
	var res []SessionRow
	var r SessionRow
	rows, err := db.Queryx(`select * from 
	(select session_id, session_serial#, count(*) seconds 
	 from v$active_session_history 
	 where sample_time >= sysdate-5/1440 group by session_id,session_serial# order by 3 desc
	)
	where rownum < 6`)
	if err != nil && err != sql.ErrNoRows {
		return res, err
	}
	for rows.Next() {
		rows.StructScan(&r)
		if r.Sid.Valid && r.Sid.String != "" {
			res = append(res, r)
		}
	}
	return res, nil
}

type EventRow struct {
	Event      sql.NullString `db:"EVENT"`
	Wait_class sql.NullString `db:"WAIT_CLASS"`
	Seconds    int            `db:"SECONDS"`
}

func ashTopEvents(db *sqlx.DB) ([]EventRow, error) {
	var res []EventRow
	var r EventRow
	rows, err := db.Queryx(`select * from 
	(select decode(session_state,'ON CPU',session_state,event) event, wait_class , count(*) seconds
	 from v$active_session_history
	 where sample_time >= sysdate-5/1440
	 group by decode(session_state,'ON CPU',session_state,event), wait_class order by 3 desc
	)
where rownum < 6`)
	if err != nil && err != sql.ErrNoRows {
		return res, err
	}
	for rows.Next() {
		rows.StructScan(&r)
		if r.Event.Valid {
			res = append(res, r)
		}
	}
	return res, nil
}

type SqltextRow struct {
	Sql_id          string        `db:"SQL_ID"`
	Plan            sql.NullInt64 `db:"PLAN_HASH_VALUE"`
	Sqltext         string        `db:"SQL_TEXT"`
	Parsing_User_Id sql.NullInt64 `db:"PARSING_USER_ID"`
}

func getSqls(db *sqlx.DB, sql_ids []SqlidRow) ([]SqltextRow, error) {
	var res []SqltextRow
	var r SqltextRow
	for _, sqlid := range sql_ids {
		if sqlid.Sql_id.Valid {
			err := db.QueryRowx("select distinct sql_id, plan_hash_value, sql_text, parsing_user_id from v$sql where sql_id = :1 and child_number = :2", sqlid.Sql_id.String, sqlid.Sql_child_number.Int64).StructScan(&r)
			if err != nil {
				// just hide this error from caller
				return res, nil
			}

			r.Sqltext = trimsql(r.Sqltext)
			if len(r.Sqltext) > 76 {
				r.Sqltext = r.Sqltext[:76] + ".."
			}
			res = append(res, r)
		}
	}
	return res, nil
}

func getMetrics(db *sqlx.DB) (instanceMetrics, error) {
	var im instanceMetrics
	var iname string

	err := db.Get(&iname, "select instance_name from v$instance")
	if err != nil {
		im.iname = "?"
	} else {
		im.iname = iname
	}

	im.mtime = time.Now().Format("15:04:05")
	rows, err := db.Query(`select metric_name, value
from v$sysmetric 
where group_id=3 and metric_name in (
  'Average Active Sessions', 'Host CPU Utilization (%)', 'Database CPU Time Ratio',
  'Executions Per Sec', 'User Calls Per Sec', 'User Transaction Per Sec',
  'Logical Reads Per Sec', 'Physical Reads Per Sec', 'Physical Writes Per Sec',
  'DB Block Gets Per Sec', 'DB Block Changes Per Sec', 'Redo Generated Per Sec'
)`)
	if err != nil {
		return im, err
	}
	for rows.Next() {
		var (
			nam string
			val float32
		)
		err = rows.Scan(&nam, &val)
		if err != nil {
			return im, err
		}
		switch nam {
		case "Host CPU Utilization (%)":
			im.cpuutil = val
		case "Database CPU Time Ratio":
			im.cpuratio = val
		case "Average Active Sessions":
			im.aas = val
		case "Executions Per Sec":
			im.execs = val
		case "User Calls Per Sec":
			im.calls = val
		case "User Transaction Per Sec":
			im.tnxs = val
		case "Logical Reads Per Sec":
			im.lios = val
		case "Physical Reads Per Sec":
			im.phyrd = val
		case "Physical Writes Per Sec":
			im.phywr = val
		case "DB Block Gets Per Sec":
			im.blkgets = val
		case "DB Block Changes Per Sec":
			im.blkchng = val
		case "Redo Generated Per Sec":
			im.redomb = val
		}
	}
	return im, nil
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
	if string(res[0]) == " " {
		res = res[1:]
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

func c216(r, g, b int) int {
	if r == g && g == b && r != 0 && r != 255 {
		return 16 + 216 + r
	} else {
		return 16 + conv216(r)*36 + conv216(g)*6 + conv216(b) + 1
	}
}

/*
func c216(r, g, b int) ui.Attribute {
	if r == g && g == b && r != 0 && r != 255 {
		return (ui.Attribute)(16 + 216 + r)
	} else {
		return (ui.Attribute)(16 + conv216(r)*36 + conv216(g)*6 + conv216(b) + 1) // +1 is a temporary dirty hack
	}
}
*/
