package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

const (
	dbdriver           string = "sqlite3"
	dbpath             string = "/var/db/objects.db"
	createTableQuery   string = "CREATE TABLE IF NOT EXISTS objects (key INTEGER PRIMARY KEY AUTOINCREMENT, value TEXT NULL)"
	selectObjectsQuery string = "SELECT key, value FROM objects"
	selectObjectQuery  string = "SELECT value FROM objects WHERE key=?"
	insertQuery        string = "INSERT INTO objects (value) VALUES (?)"

	MAX_VALUE_SIZE int = 64
)

type Object struct {
	Key   uint64 `json:"key"`
	Value string `json:"value"`
}

type Error struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   error  `json:"error,omitempty"`
}

func writeError(w http.ResponseWriter, statusCode int, msg string, err error) {
	log.Error().Err(err).Msg(msg)
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(Error{Status: statusCode, Message: msg, Error: err})
	if err != nil {
		log.Error().Err(err)

	}
}

func writeJSON(w http.ResponseWriter, o interface{}) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(o)
	if err != nil {
		log.Error().Err(err)
	}
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(createTableQuery)
	return err
}

func getObjects(db *sql.DB) http.HandlerFunc {
	// TODO implement pagination
	return func(w http.ResponseWriter, req *http.Request) {
		// get all objects from DB
		rows, err := db.Query(selectObjectsQuery)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to select objects from database", err)
			return
		}
		defer rows.Close()

		objects := []Object{}
		for rows.Next() {
			o := Object{}
			err := rows.Scan(&o.Key, &o.Value)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to get objects from database", err)
				return
			}
			objects = append(objects, o)
		}
		hlog.FromRequest(req).Info().Int("count", len(objects)).Msg("got objects from database")
		writeJSON(w, objects)
	}
}

func getObject(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)

		// routing match should be formatted as int but could error if too big
		key, err := strconv.ParseUint(vars["key"], 10, 64)
		if err != nil {
			log.Error().Err(err)
		}

		// get object from DB
		row := db.QueryRow(selectObjectQuery, key)

		// check if row can be scanned
		if err := row.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to select object from database", err)
			return
		}

		o := Object{
			Key: key,
		}

		err = row.Scan(&o.Value)
		if err != nil {
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "object not found in database", err)
			} else {
				writeError(w, http.StatusInternalServerError, "failed to get object from database", err)
			}
		} else {
			hlog.FromRequest(req).Info().Interface("object", o).Msg("got object from database")
			writeJSON(w, o)
		}
	}
}

func postObject(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// get JSON object from request body
		o := Object{}
		err := json.NewDecoder(req.Body).Decode(&o)
		if err != nil {
			log.Error().Err(err)
		}

		if len(o.Value) > MAX_VALUE_SIZE {
			writeError(w, http.StatusRequestEntityTooLarge, "payload max size is "+strconv.Itoa(MAX_VALUE_SIZE), nil)
			return
		}

		// insert object in to DB
		res, err := db.Exec(insertQuery, o.Value)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "couldn't insert object into database", err)
			return
		}
		id, err := res.LastInsertId()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "couldn't read last insertion from database", err)
			return
		}
		o.Key = uint64(id)
		hlog.FromRequest(req).Info().Interface("object", o).Msg("post objects to database")
		writeJSON(w, o)
		log.Info().Str("value", o.Value).Uint64("key", o.Key).Msg("object inserted into database")
	}
}

func ping(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := db.Ping()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to ping database", err)
		} else {
			writeJSON(w, "PONG")
		}
	}
}

func main() {
	db, err := sql.Open(dbdriver, dbpath)
	if err != nil {
		log.Fatal().Str("database", dbpath).Err(err).Msg("failed to open")
	}
	log.Info().Str("database", dbpath).Msg("database opened")

	err = createTable(db)
	if err != nil {
		log.Fatal().Str("database", dbpath).Err(err).Msg("failed to create table")
	}

	r := mux.NewRouter()

	// populate logger with custom fields
	r.Use(hlog.NewHandler(log.Logger))
	r.Use(hlog.RemoteAddrHandler("ip"))
	r.Use(hlog.MethodHandler("method"))
	r.Use(hlog.URLHandler("url"))

	r.HandleFunc("/object", postObject(db)).Methods("POST")
	r.HandleFunc("/object/{key:[0-9]+}", getObject(db)).Methods("GET")
	r.HandleFunc("/objects", getObjects(db)).Methods("GET")
	r.HandleFunc("/ping", ping(db)).Methods("GET")

	log.Info().Uint("port", 8080).Msg("starting HTTP server")
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Error().Err(err).Msg("cannot listen and serve")
	}
}
