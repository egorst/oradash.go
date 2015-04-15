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
	rows, err := db.Query("select name from gv$database")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var namedb string
		if err = rows.Scan(&namedb); err != nil {
			return err
		}
		got := fmt.Sprintf("'%s'", namedb)
		par := termui.NewPar("connected to " + got)
		par.Height = 4
		par.Border.Label = "DB Summary"
		termui.Body.AddRows(termui.NewRow(termui.NewCol(6, 0, par)))
		termui.Body.Align()
		termui.Render(termui.Body)
		//fmt.Println("connected to ", got)
	}
	return nil
}
