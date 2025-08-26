package db

import (
	"database/sql"
	"fmt"
	_ "github.com/glebarez/go-sqlite"
	"os"
	"path/filepath"
	"git.sophuwu.com/gophuwu/flags"
)

var DBPATH, INBOX, SAVEPATH string

func ChkErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getHomeBox() string {
	home, err := os.UserHomeDir()
	ChkErr(err)
	if home == "" {
		ChkErr(fmt.Errorf("unable to find $HOME"))
	}
	return filepath.Join(home, ".mailbox")
}

func getConf() {
	mailbox, err := flags.GetStringFlag("mailbox")
	ChkErr(err)
	if mailbox == "" || mailbox == "$HOME/.mailbox" {
		mailbox = getHomeBox()
	}
	if _, err = os.Stat(mailbox); os.IsNotExist(err) {
		os.MkdirAll(mailbox, 0700)
	}

	INBOX = filepath.Join(mailbox, "inbox", "new")
	if _, err = os.Stat(INBOX); os.IsNotExist(err) {
		os.MkdirAll(INBOX, 0700)
	}
	SAVEPATH = filepath.Join(mailbox, "saved")
	if _, err = os.Stat(SAVEPATH); os.IsNotExist(err) {
		os.MkdirAll(SAVEPATH, 0700)
	}
	DBPATH = filepath.Join(mailbox, "mailbox.sqlite")
}

func readRows(rows *sql.Rows) ([]EmailMeta, error) {
	var metas []EmailMeta
	var meta EmailMeta
	var err error
	for rows.Next() {
		err = rows.Scan(&meta.Id, &meta.Subject, &meta.To, &meta.From, &meta.Date)
		if err != nil {
			return metas, err
		}
		metas = append(metas, meta)
	}
	return metas, nil
}

var db *sql.DB

func openDB() error {
	var err error
	db, err = sql.Open("sqlite", DBPATH)
	if err != nil {
		return err
	}
	if db == nil {
		return fmt.Errorf("unknown reason")
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS emails (id TEXT PRIMARY KEY, subject TEXT, toaddr TEXT, fromaddr TEXT, date TEXT)")
	if err != nil {
		return err
	}
	return nil
}

func Open() {
	getConf()
	err := openDB()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening database: ", err)
		os.Exit(1)
	}
	err = parseNewMail()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error parsing new mail: ", err)
		os.Exit(1)
		return
	}
}

func Close() {
	if db != nil {
		if err := db.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "error closing database: ", err)
		}
		db = nil
	}
}

type Query struct {
	rows       []EmailMeta
	page       int
	pageSize   int
	totalRows  int
	totalPages int
	where      string
}

func (r *Query) Page() int {
	return r.page + 1
}
func (r *Query) PageSize() int {
	return r.pageSize
}
func (r *Query) TotalRows() int {
	return r.totalRows
}
func (r *Query) TotalPages() int {
	return r.totalPages + 1
}
func (r *Query) Rows() []EmailMeta {
	return r.rows
}
func (r *Query) Row(i int) (EmailMeta, error) {
	if i < 0 || i >= len(r.rows) {
		return EmailMeta{}, fmt.Errorf("index out of range: %d", i)
	}
	return r.rows[i], nil
}
func (r *Query) whereClause(q *string) {
	if r.where != "" {
		*q += " WHERE " + r.where
	}
}
func (r *Query) executeQuery() error {
	if db == nil {
		return fmt.Errorf("database not opened")
	}
	q := `select count(id) from emails`
	r.whereClause(&q)
	row := db.QueryRow(q)
	if row == nil {
		return fmt.Errorf("query error: no rows returned")
	}
	if err := row.Scan(&r.totalRows); err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	if r.pageSize <= 0 {
		r.pageSize = 5
	} else if r.pageSize > 100 {
		r.pageSize = 100
	}
	if r.totalRows == 0 {
		r.rows = []EmailMeta{}
		r.page = 0
		r.totalPages = 0
		return nil
	}
	r.totalPages = (r.totalRows - 1) / r.pageSize
	if r.page < 0 {
		r.page = 0
	} else if r.page > r.totalPages {
		r.page = r.totalPages
	}
	q = `SELECT * FROM emails`
	r.whereClause(&q)
	q += fmt.Sprintf(" ORDER BY date DESC LIMIT %d OFFSET %d", r.pageSize, r.page*r.pageSize)
	rows, err := db.Query(q)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	r.rows, err = readRows(rows)
	if err != nil {
		return fmt.Errorf("error reading rows: %w", err)
	}
	return nil
}

func (r *Query) Next() error {
	if r.page >= r.totalPages {
		return nil
	}
	r.page++
	return r.executeQuery()
}
func (r *Query) Prev() error {
	if r.page <= 0 {
		return nil
	}
	r.page--
	return r.executeQuery()
}
func (r *Query) SetWhere(where string) error {
	r.where = where
	r.page = 0
	return r.executeQuery()
}
func (r *Query) GetWhere() string {
	return r.where
}
func (r *Query) SetPage(page int) error {
	r.page = page
	return r.executeQuery()
}
func (r *Query) SetPageSize(size int) error {
	if size <= 0 {
		return fmt.Errorf("page size must be greater than zero")
	}
	if size > 100 {
		size = 100
	}
	r.pageSize = size
	return r.executeQuery()
}

func NewQuery(pageSize int) (*Query, error) {
	r := &Query{
		page:     0,
		pageSize: pageSize,
		where:    "",
	}
	if err := r.executeQuery(); err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	return r, nil
}
