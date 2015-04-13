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
    db, err := sql.Open("oci8","tools/catch22");
    if err != nil {
        fmt.Println(err)
        return
    }
    defer db.Close()
    if err = testSelect(db); err != nil {
        fmt.Println(err)
        return
    }
}

func testSelect(db *sql.DB) error {
    rows, err := db.Query("select 1,'bar' from dual")
    if err != nil {
        return err
    }
    defer rows.Close()
    db.Exec("create table zz (f1 number)")
    return nil
}
